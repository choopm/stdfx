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

package slogfx

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/choopm/stdfx/loggingfx"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// Module returns a slog constructor and adapters to common loggers
var Module = fx.Module(
	"slog", fx.Provide(
		New,
		ToStdlog,
		loggingfx.DefaultConfig,
	),
)

// New returns a new configured *slog.Logger
func New(config loggingfx.Config) (*slog.Logger, error) {
	// parse level
	slevel := slog.LevelInfo // nolint:ineffassign
	switch config.Level {
	case "trace", "debug":
		slevel = slog.LevelDebug
	case "info":
		slevel = slog.LevelInfo
	case "warn":
		slevel = slog.LevelWarn
	case "error", "fatal", "panic":
		slevel = slog.LevelError
	default:
		return nil, fmt.Errorf("unknown log.level: %s", config.Level)
	}

	// build output sink
	var output io.Writer = os.Stdout // nolint:ineffassign
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// config.Output is a filename
		var err error
		output, err = os.OpenFile(config.Output, 0644, os.ModeAppend)
		if err != nil {
			return nil, fmt.Errorf("unable to open log.output: %s", err)
		}
		// this file is closed automatically by go runtime through finalizers
	}

	// build options
	opts := &slog.HandlerOptions{
		Level: slevel,
	}

	// choose a handler to use
	var handler slog.Handler
	switch config.Format {
	case "text", "color", "human", "nice":
		handler = slog.NewTextHandler(output, opts)
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	default:
		return nil, fmt.Errorf("unknown log.format: %s", config.Format)
	}

	// build logger
	logger := slog.New(handler)

	return logger, nil
}

// ToStdlog provides a logging adapter for logging from stdlog to slog.
// It logs everything to info level by default.
func ToStdlog(log *slog.Logger) *log.Logger {
	return slog.NewLogLogger(log.Handler(), slog.LevelInfo)
}

// ToFx provides a logging adapter for logging from fxevent.Logger to slog.
// Designed to be used as a parameter for with fx.WithLogger().
func ToFx(log *slog.Logger) fxevent.Logger {
	return &fxevent.SlogLogger{
		Logger: AtLevel(log, slog.LevelDebug),
	}
}
