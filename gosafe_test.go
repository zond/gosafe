package gosafe

import (
	"bytes"
	"github.com/zond/tools"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

func compileTest(t *testing.T, c *Compiler, file string, work bool) {
	output, err := c.Compile(file)
	if output != "" {
		defer os.Remove(output)
	}
	if work {
		if err != nil {
			t.Error(file, "should compile with", c, ", but got", err)
		}
		if output == "" {
			t.Error(file, "should produce a file when compiled with", c, "but got nothing")
		} else {
			fstat, err := os.Stat(output)
			if err != nil {
				t.Error(file, "should produce a nice file when compiled with", c, "but got", err, "when stating")
			}
			wanted_mode := "-rwxr-xr-x"
			if fstat.Mode().String() != wanted_mode {
				t.Error(file, "should produce a file with mode", wanted_mode, "when compiled with", c, "but got", fstat.Mode())
			}
		}
	} else {
		if err == nil {
			t.Error(file, "should not compile with", c, ", but it did")
		}
		if output != "" {
			fstat, err := os.Stat(output)
			if err == nil {
				t.Error(file, "should not produce a file when compiled with", c, "but got", fstat, "when stating")
			}
		}
	}
}

func runStringTest(t *testing.T, c *Compiler, s string, work bool, stdin, stdout string) {
	runTest(t, c, s, work, stdin, stdout, false)
}

func runFileTest(t *testing.T, c *Compiler, f string, work bool, stdin, stdout string) {
	runTest(t, c, f, work, stdin, stdout, true)
}

func runTest(t *testing.T, c *Compiler, data string, work bool, stdin, stdout string, file bool) {
	tools.TimeIn("runTest")
	defer tools.TimeOut("runTest")
	var cmd *Cmd
	var err error
	if file {
		cmd, err = c.RunFile(data)
	} else {
		cmd, err = c.Run(data)
	}
	cmdTest(t, cmd, err, data, work, stdin, stdout)
}

func cmdTest(t *testing.T, cmd *Cmd, err error, data string, work bool, stdin, stdout string) {
	if work && err != nil {
		t.Error(data, "should compile, but got", err)
	} else if !work && err == nil {
		t.Error(data, "should not compile, but it did")
	}
	if cmd != nil {
		outbuffer := bytes.NewBufferString("")
		done := make(chan bool)
		go func() {
			b, err := ioutil.ReadAll(cmd.Stdout)
			if err != nil {
				t.Error(data, "should have a readable stdout, but got", err)
			}
			outbuffer.Write(b)
			done <- true
		}()
		cmd.Stdin.Write([]byte(stdout))
		cmd.Stdin.Close()
		<-done
		outs := strings.Trim(string(outbuffer.Bytes()), "\x000")
		if outs != stdout {
			t.Errorf("%v should generate stdout %v (%v) but generated %v (%v)\n", data, stdout, []byte(stdout), outs, []byte(outs))
		}
	}
}

func TestDisallowedRunFmt(t *testing.T) {
	c := NewCompiler()
	runFileTest(t, c, "testdata/test1.go", false, "", "")
}

func TestDisallowedRunString(t *testing.T) {
	c := NewCompiler()
	s := "package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"teststring\") }"
	runStringTest(t, c, s, false, "", "")
}

func TestAllowedRunString(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	s := "package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"teststring\") }\n"
	runStringTest(t, c, s, true, "", "teststring")
}

func TestSpeedString(t *testing.T) {
	tools.TimeClear()
	c := NewCompiler()
	c.Allow("fmt")
	n := 10
	s := "package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"teststring\") }\n"
	for i := 0; i < n; i++ {
		runStringTest(t, c, s, true, "", "teststring")
	}
}

func TestSpeed(t *testing.T) {
	tools.TimeClear()
	c := NewCompiler()
	c.Allow("fmt")
	n := 10
	for i := 0; i < n; i++ {
		runFileTest(t, c, "testdata/test1.go", true, "", "test1.go")
	}
}

func mapTest(t *testing.T, i1, i2 interface{}) {
	if m1, ok := i1.(map[string]interface{}); ok {
		if m2, ok := i2.(map[string]interface{}); ok {
			if len(m1) == len(m2) {
				for k, v1 := range m1 {
					if v2, ok := m2[k]; !ok || !reflect.DeepEqual(v1, v2) {
						t.Error("expected ", m1, " but got ", m2)
					}
				}
			} else {
				t.Error("expected ", m1, " but got ", m2)
			}
		} else {
			t.Error("expected two maps, but got ", i1, " and ", i2)
		}
	} else {
		t.Error("expected two maps, but got ", i1, " and ", i2)
	}
}

func TestHandling(t *testing.T) {
	c := NewCompiler()
	c.Allow("github.com/zond/gosafe/child")
	s := "testdata/test3.go"
	cmd, err := c.CommandFile(s)
	if err == nil {
		handleTest(t, cmd)
		handleTest(t, cmd)
		handleTest(t, cmd)
		handleTest(t, cmd)
	} else {
		t.Error(s, "should compile, but got", err)
	}
}

func handleTest(t *testing.T, cmd *Cmd) {
	data := make(map[string]interface{})
	data["yo"] = "who's in the house?"
	var resp interface{}
	err := cmd.Handle(data, &resp)
	if err == nil {
		data["returning"] = true
		mapTest(t, data, resp)
	} else {
		t.Error(cmd.Binary, "should handle", data, ", but got", err)
	}
}

func continuousHandleTest(t *testing.T, cmd *Cmd, ti interface{}, ni map[interface{}]bool) {
	data := make(map[string]interface{})
	data["yo"] = "who's in the house?"
	var resp interface{}
	err := cmd.Handle(data, &resp)
	if err == nil {
		if m, ok := resp.(map[string]interface{}); ok {
			if len(m) == 4 {
				if reflect.DeepEqual(ti, m["t"]) {
					if _, ok := ni[m["n"]]; ok {
						t.Error("handling should give a map with a unique 'n' (have ", ni, "), but got", m)
					} else {
						ni[m["n"]] = true
					}
				} else {
					t.Error("handling should give a map with 't'", ti, "but got", m["t"])
				}
			} else {
				t.Error("handling should give a map of size 4, got ", m)
			}
		} else {
			t.Error("handling should give a map, got", resp)
		}
	} else {
		t.Error("expected handling, got", err)
	}
}

func TestContinuousHandling(t *testing.T) {
	c := NewCompiler()
	c.Allow("github.com/zond/gosafe/child")
	c.Allow("math/rand")
	c.Allow("time")
	c.Allow("fmt")
	s := "testdata/test4.go"
	cmd, err := c.CommandFile(s)
	if err == nil {
		data := make(map[string]interface{})
		data["yo"] = "who's in the house?"
		var resp interface{}
		err := cmd.Handle(data, &resp)
		var ti interface{}
		ni := make(map[interface{}]bool)
		if err == nil {
			if m, ok := resp.(map[string]interface{}); ok {
				if len(m) == 4 {
					ti = m["t"]
					ni[m["n"]] = true
					if m["yo"] != "who's in the house?" {
						t.Error(s, "should generate a map based on", data, " but got", m)
					}
					continuousHandleTest(t, cmd, ti, ni)
					continuousHandleTest(t, cmd, ti, ni)
					continuousHandleTest(t, cmd, ti, ni)
					continuousHandleTest(t, cmd, ti, ni)
				} else {
					t.Error(s, "should generate maps of len 4, but got", m)
				}
			} else {
				t.Error(s, "should generate maps, but got", resp)
			}
		} else {
			t.Error(s, "should handle", data, "but got", err)
		}
	} else {
		t.Error(s, "should compile, but got", err)
	}
}

func TestRepeatedRuns(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	s := "package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"teststring\") }\n"
	cmd, err := c.Command(s)
	if err == nil {
		cmd.Start()
		cmdTest(t, cmd, err, s, true, "", "teststring")
		cmd.Start()
		cmdTest(t, cmd, err, s, true, "", "teststring")
		cmd.Start()
		cmdTest(t, cmd, err, s, true, "", "teststring")
	} else {
		t.Error(s, "should compile, got", err)
	}
}

func TestGosafety(t *testing.T) {
	c := NewCompiler()
	c.Allow("time")
	c.Allow("os")
	c.Allow("fmt")
	c.Allow("github.com/zond/gosafe/child")
	f := "testdata/test3.go"
	cmd, err := c.RunFile(f)
	if err == nil {
		done := make(chan bool)
		data := make(map[string]interface{})
		data["yo"] = "who's in the house?"
		go func() {
			var indata interface{}
			if cmd.Decode(&indata); err == nil {
				if injson, ok := indata.(map[string]interface{}); ok {
					data["returning"] = true
					mapTest(t, data, injson)
				} else {
					t.Error(f, "should send us a map[string]interface{}, got", indata)
				}
			} else {
				t.Error(f, "should send us json data, got", err)
			}
			done <- true
			cmd.Stdin.Close()
		}()
		if err = cmd.Encode(data); err != nil {
			t.Error(f, "should get some json, got", err)
		}
		<-done
	} else {
		t.Error(f, "should be runnable, but got", err)
	}
}

func TestAllowedRunFmt(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	runFileTest(t, c, "testdata/test1.go", true, "", "test1.go")
}

func TestAllowedFmt(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	compileTest(t, c, "testdata/test1.go", true)
}

func TestDisallowedFmt(t *testing.T) {
	c := NewCompiler()
	compileTest(t, c, "testdata/test1.go", false)
}

func TestAllowedC(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	c.Allow("C")
	compileTest(t, c, "testdata/test2.go", true)
}

func TestDisallowedC(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	compileTest(t, c, "testdata/test2.go", false)
}
