
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

// Stdin returns a *json.Decoder that decodes json from stdin.
func Stdin() *json.Decoder {
	return json.NewDecoder(os.Stdin)
}
// Stdout returns a *json.Encoder that encodes json to stdout.
func Stdout() *json.Encoder {
	return json.NewEncoder(os.Stdout)
}

// These are the types of return data from a child.Server
const (
	Error = iota
	Return
	Callback
)

// NoSuchService is returned when there is no registered service of the wanted name.
const NoSuchService = "No such service: %s"

// Args is a shorthand for the array of interfaces used as arguments
type Args []interface{}

// Call is the type of data the Server expects.
type Call struct {
	Name string
	Args Args
}

// Response is the type of data the Server outputs.
type Response struct {
	Type int
	Payload interface{}
}

// Service is the type of function the Server can use to serve requests.
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

// Server serves requests by listening to Stdin, running Services matching the Name of incoming Calls, and responding
// with their return values.
type Server map[string]Service
func (self Server) Register(name string, service Service) Server {
	self[name] = service
	return self
}
// Start the server and run it forever.
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

// Create a new Server.
func NewServer() Server {
	return make(Server)
}