# webserver

This example shows the usage of every stdfx feature.

You can choose to run several subcommands:

```shell
# run the server
go run ./cmd/webserver -c . server
# or
task run

# or try different subcommands
go run ./cmd/webserver -c . help
go run ./cmd/webserver -c . version
go run ./cmd/webserver -c . config validate
go run ./cmd/webserver -c . config get webserver.port
```

You can also build the binary and run it natively:

```shell
task build

# afterwards run

cmd/webserver/webserver -c . server
```
