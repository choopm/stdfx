# stdfx

[![Go Reference](https://pkg.go.dev/badge/github.com/choopm/stdfx.svg)](https://pkg.go.dev/github.com/choopm/stdfx)
[![Actions Status](https://github.com/choopm/stdfx/workflows/unittest/badge.svg)](https://github.com/choopm/stdfx/actions/workflows/unittest.yml)

## Documentation

### Description

stdfx provides standard functionality for golang apps built using
[uber-go/fx](https://github.com/uber-go/fx).

By using stdfx as an application starter you benefit from:

- common app interface
- cli arguments to adjust behavior
- config file discovery and parsing
- override config using environment variables
- builtin cobra subcommands like config or version
- configurable structured logging

See [examples/webserver](./examples/webserver/) to test and experience it in action.

It acts as a demo for every stdfx feature.

### Usage example

A minimal usage might look like this:

<!-- markdownlint-disable MD010 -->
```golang
package main

import (
	"go.uber.org/fx"
	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/loggingfx/zerologfx"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// version is provided by `-ldflags "-X main.version=1.0.0"`
var version string = "unknown"

func main() {
	fx.New(
		// logging
		zerologfx.Module,
		fx.WithLogger(zerologfx.ToFx),
		fx.Decorate(zerologfx.Decorator[yourapp.Config]),

		// viper configuration
		fx.Provide(stdfx.ConfigFile[yourapp.Config]("yourapp")),

		// cobra commands
		fx.Provide(
			stdfx.AutoRegister(stdfx.VersionCommand(version)),
			stdfx.AutoRegister(stdfx.ConfigCommand[yourapp.Config]),
			stdfx.AutoRegister(yourCobraCommand),
			stdfx.AutoCommand, // add registered commands to root
		),

		// app start
		fx.Invoke(stdfx.ContainerEntrypoint("*")), // program is container entrypoint
		fx.Invoke(stdfx.Unprivileged), // abort when being run as root
		fx.Invoke(stdfx.Commander),    // run root cobra command
	).Run()
}

// yourCobraCommand returns a *cobra.Command to start the server from a ConfigProvider
func yourCobraCommand(
	configProvider configfx.Provider[yourapp.Config],
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "server starts the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// fetch the config
			cfg, err := configProvider.Config()
			if err != nil {
				return err
			}

			// rebuild logger and make it global
			logger, err := zerologfx.New(cfg.Logging)
			if err != nil {
				return err
			}
			log.Logger = *logger

			// create server instance
			server, err := yourapp.NewServer(cfg, logger)
			if err != nil {
				return err
			}

			// start server using context
			return server.Start(cmd.Context())
		},
	}

	return cmd
}
```
<!-- markdownlint-enable MD010 -->

## Development

### Dev container

Open this project in Visual Studio Code and select to reopen it inside a dev container.

*If you experience any issues, make sure your IDE supports dev containers:
<https://code.visualstudio.com/docs/devcontainers/containers>*

### Tasks

This project uses [task](https://taskfile.dev/).

Run `task --list` to list all available tasks.
