
package main

import (
	"github.com/zond/gosafe"
	"fmt"
)

func main() {
	c := gosafe.NewCompiler()
	c.Allow("math")
	cmd, err := c.Run("package main\nimport (\n\"fmt\"\n\"math\"\n)\nfunc main() { fmt.Println(math.Sin(10)) }\n")
	fmt.Println(cmd, ", ", err)
	c.Allow("fmt")
	cmd, err = c.Run("package main\nimport (\n\"fmt\"\n\"math\"\n)\nfunc main() { fmt.Println(math.Sin(10)) }\n")
	fmt.Println(cmd, ", ", err)
	close(cmd.Stdin)
	for b := range cmd.Stdout {
		fmt.Print(string(b))
	}
}
