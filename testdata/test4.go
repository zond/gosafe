package main

import (
	"github.com/zond/gosafe/child"
	"math/rand"
	"time"
	"fmt"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	json_in := child.Stdin()
	json_out := child.Stdout()
	t := rand.Int()
	n := 0
	var json map[string]interface{}
	for {
		if err := json_in.Decode(&json); err == nil {
			json["returning"] = true
			json["n"] = fmt.Sprint(n)
			json["t"] = fmt.Sprint(t)
			json_out.Encode(json)
			n++
		} else {
			break
		}
	}
}
