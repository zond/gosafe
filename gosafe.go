
package gosafe

import (
	"github.com/zond/tools"
	"fmt"
	"go/parser"
	"go/token"
	"go/ast"
	"bytes"
	"crypto/sha1"
	"hash"
	"os/exec"
	"path"
	"os"
	"time"
	"io"
)

var hasher hash.Hash
func init() {
	hasher = sha1.New()
}

type visitor func(ast.Node)
func (self visitor) Visit(node ast.Node) ast.Visitor {
	self(node)
	return self
}

type Error string
func (self Error) Error() string {
	return string(self)
}

type Cmd struct {
	Cmd *exec.Cmd
	Stdin io.WriteCloser
	Stdout io.Reader
	Stderr io.Reader
}

type Compiler struct {
	allowed map[string]bool
	okChecked map[string]time.Time
	okCompiled map[string]time.Time
}
func NewCompiler() *Compiler {
	return &Compiler{make(map[string]bool), make(map[string]time.Time), make(map[string]time.Time)}
}
func (self *Compiler) Allow(p string) {
	self.allowed[fmt.Sprint("\"", p, "\"")] = true
}
func (self *Compiler) shorten(s string) string {
	hasher.Reset()
	for allowed, _ := range self.allowed {
		hasher.Write([]byte(allowed))
	}
	hasher.Write([]byte(s))
	return tools.NewBigIntBytes(hasher.Sum(nil)).BaseString(tools.MAX_BASE)
}
func (self *Compiler) Check(file string) error {
	tools.TimeIn("Check")
	defer tools.TimeOut("Check")
	fstat, err := os.Stat(file)
	if err != nil {
		// Problem stating file
		return err
	}
	if checkTime, ok := self.okChecked[file]; ok && checkTime.After(fstat.ModTime()) {
		// Was checked before, and after the file was last changed
		return nil
	}
	tools.TimeIn("actual Check")
	defer tools.TimeOut("actual Check")
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
func (self *Compiler) RunFile(file string) (cmd *Cmd, err error) {
	tools.TimeIn("RunFile")
	defer tools.TimeOut("RunFile")
	compiled, err := self.Compile(file)
	if err != nil {
		return nil, err
	} 
	cmd = &Cmd{}
	cmd.Cmd = exec.Command(compiled)
	if cmd.Stdin, err = cmd.Cmd.StdinPipe(); err != nil {
		return nil, err
	}
	if cmd.Stdout, err = cmd.Cmd.StdoutPipe(); err != nil {
		return nil, err
	}
	if cmd.Stderr, err = cmd.Cmd.StderrPipe(); err != nil {
		return nil, err
	}
	if err = cmd.Cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		if err = cmd.Cmd.Wait(); err != nil {
			cmd.Cmd.Stderr.Write([]byte(err.Error()))
		}
	}()
	return cmd, nil
}
func (self *Compiler) Run(s string) (cmd *Cmd, err error) {
	tools.TimeIn("Run")
	defer tools.TimeOut("Run")
	output := path.Join(os.TempDir(), fmt.Sprintf("%s.gosafe.go", self.shorten(s)))
	file, err := os.Create(output)
	if err != nil {
		return nil, err
	}
	defer func() {
		os.Remove(output)
	}()
	file.WriteString(s)
	err = file.Close()
	if err != nil {
		return nil, err
	}
	return self.RunFile(file.Name())
}
func (self *Compiler) Compile(file string) (output string, err error) {
	tools.TimeIn("Compile")
	defer tools.TimeOut("Compile")
	output = path.Join(os.TempDir(), fmt.Sprintf("%s.gosafe", self.shorten(file)))
	err = self.CompileTo(file, output)
	if err != nil {
		return "", err
	}
	return output, nil
}
func (self *Compiler) CompileTo(file, output string) error {
	tools.TimeIn("CompileTo")
	defer tools.TimeOut("CompileTo")
	fstat, err := os.Stat(file)
	if err != nil {
		// Problem stating file
		return err
	}
	if compileTime, ok := self.okCompiled[file]; ok && compileTime.After(fstat.ModTime()) {
		// Was compiled before, and after the file was last changed
		return nil
	}
	err = self.Check(file)
	if err != nil {
		return err
	}
	tools.TimeIn("actual CompileTo")
	defer tools.TimeOut("actual CompileTo")
	var stderr bytes.Buffer
	args := []string{"build", "-ldflags", fmt.Sprint("-o ", output), file}
	cmd := exec.Command("go", args...)
	cmd.Stderr = &stderr
	err = cmd.Run()
	if len((&stderr).Bytes()) > 0 {
		return Error(string(stderr.Bytes()))
	}
	if err != nil {
		return err
	}
	self.okCompiled[file] = time.Now()
	return nil
}
