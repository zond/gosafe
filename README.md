# gosafe

A Go tool to safely compile and run Go programs by only allowing importing of whitelisted packages.

## Why

To enable running any piece of Go code (even if it comes from unknown sources) with ease and safety.

## How

Use `Compiler.Allow` to allow given packages, then run code with `Compiler.Run` or `Compiler.RunFile`.

See https://github.com/zond/gosafe/blob/master/examples/example.go

## Communicating with child processes

Use `child.Stdin()`, `child.Stdout()` and `child.Stderr()` in https://github.com/zond/gosafe/blob/master/child/child.go to communicate with the child processes via structured data. 

See https://github.com/zond/gosafe/blob/master/testfiles/test3.go for an example.

## On demand child processes

Use `gosafe.Compiler#Command`, `gosafe.Compiler#CommandFile` and `gosafe.Cmd#Handle` to create child process handlers that will stay dormant until needed (when `gosafe.Cmd#Handle` is called), and die again after a customizable timeout without new messages.

See https://github.com/zond/gosafe/tree/master/examples/spinner for an example.

## Documentation

http://go.pkgdoc.org/github.com/zond/gosafe
