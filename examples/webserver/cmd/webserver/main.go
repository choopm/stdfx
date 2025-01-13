/*
Copyright 2024 Christoph Hoopmann

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"go.uber.org/fx"

	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/examples/webserver"
	"github.com/choopm/stdfx/loggingfx/zerologfx"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// version is provided by `-ldflags "-X main.version=1.0.0"`
var version string = "unknown"

// main serves as the entrypoint
func main() {
	fx.New(
		// logging
		zerologfx.Module,
		fx.WithLogger(zerologfx.ToFx),
		fx.Decorate(zerologfx.Decorator[webserver.Config]),

		// viper configuration
		fx.Provide(stdfx.ConfigFile[webserver.Config]("webserver")),

		// cobra commands
		fx.Provide(
			stdfx.AutoRegister(stdfx.VersionCommand(version)),
			stdfx.AutoRegister(stdfx.ConfigCommand[webserver.Config]),
			stdfx.AutoRegister(serverCommand),
			stdfx.AutoCommand, // add registered commands to root
		),

		// app start
		fx.Invoke(stdfx.Unprivileged), // abort when being run as root
		fx.Invoke(stdfx.Commander),    // run root cobra command
	).Run()
}

// serverCommand returns a *cobra.Command to start the server from a ConfigProvider
func serverCommand(
	configProvider configfx.Provider[webserver.Config],
	logger *zerolog.Logger,
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

			// create server instance
			server, err := webserver.NewServer(cfg, logger)
			if err != nil {
				return err
			}

			// start server using context
			return server.Start(cmd.Context())
		},
	}

	return cmd
}
