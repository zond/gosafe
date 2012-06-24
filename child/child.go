
/*
 A safe package to allow importing into child code. 

 Will simplify communicating with the parent process by letting the child process send and receive structured data over stdin/stdout in a format compatible with gosafe.Cmd.Encode/Decode.
*/
package child

import (
	"io"
	"os"
	"encoding/json"
	"fmt"
	"errors"
)

// Will produce a json.Decoder connected to os.Stdin
func Stdin() *json.Decoder {
	return json.NewDecoder(os.Stdin)
}
// Will produce a json.Encoder connected to os.Stdout
func Stdout() *json.Encoder {
	return json.NewEncoder(os.Stdout)
}

const (
	Error = iota
	Return
	Callback
)

const NoSuchService = "No such service: %s"

type Args []interface{}

type Call struct {
	Name string
	Args Args
}

type Response struct {
	Type int
	Payload interface{}
}

type Service func(args... interface{}) interface{}
func (self Service) callSafe(args... interface{}) (rval interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			rval = nil
			err = errors.New(fmt.Sprint(r))
		}
	}()
	return self(args...), nil
}

type Server map[string]Service
func (self Server) Register(name string, service Service) Server {
	self[name] = service
	return self
}
func (self Server) Start() {
	done := make(chan bool)
	go self.serve(done)
	<-done
}
func (self Server) serve(c chan bool) {
	defer func() {
		close(c)
	}()
	stdin := Stdin()
	stdout := Stdout()
	for {
		var call Call
		if err := stdin.Decode(&call); err == nil {
			if service, ok := self[call.Name]; ok {
				if rval, err := service.callSafe(call.Args...); err == nil {
					stdout.Encode(Response{Return, rval})
				} else {
					stdout.Encode(Response{Error, err})
				}
			} else {
				stdout.Encode(Response{Error, fmt.Sprintf(NoSuchService, call.Name)})
			}
		} else {
			if err == io.EOF {
				break
			} else {
				stdout.Encode(Response{Error, err})
			}
		}
	}
}

func NewServer() Server {
	return make(Server)
}