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

package zerologfx

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/choopm/stdfx/loggingfx"
	"github.com/choopm/stdfx/loggingfx/slogfx"
	slogzerolog "github.com/samber/slog-zerolog/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/rs/zerolog"
)

// Module returns a zerolog constructor and adapters to common loggers
var Module = fx.Module(
	"zerolog", fx.Provide(
		New,
		ToSlog,
		loggingfx.DefaultConfig,
	),
)

// New returns a new configured *zerolog.Logger
func New(config loggingfx.Config) (*zerolog.Logger, error) {
	// global options
	zerolog.TimeFieldFormat = config.TimeFormat

	// enable/disable coloring for known formats
	noColor := false
	switch config.Format {
	case "text", "json":
		noColor = true
	case "color", "human", "nice":
		noColor = false
	default:
		return nil, fmt.Errorf("unknown log.format: %s", config.Format)
	}

	// parse level
	zlevel := zerolog.InfoLevel // nolint:ineffassign
	switch config.Level {
	case "disabled":
		zlevel = zerolog.Disabled
	case "trace":
		zlevel = zerolog.TraceLevel
	case "debug":
		zlevel = zerolog.DebugLevel
	case "info":
		zlevel = zerolog.InfoLevel
	case "warn":
		zlevel = zerolog.WarnLevel
	case "error":
		zlevel = zerolog.ErrorLevel
	case "fatal":
		zlevel = zerolog.FatalLevel
	case "panic":
		zlevel = zerolog.PanicLevel
	default:
		return nil, fmt.Errorf("unknown log.level: %s", config.Level)
	}

	// build output sink
	fileOutput := false
	var output io.Writer = os.Stdout // nolint:ineffassign
	switch config.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	default:
		// config.Output is a filename
		fileOutput = true

		var err error
		output, err = os.OpenFile(config.Output, 0644, os.ModeAppend)
		if err != nil {
			return nil, fmt.Errorf("unable to open log.output: %s", err)
		}
		// this file is closed automatically by go runtime through finalizers
	}

	// wrap output into a synchronnized writer (files are already synced)
	if !fileOutput {
		output = zerolog.SyncWriter(output)
	}

	// if we are text based stdout/stderr, wrap it into a ConsoleWriter
	if !fileOutput && config.Format != "json" {
		output = zerolog.ConsoleWriter{
			Out:          output,
			NoColor:      noColor,
			TimeFormat:   config.TimeFormat,
			TimeLocation: time.Local, // you may overwrite location using env TZ
		}
	}

	// build logger
	logger := zerolog.New(output).
		Level(zlevel).
		With().
		Timestamp().
		// Caller().
		Logger()
	// // throttle to 10 messages per second
	// Sample(&zerolog.BurstSampler{
	// 	Burst:  10,
	// 	Period: 1 * time.Second,
	// })

	return &logger, nil
}

// ToSlog provides a logging adapter for logging from slog to zerolog.
// Use this whenever something requires slog and you wish to use zerolog instead.
func ToSlog(log *zerolog.Logger) *slog.Logger {
	// get the current zap og.Level and use it
	// as a default for the slog adapter
	slevel, zlevel := slog.LevelDebug, log.GetLevel()
	for s, z := range slogzerolog.LogLevels {
		if zlevel != z {
			continue
		}
		slevel = s
		break
	}

	return slog.New(slogzerolog.Option{
		Level:  slevel,
		Logger: log,
	}.NewZerologHandler())
}

// ToFx provides a logging adapter for logging from fxevent.Logger to zerolog.
// Designed to be used as a parameter for with fx.WithLogger().
// It will rewrite all log levels to debug if other than error.
func ToFx(log *zerolog.Logger) fxevent.Logger {
	return &fxevent.SlogLogger{
		Logger: slogfx.AtLevelMap(
			ToSlog(log),
			map[slog.Level]slog.Level{
				slog.LevelDebug: slog.LevelDebug,
				slog.LevelInfo:  slog.LevelDebug,
				slog.LevelWarn:  slog.LevelDebug,
				slog.LevelError: slog.LevelError,
			},
		),
	}
}
