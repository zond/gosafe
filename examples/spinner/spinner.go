package main

import (
	"github.com/zond/gosafe"
	"fmt"
	"time"
)

func main() {
	c := gosafe.NewCompiler()
	c.Allow("github.com/zond/gosafe/child")
	c.Allow("math/rand")
	c.Allow("fmt")
	c.Allow("time")
	c.Allow("os")
	m := make(map[string]interface{})
	m["do"] = "go"
	if cmd, err := c.CommandFile("child.go"); err == nil {
		cmd.Timeout = time.Second / 2
		var rval map[string]interface{}
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
		fmt.Println("sleeping...")
		time.Sleep(time.Second)
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
		if err := cmd.Handle(m, &rval); err == nil {
			fmt.Println(rval["runid"], rval["rand"])
		} else {
			fmt.Println("unable to handle: ", err)
		}
	} else {
		fmt.Println(err)
		break
	}
	
}