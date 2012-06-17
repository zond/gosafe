
package main

import (
	"github.com/zond/gosafe/child"
	"fmt"
	"time"
	"os"
)

func main() {
	runid := time.Now().UnixNano()
	json_in := child.Stdin()
	json_out := child.Stdout()
	var json interface{}
	n := 0
	for {
		if err := json_in.Decode(&json); err == nil {
			resp := make(map[string]interface{})
			resp["runid"] = fmt.Sprint(runid)
			resp["rand"] = fmt.Sprint(n)
			json_out.Encode(resp)
		} else {
			fmt.Fprintln(os.Stderr, "when decoding", err)
			return
		}
		n ++
	}
}
