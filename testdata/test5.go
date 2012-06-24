package main

import (
	child "../child"
)

func sin(args... interface{}) interface{} {
	if len(args) != 1 {
		panic("Only one argument to 'sin'")
	}
	f, ok := args[0].(float64)
	if !ok {
		panic("Only floats to 'sin'")
	}
	r, err := child.Call("sin", f)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func main() {
	child.NewServer().Register("sin", sin).Start()
}
