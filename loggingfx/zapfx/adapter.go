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

package zapfx

import (
	"fmt"
	"log/slog"

	"github.com/choopm/stdfx/loggingfx"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Module returns a zap constructor and adapters to common loggers
var Module = fx.Module(
	"zap", fx.Provide(
		New,
		ToSlog,
		loggingfx.DefaultConfig,
	),
)

// New returns a new configured *zap.Logger
func New(config loggingfx.Config) (*zap.Logger, error) {
	var zconfig zap.Config

	// choose production development
	switch config.Format {
	case "text", "json":
		zconfig = zap.NewProductionConfig()
	case "color", "human", "nice":
		zconfig = zap.NewDevelopmentConfig()
	default:
		return nil, fmt.Errorf("unknown log.format: %s", config.Format)
	}

	// parse and set level
	switch config.Level {
	case "trace", "debug":
		zconfig.Level.SetLevel(zapcore.DebugLevel)
	case "info":
		zconfig.Level.SetLevel(zapcore.InfoLevel)
	case "warn":
		zconfig.Level.SetLevel(zapcore.WarnLevel)
	case "error":
		zconfig.Level.SetLevel(zapcore.ErrorLevel)
	case "fatal":
		zconfig.Level.SetLevel(zapcore.FatalLevel)
	case "panic":
		zconfig.Level.SetLevel(zapcore.PanicLevel)
	default:
		return nil, fmt.Errorf("unknown log.level: %s", config.Level)
	}

	// set output sink
	zconfig.OutputPaths = []string{config.Output}

	// if we are text based stdout/stderr, enable coloring
	if config.Output == "stdout" || config.Output == "stderr" {
		switch config.Format {
		case "color", "human", "nice":
			zconfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	}

	// build logger
	logger, err := zconfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// ToSlog provides a logging adapter for logging from slog to zap.
// Use this whenever something requires slog and you wish to use zap instead.
func ToSlog(log *zap.Logger) *slog.Logger {
	// get the current zap og.Level and use it
	// as a default for the slog adapter
	slevel, zlevel := slog.LevelDebug, log.Level()
	for s, z := range slogzap.LogLevels {
		if zlevel != z {
			continue
		}
		slevel = s
		break
	}

	return slog.New(slogzap.Option{
		Level:  slevel,
		Logger: log,
	}.NewZapHandler())
}

// ToFx provides a logging adapter for logging from fxevent.Logger to slog.
// Designed to be used as a parameter for with fx.WithLogger().
// Unlike the other ToFx methods this one does not enforce DebugLevel.
func ToFx(log *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{
		Logger: log,
	}
}
