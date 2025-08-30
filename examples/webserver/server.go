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

package webserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

// Server state struct
type Server struct {
	config *Config
	log    *zerolog.Logger
	mux    *http.ServeMux
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
		mux:    http.NewServeMux(),
	}

	return s, nil
}

// Start starts the server using ctx
func (s *Server) Start(ctx context.Context) error {
	s.log.Trace().
		Interface("config", s.config).
		Msg("initializing server")

	if err := s.Reconfigure(s.config); err != nil {
		return err
	}

	s.log.Trace().
		Msg("starting server")
	g, ctx := errgroup.WithContext(ctx)

	// build and start webserver
	addr := net.JoinHostPort(s.config.Webserver.Host,
		strconv.Itoa(s.config.Webserver.Port),
	)
	server := &http.Server{Addr: addr, Handler: s}
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

// Reconfigure restarts the server using a new config or error
func (s *Server) Reconfigure(cfg *Config) error {
	mux := http.NewServeMux()

	// register routes
	for _, route := range s.config.Routes {
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, route.Content)
		})
	}

	// replace server mux
	s.mux = mux

	return nil
}

// ServeHTTP implements http.Handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
