package main

import (
	gosafe "../../"
	"fmt"
)

var db map[string]interface{}

func get(args... interface{}) interface{} {
	rval := db[args[0].(string)]
	return rval
}

func set(args... interface{}) interface{} {
	db[args[0].(string)] = args[1]
	return nil
}

func main() {
	db = make(map[string]interface{})
	c := gosafe.NewCompiler()
	c.Allow("../../child")
	cmd, err := c.CommandFile("child.go")
	if err != nil {
		panic(err.Error())
	}
	cmd.Register("get", get)
	cmd.Register("set", set)
	fmt.Println(cmd.Call("sum", 0.1))
	fmt.Println(cmd.Call("sum", 0.4))
	fmt.Println(cmd.Call("sum", 1.2))
	fmt.Println(cmd.Call("sum", 30))
}