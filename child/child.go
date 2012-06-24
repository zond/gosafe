
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

var stdin *json.Decoder
var stdout *json.Encoder

// Stdin returns a *json.Decoder that decodes json from stdin.
func Stdin() *json.Decoder {
	if stdin == nil {
		stdin = json.NewDecoder(os.Stdin)
	}
	return stdin
}
// Stdout returns a *json.Encoder that encodes json to stdout.
func Stdout() *json.Encoder {
	if stdout == nil {
		stdout = json.NewEncoder(os.Stdout)
	}
	return stdout
}

// These are the types of return data from a child.Server
const (
	invalid = iota
	Error
	Return
	Callback
)

// NoSuchService is returned when there is no registered service of the wanted name.
const NoSuchService = "No such service: %s"

// NotProperRequest is returned if the server process make a callback home with a Payload other than a nested Request.
const NotProperRequest = "Not proper request: %+v"

// UnknownResponseType is returned if Responses have Type other than Error, Return or Callback.
const UnknownResponseType = "Unknown response type: %+v"

// BadResponseType is returned when the Response is of unexpected type.
const BadResponseType = "Bad response type: %+v"

// Args is a shorthand for the array of interfaces used as arguments
type Args []interface{}

// Request is the type of data the Server expects.
type Request struct {
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

// Server serves requests by listening to Stdin, running Services matching the Name of incoming Requests, and responding
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
// Handle handles a single Request.
func (self Server) Handle(c Request) Response {
	if service, ok := self[c.Name]; ok {
		if rval, err := service.callSafe(c.Args...); err == nil {
			return Response{Return, rval}
		} else {
			return Response{Error, err}
		}
	}
	return Response{Error, fmt.Sprintf(NoSuchService, c.Name)}
}
func (self Server) serve(c chan bool) {
	defer func() {
		close(c)
	}()
	stdin := Stdin()
	stdout := Stdout()
	for {
		var call Request
		if err := stdin.Decode(&call); err == nil {
			stdout.Encode(self.Handle(call))
		} else {
			if err == io.EOF {
				break
			} else {
				stdout.Encode(Response{Error, err})
			}
		}
	}
}

// Call sends a request through Stdout to the parent process and returns the response.
// The parent process has to have gosafe.Cmd#Register'ed the name used 
func Call(name string, args... interface{}) (rval interface{}, err error) {
	if stdin == nil || stdout == nil {
		panic("You can't make callbacks if you haven't initialized Stdin() and Stdout()!")
	}
	if err := stdout.Encode(Response{Callback, Request{name, args}}); err != nil {
		return nil, err
	}
	response := Response{}
	if err = stdin.Decode(&response); err != nil {
		return nil, err
	}
	if response.Type == Error {
		return nil, errors.New(fmt.Sprint(response.Payload))
	} else if response.Type != Return {
		return nil, errors.New(fmt.Sprintf(BadResponseType, response))
	}
	return response.Payload, nil
}

// Create a new Server.
func NewServer() Server {
	return make(Server)
}