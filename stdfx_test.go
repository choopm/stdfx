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

package stdfx_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/examples/everything"
	"github.com/choopm/stdfx/globals"
	"github.com/choopm/stdfx/loggingfx/zerologfx"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

// version is provided by `-ldflags "-X main.version=1.0.0"`
var version string = "unknown"

var configContent = `
log:
  format: color
  level: info
  output: stdout
  timeFormat: "2006-01-02T15:04:05Z07:00"

webserver:
  host: 0.0.0.0
  port: 8080

routes:
- path: /
  content: hello world
- path: /example
  content: example from tests
`

// TestExampleEverything tests the same things as examples/everything
func TestExampleEverything(t *testing.T) {
	d, _ := t.Deadline()
	ctx, cancel := context.WithDeadline(context.Background(), d)
	defer cancel()

	// create config file
	tempDir, err := os.MkdirTemp(os.TempDir(), "go-test")
	require.Nil(t, err)
	configFile := filepath.Join(tempDir, "server.yaml")
	require.Nil(t, os.WriteFile(configFile, []byte(configContent), 0644))
	defer os.RemoveAll(tempDir)

	// update os.Args as if the user started us using arguments
	globals.RootFlagConfigPathDefault = tempDir
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{os.Args[0], "-c", tempDir, "server"}

	// build app
	app := fx.New(
		// logging
		zerologfx.Module,
		fx.WithLogger(zerologfx.ToFx),
		fx.Decorate(zerologfx.Decorator[everything.Config]),

		// viper configuration
		fx.Provide(stdfx.ConfigFile[everything.Config]("server")),
		// cobra commands
		fx.Provide(
			stdfx.AutoRegister(stdfx.VersionCommand(version)),
			stdfx.AutoRegister(stdfx.ConfigCommand[everything.Config]),
			stdfx.AutoRegister(serverCommand),
			stdfx.AutoCommand, // add registered commands to root
		),

		// app start
		fx.Invoke(stdfx.Unprivileged), // abort when being run as root
		fx.Invoke(stdfx.Commander),    // run root cobra command
	)

	// start the app
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return app.Start(ctx)
	})
	time.Sleep(time.Second * 3)

	// test http requests
	client := http.DefaultClient
	req, err := http.NewRequest("GET", "http://localhost:8080/example", nil)
	require.Nil(t, err)
	res, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, 200, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.Nil(t, err)
	assert.Contains(t, string(body), "example from tests")

	// stop the app
	cancel()

	// we should not see any errors
	assert.Nil(t, g.Wait())
}

// serverCommand returns a *cobra.Command to start the server from a ConfigProvider
func serverCommand(
	configProvider configfx.Provider[everything.Config],
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
			server, err := everything.NewServer(cfg, logger)
			if err != nil {
				return err
			}

			// start server using context
			return server.Start(cmd.Context())
		},
	}

	return cmd
}
