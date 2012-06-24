/*
 A Go tool to safely compile and run Go programs by only allowing importing of whitelisted packages.
 
 Use gosafe.Compiler.Allow(string) to allow given packages, then run code with gosafe.Compiler.Run(string) or gosafe.Compiler.RunFile(string).
 
 Use child.Stdin(), child.Stdout() and child.Stderr() in github.com/zond/gosafe/child to communicate with the child processes via structured data.
 
 Use gosafe.Compiler.Command(string), gosafe.Compiler.CommandFile(string) and gosafe.Cmd.Handle(interface{}, interface{} to create child process handlers that will stay dormant until needed (when gosafe.Cmd.Handle(...) is called), and die again after a customizable timeout without new messages.

 Go to https://github.com/zond/gosafe for the source, of course.
 */
package gosafe

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/zond/tools"
	"go/ast"
	"go/parser"
	"go/token"
	"hash"
	"io"
	"os"
	"os/exec"
	"path"
	"time"
)

const HANDLER_TIMEOUT = time.Second * 10

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

/*
 A wrapper around os/exec.Cmd that provides ready io.Readers and io.Writers for communicating with the contained process.

 Also provides gosafe.Cmd.Encode(interface{}) and gosafe.Cmd.Decode(interface{}) that sends/receives structured data to the child process.

 Use gosafe.Cmd.Handle() to spin up child processes on demand. If they continue living and handling messages after responding to the first call, they will keep on living and handling incoming messages until they get killed from timeout.
*/
type Cmd struct {
	Binary    string
	Cmd       *exec.Cmd
	Stdin     io.WriteCloser
	Stdout    io.Reader
	Stderr    io.Writer
	encoder   *json.Encoder
	decoder   *json.Decoder
	lastEvent time.Time
	// The amount of time idle child processes are allowed to live without handling messages.
	Timeout   time.Duration
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
// Encode sends i to the child process stdin through a json.Encoder.
func (self *Cmd) Encode(i interface{}) error {
	if self.encoder == nil {
		self.encoder = json.NewEncoder(self.Stdin)
	}
	return self.encoder.Encode(i)
}
// Decode receives i from the child process stdout through a json.Decoder.
func (self *Cmd) Decode(i interface{}) error {
	if self.decoder == nil {
		self.decoder = json.NewDecoder(self.Stdout)
	}
	return self.decoder.Decode(i)
}
// Kill will kill the child process if it is alive.
func (self *Cmd) Kill() error {
	if self.Cmd == nil {
		return nil
	}
	if self.Cmd.Process == nil {
		return nil
	}
	return self.Cmd.Process.Kill()
}
// Pid returns the pid of the child process, and whether it was alive.
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
func (self *Cmd) timeout() time.Duration {
	if self.Timeout == 0 {
		return HANDLER_TIMEOUT
	}
	return self.Timeout
}
// Handle starts the child process if it is dead, sends i to the child process using Encode and receives o with the response using Decode.
// Will create a timer that kills this process after gosafe.Cmd.Timeout has passed if no new messages arrive.
func (self *Cmd) Handle(i, o interface{}) error {
	if _, running := self.Pid(); !running {
		return self.reHandle(i, o)
	}
	self.lastEvent = time.Now()
	err := self.Encode(i)
	if err != nil {
		if err.Error() == "write |1: bad file descriptor" {
			return self.reHandle(i, o)
		}
		return err
	}
	err = self.Decode(&o)
	if err != nil {
		if err == io.EOF {
			return self.reHandle(i, o)
		}
		return err
	}
	go func() {
		<-time.After(self.timeout())
		if time.Now().Sub(self.lastEvent) > self.timeout() {
			self.lastEvent = time.Now()
			if err := self.Kill(); err != nil {
				fmt.Fprintln(os.Stderr, "While trying to kill an idle process: ", err)
			}
		}
	}()
	return nil
}
// Start clears all child process-specific state of this Cmd and restart the process.
func (self *Cmd) Start() error {
	self.Cmd = exec.Command(self.Binary)
	self.encoder = nil
	self.decoder = nil
	self.lastEvent = time.Now()
	var err error
	if self.Stdin, err = self.Cmd.StdinPipe(); err != nil {
		return err
	}
	if self.Stdout, err = self.Cmd.StdoutPipe(); err != nil {
		return err
	}
	if self.Stderr == nil {
		self.Cmd.Stderr = os.Stderr
	} else {
		self.Cmd.Stderr = self.Stderr
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

// A compiler of potentially unsafe code.
type Compiler struct {
	allowed    map[string]bool
	okChecked  map[string]time.Time
	okCompiled map[string]time.Time
}

func NewCompiler() *Compiler {
	return &Compiler{make(map[string]bool), make(map[string]time.Time), make(map[string]time.Time)}
}
// Allow will add p to the allowed list of golang packages for this gosafe.Compiler.
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
// Check will return an error if this gosafe.Compiler doesn't allow  the given file to be compiled.
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
			if index < len(disallowed)-1 {
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
// RunFile will start a gosafe.Cmd encapsulating the given file and return it.
func (self *Compiler) RunFile(file string) (cmd *Cmd, err error) {
	cmd, err = self.CommandFile(file)
	if err != nil {
		return nil, err
	}
	cmd.Start()
	return cmd, nil
}
// Run will start a gosafe.Cmd encapsulating the given code and return it.
func (self *Compiler) Run(s string) (cmd *Cmd, err error) {
	cmd, err = self.Command(s)
	if err != nil {
		return nil, err
	}
	cmd.Start()
	return cmd, nil
}
// CommandFile will return a gosafe.Cmd encapsulating the given file.
func (self *Compiler) CommandFile(file string) (cmd *Cmd, err error) {
	compiled, err := self.Compile(file)
	if err != nil {
		return nil, err
	}
	cmd = &Cmd{Binary: compiled}
	return cmd, nil
}
// Command will return a gosafe.Cmd encapsulating the given code.
func (self *Compiler) Command(s string) (cmd *Cmd, err error) {
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
// Compile will compile the given file to a temporary file if deemed safe, and return the path to the resulting binary.
func (self *Compiler) Compile(file string) (output string, err error) {
	output = path.Join(os.TempDir(), fmt.Sprintf("%s.gosafe", self.shorten(file)))
	err = self.CompileTo(file, output)
	if err != nil {
		return "", err
	}
	return output, nil
}
// CompileTo will compile the given file to a given path file if deemed safe.
func (self *Compiler) CompileTo(file, output string) error {
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
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	args := []string{"build", "-ldflags", fmt.Sprint("-o ", output), file}
	cmd := exec.Command("go", args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err = cmd.Run()
	if len((stderr).Bytes()) > 0 {
		return Error(string(stderr.Bytes()))
	}
	if len((stdout).Bytes()) > 0 {
		return Error(string(stdout.Bytes()))
	}
	if err != nil {
		return err
	}
	self.okCompiled[file] = time.Now()
	return nil
}
