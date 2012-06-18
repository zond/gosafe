
/*
 A safe package to allow importing into child code. 

 Will simplify communicating with the parent process by letting the child process send and receive structured data over stdin/stdout in a format compatible with gosafe.Cmd.Encode/Decode.
*/
package child

import (
	"os"
	"encoding/json"
)

// Will produce a json.Decoder connected to os.Stdin
func Stdin() *json.Decoder {
	return json.NewDecoder(os.Stdin)
}
// Will produce a json.Encoder connected to os.Stdout
func Stdout() *json.Encoder {
	return json.NewEncoder(os.Stdout)
}
