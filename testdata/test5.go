package main

import (
	child "../child"
	"math"
)

func sin(args... interface{}) interface{} {
	if len(args) != 1 {
		panic("Only one argument to 'sin'")
	}
	f, ok := args[0].(float64)
	if !ok {
		panic("Only floats to 'sin'")
	}
	return math.Sin(f)
}

func main() {
	child.NewServer().Register("sin", sin).Start()
}
