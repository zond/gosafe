
package main

import (
	"../../child"
)

func sum(args... interface{}) interface{} {
	i := args[0].(float64)
	x, _ := child.Call("get", "sum")
	f, ok := x.(float64)
	if !ok {
		f = 0.0
	}
	i = i + f
	child.Call("set", "sum", i)
	return i
}

func main() {
	server := child.NewServer()
	server.Register("sum", sum)
	server.Start()
}
