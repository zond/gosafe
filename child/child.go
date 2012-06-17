
package child

import (
	"os"
	"encoding/json"
)

func Stdin() *json.Decoder {
	return json.NewDecoder(os.Stdin)
}
func Stdout() *json.Encoder {
	return json.NewEncoder(os.Stdout)
}
func Stderr() *json.Encoder {
	return json.NewEncoder(os.Stderr)
}
