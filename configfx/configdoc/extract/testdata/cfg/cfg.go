// Package cfg is a config to extract docs from.
package cfg

import "time"

// Config is the whole config.
type Config struct {
	// Log configures logging.
	Log Log `mapstructure:"log"`

	Listener `mapstructure:",squash"`

	// Roles are granted to everyone.
	Roles []*Role `mapstructure:"roles"`
	// only read on submission

	// Timeout bounds everything.
	Timeout time.Duration `mapstructure:"timeout"` // a duration

	// Derived is set at runtime.
	Derived string
	// Skipped is never read.
	Skipped string `mapstructure:"-"`
	private string
}

// Log configures logging.
type Log struct {
	// Level is the least severity logged.
	Level        Level  `mapstructure:"level"`
	Undocumented string `mapstructure:"undocumented"`
}

// Level is a severity.
type Level string

const (
	// LevelDebug logs a lot.
	LevelDebug Level = "debug"
	// LevelInfo logs what matters.
	LevelInfo Level = "info"

	unrelated = "x"
)

// Listener is where to listen.
type Listener struct {
	// Host to listen on
	Host string `mapstructure:"host"`

	// Port to listen on
	Port int `mapstructure:"port"`
}

type (
	// Role is a set of scopes.
	Role struct {
		// Name of the role
		Name string `mapstructure:"name"`
	}
)
