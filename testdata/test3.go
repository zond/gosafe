/*
 * cmd, _ := c.RunFile(f)
 * outj := json.NewEncoder(cmd.Stdin)
 * inj := json.NewDecoder(cmd.Stdout)
 * var json_from_process interface{}
 * inj.Decode(&json_from_process)
 * my_json_response := "hello!"
 * outj.Encode(my_json_response)
 */
package main

import (
	child "../child"
)

func main() {
	json_in := child.Stdin()
	var json map[string]interface{}
	json_in.Decode(&json)
	json["returning"] = true
	json_out := child.Stdout()
	json_out.Encode(json)
}
