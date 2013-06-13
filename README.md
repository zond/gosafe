# gosafe

A Go tool to safely compile and run Go programs by only allowing importing of whitelisted packages.

## Caveats

If you are not careful, running in parallell might let the child processes execute arbitrary code: https://github.com/zond/gosafe/issues/1

## Why

To enable running any piece of Go code (even if it comes from unknown sources) with ease and safety.

## How

Use `Compiler.Allow` to allow given packages, then run code with `Compiler.Run` or `Compiler.RunFile`.

See https://github.com/zond/gosafe/blob/master/examples/example.go

## Communicating with child processes

Use `child.Stdin()`, `child.Stdout()` and `child.Stderr()` in https://github.com/zond/gosafe/blob/master/child/child.go to communicate with the child processes via structured data. 

## On demand child processes

Use `gosafe.Compiler#Command`, `gosafe.Compiler#CommandFile` and `gosafe.Cmd#Handle` to create child process handlers that will stay dormant until needed (when `gosafe.Cmd#Handle` is called), and die again after a customizable timeout without new messages.

See https://github.com/zond/gosafe/tree/master/examples/spinner for an example.

## On demand child processes with transparent method calling and callbacks to the mother process

Use `child.NewServer`, `child.Server#Register` and `child.Server#Start` to create child processes serving many different types of calls from the parent process.

Then use `gosafe.Cmd#Register` to register callbacks that the child processes can use to access data outside their runtime (such as private persistence providers for example) before responding with their final return value.

See https://github.com/zond/gosafe/blob/master/examples/server/server.go for an example.

## Documentation

http://go.pkgdoc.org/github.com/zond/gosafe
