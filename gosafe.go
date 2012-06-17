
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
	"encoding/json"
	"os/exec"
	"path"
	"os"
	"time"
	"io"
)

const HANDLER_WIPE = time.Second * 10
const HANDLER_RESCUE = time.Second * 5

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
	Binary string
	Cmd *exec.Cmd
	Stdin io.WriteCloser
	Stdout io.Reader
	encoder *json.Encoder
	decoder *json.Decoder
	lastHandle time.Time
}
func (self *Cmd) String() string {
	pid, running := self.Pid()
	var s string
	if running {
		s = fmt.Sprint(pid)
	} else {
		s = "dead"
	}
	return fmt.Sprintf("<Cmd %v %v>", self.Binary, s)
}
func (self *Cmd) Encode(i interface{}) error {
	if self.encoder == nil {
		self.encoder = json.NewEncoder(self.Stdin)
	}
	return self.encoder.Encode(i)
}
func (self *Cmd) Decode(i interface{}) error {
	if self.decoder == nil {
		self.decoder = json.NewDecoder(self.Stdout)
	}
	return self.decoder.Decode(i)
}
func (self *Cmd) Kill() error {
	if self.Cmd == nil {
		return nil
	}
	if self.Cmd.Process == nil {
		return nil
	}
	return self.Cmd.Process.Kill()
}
func (self *Cmd) Pid() (int, bool) {
	if self.Cmd == nil {
		return 0, false
	}
	if self.Cmd.Process == nil {
		return 0, false
	}
	if proc, err := os.FindProcess(self.Cmd.Process.Pid); err == nil {
		return proc.Pid, true
	} 
	return 0, false
}
func (self *Cmd) reHandle(i, o interface{}) error {
	self.Start()
	return self.Handle(i, o)
}
func (self *Cmd) Handle(i, o interface{}) error {
	if _, running := self.Pid(); !running {
		return self.reHandle(i, o)
	}
	self.lastHandle = time.Now()
	err := self.Encode(i)
	if err != nil {
		return err
	}
	err = self.Decode(&o)
	if err != nil {
		if err.Error() == "EOF" {
			return self.reHandle(i, o)
		}
		return err
	}
	go func() {
		<- time.After(HANDLER_WIPE)
		if time.Now().Sub(self.lastHandle) > HANDLER_RESCUE {
			self.Kill()
		}
	}()
	return nil
}
func (self *Cmd) Start() error {
	self.Cmd = exec.Command(self.Binary)
	self.encoder = nil
	self.decoder = nil
	self.lastHandle = time.Now()
	var err error
	if self.Stdin, err = self.Cmd.StdinPipe(); err != nil {
		return err
	}
	if self.Stdout, err = self.Cmd.StdoutPipe(); err != nil {
		return err
	}
	if err := self.Cmd.Start(); err != nil {
		return err
	}
	go func() {
		if err = self.Cmd.Wait(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	return nil
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
	cmd, err = self.CommandFile(file)
	if err != nil {
		return nil, err
	}
	cmd.Start()
	return cmd, nil
}
func (self *Compiler) Run(s string) (cmd *Cmd, err error) {
	cmd, err = self.Command(s)
	if err != nil {
		return nil, err
	} 
	cmd.Start()
	return cmd, nil
}
func (self *Compiler) CommandFile(file string) (cmd *Cmd, err error) {
	tools.TimeIn("RunFile")
	defer tools.TimeOut("RunFile")
	compiled, err := self.Compile(file)
	if err != nil {
		return nil, err
	} 
	cmd = &Cmd{Binary: compiled}
	return cmd, nil
}
func (self *Compiler) Command(s string) (cmd *Cmd, err error) {
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
	return self.CommandFile(file.Name())
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
