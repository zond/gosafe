package main

import (
	gosafe "../../"
	"fmt"
	"time"
)

func fetch(cmd *gosafe.Cmd) {
	var m = map[string]interface{}{	"do": "go" }
	var rval map[string]interface{}
	if err := cmd.Handle(m, &rval); err == nil {
		fmt.Println(rval["runid"], rval["rand"])
	} else {
		fmt.Println("unable to handle: ", err)
	}
}

func main() {
	c := gosafe.NewCompiler()
	c.Allow("../../child")
	c.Allow("fmt")
	c.Allow("time")
	if cmd, err := c.CommandFile("child.go"); err == nil {
		cmd.Timeout = time.Second / 2
		fetch(cmd)
		fetch(cmd)
		fetch(cmd)
		fmt.Println("sleeping...")
		time.Sleep(time.Second)
		fetch(cmd)
		fetch(cmd)
		fetch(cmd)
	} else {
		fmt.Println(err)
	}
	
}