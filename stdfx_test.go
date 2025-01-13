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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/globals"
	"github.com/choopm/stdfx/loggingfx"
	"github.com/choopm/stdfx/loggingfx/zerologfx"
	"github.com/go-viper/mapstructure/v2"
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

// TestExampleWebserver tests the same things as examples/webserver
func TestExampleWebserver(t *testing.T) {
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
		fx.Decorate(zerologfx.Decorator[Config]),

		// viper configuration
		fx.Provide(stdfx.ConfigFile[Config]("server")),
		// cobra commands
		fx.Provide(
			stdfx.AutoRegister(stdfx.VersionCommand(version)),
			stdfx.AutoRegister(stdfx.ConfigCommand[Config]),
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
	configProvider configfx.Provider[Config],
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
			server, err := NewServer(cfg, logger)
			if err != nil {
				return err
			}

			// start server using context
			return server.Start(cmd.Context())
		},
	}

	return cmd
}

// Config struct stores all config data.
type Config struct {
	Logging loggingfx.Config `mapstructure:"log"`

	// Webserver defines the http server config
	Webserver WebserverConfig `mapstructure:"webserver"`

	// Routes defines the webserver routes
	Routes []*Route `mapstructure:"routes" default:"[]"`
}

// Validate validates the Config
func (c *Config) Validate() error {
	if err := c.Webserver.Validate(); err != nil {
		return err
	}
	for i, route := range c.Routes {
		if err := route.Validate(); err != nil {
			return fmt.Errorf("route %d (%s): %s", i, route.Path, err)
		}
	}

	return nil
}

// WebserverConfig holds the webserver config
type WebserverConfig struct {
	// Host is the listening host to use when starting a server
	Host string `mapstructure:"host" default:"0.0.0.0"`

	// Port is the listening port to use when starting a server
	Port int `mapstructure:"port" default:"8080"`
}

// Validate validates the HTTPConfig
func (c *WebserverConfig) Validate() error {
	if len(c.Host) == 0 {
		return fmt.Errorf("missing webserver.host")
	}
	if c.Port == 0 {
		return fmt.Errorf("missing webserver.port")
	}

	return nil
}

// Route maps paths to content
type Route struct {
	// Path is the webserver path to register
	Path string `mapstructure:"path"`

	// Content is the content to deliver on this path
	Content any `mapstructure:"content"`
}

// Validate validates the config
func (c *Route) Validate() error {
	if len(c.Path) == 0 {
		return fmt.Errorf("missing path")
	}
	if c.Content == nil {
		return fmt.Errorf("missing content")
	}

	return nil
}

// DecodeHook returns the composite decoding hook for decoding Config
func (c *Config) DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
	// knx group addresses listed as an example
	// knxGroupAddressDecoder(),
	)
}

// // knxGroupAddressDecoder returns a decoder for knx group addresses.
// // It parses strings of "1/2/3" into cemi.GroupAddr.
// func knxGroupAddressDecoder() mapstructure.DecodeHookFunc {
// 	// groupAddressDecoder returns a DecodeHookFunc that converts
// 	// string to cemi.GroupAddress or error.
// 	return func(
// 		f reflect.Type,
// 		t reflect.Type,
// 		data interface{},
// 	) (interface{}, error) {
// 		if f.Kind() != reflect.String {
// 			return data, nil
// 		}
// 		if t != reflect.TypeOf(cemi.GroupAddr(0)) {
// 			return data, nil
// 		}

// 		// Convert it by parsing
// 		return cemi.NewGroupAddrString(data.(string))
// 	}
// }

// LoggingConfig returns the loggingfx.Config.
// This implements an interface to support log decorators.
func (c *Config) LoggingConfig() loggingfx.Config {
	return c.Logging
}

// Server state struct
type Server struct {
	config *Config
	log    *zerolog.Logger
}

// NewServer creates a new *Server instance using a provided config
func NewServer(config *Config, logger *zerolog.Logger) (*Server, error) {
	// validate config
	if config == nil {
		return nil, errors.New("missing config")
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config: %s", err)
	}

	// init logger if missing
	if logger == nil {
		l := zerolog.Nop()
		logger = &l
	}

	s := &Server{
		config: config,
		log:    logger,
	}

	return s, nil
}

// Start starts the server using ctx
func (s *Server) Start(ctx context.Context) error {
	s.log.Trace().
		Interface("config", s.config).
		Msg("initializing server")

	// register routes
	for _, route := range s.config.Routes {
		http.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, route.Content)
		})
	}

	s.log.Trace().
		Msg("starting server")
	g, ctx := errgroup.WithContext(ctx)

	// build and start webserver
	addr := net.JoinHostPort(s.config.Webserver.Host,
		strconv.Itoa(s.config.Webserver.Port),
	)
	server := &http.Server{Addr: addr, Handler: nil}
	// shutdown hook, registered before starting
	context.AfterFunc(ctx, func() {
		_ = server.Close()
	})
	g.Go(func() error {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		return nil
	})

	// wait for started tasks
	s.log.Info().
		Str("addr", addr).
		Msg("server is running")
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	s.log.Info().Msg("server stopped")

	return nil
}
