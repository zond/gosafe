
package gosafe

import (
	"testing"
	"bytes"
	"strings"
)

func compileTest(t *testing.T, c *Compiler, file string, work bool) {
	err := c.Compile(file)
	if work && err != nil {
		t.Error(file, "should compile with", c, ", but got", err)
	} else if !work && err == nil {
		t.Error(file, "should not compile with", c, ", but it did")
	}
}

func runTest(t *testing.T, c *Compiler, file string, work bool, stdin, stdout, stderr string) {
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

