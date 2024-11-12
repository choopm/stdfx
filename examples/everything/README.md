# everything

This example shows the usage of every stdfx feature.

You can choose to run several subcommands:

```shell
# run the server
go run ./cmd/everything -c . server
# or
task run

# or try different subcommands
go run ./cmd/everything -c . help
go run ./cmd/everything -c . version
go run ./cmd/everything -c . config validate
go run ./cmd/everything -c . config get webserver.port
```

You can also build the binary and run it natively:

```shell
task build

# afterwards run

cmd/everything/everything -c . server
```
