
package main

import (
	"github.com/zond/gosafety"
	"os"
	"fmt"
)

func main() {
	fmt.Fprintln(os.Stderr, "test3: 1")
	json_in := gosafety.Stdin().PacketReader().JSONReader()
	fmt.Fprintln(os.Stderr, "test3: 2")
	json := (<-json_in).(map[string]interface{})
	fmt.Fprintln(os.Stderr, "test3: 3")
	json["returning"] = true
	fmt.Fprintln(os.Stderr, "test3: 4")
	json_out := gosafety.Stdout().PacketWriter().JSONWriter()
	fmt.Fprintln(os.Stderr, "test3: 5")
	json_out <- json
	fmt.Fprintln(os.Stderr, "test3: 6")
	<-json_in
	fmt.Fprintln(os.Stderr, "test3: 7")
}
