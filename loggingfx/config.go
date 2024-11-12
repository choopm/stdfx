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

package loggingfx

import (
	"fmt"
	"os"
	"time"

	"github.com/creasty/defaults"
)

// ConfigWithLogging denotes types which implement LoggingConfig().
// Used to decorate loggers if a config provides logging details.
type ConfigWithLogging interface {
	LoggingConfig() Config
}

// Config defines a configuration for use with loggers
type Config struct {
	// Level must be supported by the selected log adapter, most support this:
	// "debug", "info", "warn", "error"
	// some include more level:
	// "trace", "fatal"
	Level string `mapstructure:"level" default:"info"`

	// Output is the logging sink to use, currently supported:
	// "stdout", "stderr", "<filename>"
	Output string `mapstructure:"output" default:"stdout"`

	// Format is the logging encoding, currently supported:
	// "text", "json"
	Format string `mapstructure:"format" default:"text"`

	// FormatTime is the time encoding, all golang time formats are supported.
	// Defaults to [time.RFC3339]
	TimeFormat string `mapstructure:"timeFormat" default:""`
}

// DefaultConfig returns the default logging configuration to be used until a
// config file has been parsed to configure the real logger.
// It reads environment variables LOG_* to adjust logging as early as possible
// before even config parsing takes place.
func DefaultConfig() (Config, error) {
	config := Config{
		Level:      os.Getenv("LOG_LEVEL"),
		Output:     os.Getenv("LOG_OUTPUT"),
		Format:     os.Getenv("LOG_FORMAT"),
		TimeFormat: os.Getenv("LOG_TIMEFORMAT"),
	}

	if err := defaults.Set(&config); err != nil {
		return config, fmt.Errorf("settings defaults: %s", err)
	}

	// no other way to set the constant in struct tags without copying its value
	if len(config.TimeFormat) == 0 {
		config.TimeFormat = time.RFC3339
	}

	return config, nil
}
