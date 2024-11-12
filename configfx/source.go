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

package configfx

import (
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/choopm/stdfx/globals"
	"github.com/spf13/viper"
)

// Source defines a common interface for config sources
type Source[T any] interface {
	// Viper shall return a *viper.Viper intance for
	// any type implementing this interface.
	Viper(opts ...viper.Option) *viper.Viper
}

// SourceFile is a config source using files
type SourceFile[T any] struct {
	Source[T]

	// log defines the Logger instance to use
	log *slog.Logger

	// configName is the name of a config file without extension
	configName string
	// searchPaths are additional paths to use when looking for configName
	searchPaths []string

	// flagEnvPrefix for use as a flag with viper autoenv
	flagEnvPrefix *string
	// flagConfigPath for use as a flag to provide an additional path
	flagConfigPath *string
	// flagConfigPath for use as a flag to provide an absolute config path
	flagAbsolutePath *string
}

// NewSourceFile returns a Source constructor based on a config file.
// configName specifies the file to search for in default paths.
// A developer can optionally override searchPaths.
// userFlags can be used to allow adjustment of config loading by
// users using cobra.Command.PersistentFlags for example.
func NewSourceFile[T any](
	configName string,
	searchPaths ...string,
) func(*slog.Logger) Source[T] {
	return func(log *slog.Logger) Source[T] {
		// get default env prefix from configName
		defEnvPrefix := DefaultEnvironmentPrefix(configName)

		// use default searchPaths if nothing was provided by library user
		if len(searchPaths) == 0 {
			searchPaths = DefaultFileSearchPaths(configName)
		}

		return &SourceFile[T]{
			// general
			log: log.With(slog.String("context", "config-file")),

			// config file specific
			configName:  configName,
			searchPaths: searchPaths,

			// globalFlags for adjustment of config loading
			flagEnvPrefix: globals.RootFlags.StringP(
				"env-prefix", "e", defEnvPrefix,
				"Environment prefix to use when overriding config via AutomaticEnv"),
			flagConfigPath: globals.RootFlags.StringP(
				"config-path", "c", globals.RootFlagConfigPathDefault,
				"Config search directory. "+
					"Expected to contain a '"+configName+"' config file "+
					"with any supported extension,\nexamples: "+
					configName+".<"+
					strings.Join(viper.SupportedExts, "|")+
					">"),
			flagAbsolutePath: globals.RootFlags.StringP(
				"config-file", "f", "",
				"Absolute path to config file to use. "+
					"Takes precedence over -c, --config-path"),
		}
	}
}

// Viper implements Source[T]
// It returns a fresh *Viper with opts to read from using a [Provider[T]].
func (s *SourceFile[T]) Viper(
	opts ...viper.Option,
) *viper.Viper {
	// Construct viper using passed default options
	v := viper.NewWithOptions(
		opts...,
	)

	// strip extension if given and not using absConfigFile
	ext := filepath.Ext(s.configName)
	if len(ext) > 0 && len(*s.flagAbsolutePath) == 0 {
		s.log.Warn("removing extension from config-name",
			"config-name", s.configName,
			"extension", ext,
		)
		s.configName = s.configName[:len(s.configName)-len(ext)]
	}

	// environment overrides
	s.log.Debug("enabling config env replacer",
		"env-prefix", s.flagEnvPrefix,
	)
	v.AutomaticEnv()
	v.SetEnvPrefix(*s.flagEnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(
		".", "_",
		"-", "_",
	))

	if len(*s.flagAbsolutePath) > 0 {
		// use this file explicitly
		s.log.Debug("using explicit config file",
			"filepath", *s.flagAbsolutePath)

		v.SetConfigFile(*s.flagAbsolutePath)

	} else {
		s.log.Debug("using auto-search of config file",
			"config-name", s.configName)

		// auto-detect and search for config filename without extension
		v.SetConfigName(s.configName)

		// user flag provided search path
		if len(*s.flagConfigPath) > 0 {
			s.log.Debug("adding flag provided config-path to search",
				"path", *s.flagConfigPath)
			v.AddConfigPath(*s.flagConfigPath)
		}
		// in-code provided search paths
		// (either default paths or developer provided)
		for _, path := range s.searchPaths {
			v.AddConfigPath(path)
		}
	}

	return v
}
