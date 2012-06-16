
package gosafe

import (
	"time"
	"github.com/zond/tools"
	"testing"
	"fmt"
	"bytes"
	"os"
	"strings"
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

func runTest(t *testing.T, c *Compiler, file string, work bool, stdin, stdout, stderr string) {
	tools.TimeIn("runTest")
	defer tools.TimeOut("runTest")
	in_chan, out_chan, err_chan, err := c.Run(file)
	if work && err != nil {
		t.Error(file, "should compile with", c, ", but got", err)
	} else if !work && err == nil {
		t.Error(file, "should not compile with", c, ", but it did")
	}
	if in_chan != nil {
		errbuffer := bytes.NewBufferString("")
		outbuffer := bytes.NewBufferString("")
		inbuffer := bytes.NewBufferString(stdin)
		next_in_byte, err := inbuffer.ReadByte()
		if err != nil {
			close(in_chan)
			in_chan = nil
		}
		cont := true
		for cont {
			select {
			case err_byte, ok := <- err_chan:
				if !ok {
					cont = false
				}
				errbuffer.WriteByte(err_byte)
			case out_byte, ok := <- out_chan:
				if !ok {
					cont = false
				}
				outbuffer.WriteByte(out_byte)
			case in_chan <- next_in_byte:
				next_in_byte, err = inbuffer.ReadByte()
				if err != nil {
					close(in_chan)
					in_chan = nil
				}
			}
		}
		errs := strings.Trim(string(errbuffer.Bytes()), "\x000")
		if errs != stderr {
			t.Errorf("%v should generate stderr %v (%v) but generated %v (%v)\n", file, stderr, []byte(stderr), errs, []byte(errs))
		}
		outs := strings.Trim(string(outbuffer.Bytes()), "\x000")
		if outs != stdout {
			t.Errorf("%v should generate stdout %v (%v) but generated %v (%v)\n", file, stdout, []byte(stdout), outs, []byte(outs))
		}
	}
}

func TestDisallowedRunFmt(t *testing.T) {
	c := NewCompiler()
	runTest(t, c, "testfiles/test1.go", false, "", "", "")
}

func TestSpeed(t *testing.T) {
	tools.TimeClear()
	c := NewCompiler()
	c.Allow("fmt")
	start := time.Now()
	n := 10
	for i := 0; i < n; i++ {
		runTest(t, c, "testfiles/test1.go", true, "", "test1.go", "")
	}
	fmt.Println(n, "runs takes", time.Now().Sub(start))
}

func TestAllowedRunFmt(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	runTest(t, c, "testfiles/test1.go", true, "", "test1.go", "")
}

func TestAllowedFmt(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	compileTest(t, c, "testfiles/test1.go", true)
}

func TestDisallowedFmt(t *testing.T) {
	c := NewCompiler()
	compileTest(t, c, "testfiles/test1.go", false)
}

func TestAllowedC(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	c.Allow("C")
	compileTest(t, c, "testfiles/test2.go", true)
}

func TestDisallowedC(t *testing.T) {
	c := NewCompiler()
	c.Allow("fmt")
	compileTest(t, c, "testfiles/test2.go", false)
}

