
package gosafe

import (
	"fmt"
	"go/parser"
	"go/token"
	"go/ast"
	"bytes"
	"os/exec"
	"io"
	"os"
	"time"
)

type visitor func(ast.Node)
func (self visitor) Visit(node ast.Node) ast.Visitor {
	self(node)
	return self
}

type Error string
func (self Error) Error() string {
	return string(self)
}

type ChannelO chan<- byte
func (self ChannelO) Write(bytes []byte) (n int, err error) {
	for _, byte := range bytes {
		self <- byte
	}
	return len(bytes), nil
}


type ChannelI <-chan byte
func (self ChannelI) Read(bytes []byte) (n int, err error) {
	for index, _ := range bytes {
		if b, ok := <- self; ok {
			bytes[index] = b
		} else {
			return index, io.EOF
		}
	}
	return len(bytes), nil
}

type Compiler struct {
	allowed map[string]bool
	okChecked map[string]time.Time
}
func NewCompiler() *Compiler {
	return &Compiler{make(map[string]bool), make(map[string]time.Time)}
}
func (self *Compiler) Allow(p string) {
	self.allowed[fmt.Sprint("\"", p, "\"")] = true
}
func (self *Compiler) Check(file string) error {
	fstat, err := os.Stat(file)
	if err != nil {
		// Problem stating file
		return err
	}
	if checkTime, ok := self.okChecked[file]; ok && checkTime.After(fstat.ModTime()) {
		// Was checked before, and after the file was last changed
		return nil
	}
	var disallowed []string 
	tree, _ := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	ast.Walk(visitor(func(node ast.Node) {
		if importNode, isImport := node.(*ast.ImportSpec); isImport {
			if importNode.Path != nil {
				if _, ok := self.allowed[importNode.Path.Value]; !ok {
					// This import declaration imports a package that is not allowed
					disallowed = append(disallowed, importNode.Path.Value)
				}
			}
		}
	}), tree)
	if len(disallowed) > 0 {
		var buffer bytes.Buffer
		for index, pkg := range disallowed {
			fmt.Fprint(&buffer, pkg)
			if index < len(disallowed) - 1 {
				fmt.Fprint(&buffer, ", ")
			}
		}
		// We tried to import non-allowed packages
		return Error(fmt.Sprint("Imports of disallowed libraries: ", string((&buffer).Bytes())))
	}
	// We checked this file as OK now
	self.okChecked[file] = time.Now()
	return nil
}
func (self *Compiler) run(file string, stdin <-chan byte, stdout chan<- byte, stderr chan<- byte) {
	defer close(stdout)
	defer close(stderr)
	cmd := exec.Command("go", "run", file)
	cmd.Stdout = ChannelO(stdout)
	cmd.Stdin = ChannelI(stdin)
	cmd.Stderr = ChannelO(stderr)
	err := cmd.Start()
	if err == nil {
		err = cmd.Wait()
		if err != nil {
			ChannelO(stderr).Write([]byte(err.Error()))
		}
	} else {
		ChannelO(stderr).Write([]byte(err.Error()))
	}
}
func (self *Compiler) Run(file string) (stdi chan<- byte, stdo <-chan byte, stde <-chan byte, err error) {
	err = self.Check(file)
	if err != nil {
		return nil, nil, nil, err
	} 
	stderr := make(chan byte)
	stdin := make(chan byte)
	stdout := make(chan byte)
	go self.run(file, (<-chan byte)(stdin), (chan<- byte)(stdout), (chan<- byte)(stderr))
	return (chan<- byte)(stdin), (<-chan byte)(stdout), (<-chan byte)(stderr), nil
}
func (self *Compiler) Compile(file string) error {
	err := self.Check(file)
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd := exec.Command("go", "build", file)
	cmd.Stderr = &stderr
	err = cmd.Run()
	if len((&stderr).Bytes()) > 0 {
		return Error(string(stderr.Bytes()))
	}
	if err != nil {
		return err
	}
	return nil
}
