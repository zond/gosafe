/*
 * cmd, _ := c.RunFile(f)
 * outj := gosafety.NewJSONWriter(cmd.Stdin)
 * inj := gosafety.NewJSONReader(cmd.Stdout)
 * json_from_process := <- inj
 * outj <- my_json_response
 */
package main

import (
	"github.com/zond/gosafety"
)

func main() {
	json_in := gosafety.Stdin().PacketReader().JSONReader()
	json := (<-json_in).(map[string]interface{})
	json["returning"] = true
	json_out := gosafety.Stdout().PacketWriter().JSONWriter()
	json_out <- json
	<-json_in
}
