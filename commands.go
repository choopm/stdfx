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

package stdfx

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/globals"
	"github.com/earthboundkid/versioninfo/v2"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

// AppVersion is the version given to [VersionCommand]
var AppVersion = "unknown"

// VersionCommand a version *cobra.Command constructor to print version information.
// Supply your build tag as version and it will add runtime and compiler details.
func VersionCommand(version string) func(log *slog.Logger) *cobra.Command {
	if version != "" {
		AppVersion = "unknown"
	}

	return func(log *slog.Logger) *cobra.Command {
		cmd := &cobra.Command{
			Use:   "version",
			Short: "print version and exit",
			Run: func(cmd *cobra.Command, args []string) {
				log.Info("build info",
					slog.String("short", versioninfo.Short()),
					slog.String("revision", versioninfo.Revision),
					slog.Time("last-commit", versioninfo.LastCommit),
					slog.Bool("dirty-build", versioninfo.DirtyBuild),
					slog.String("go-version", runtime.Version()),
					slog.String("go-os", runtime.GOOS),
					slog.String("go-arch", runtime.GOARCH),
					slog.String("version", AppVersion),
				)
			},
		}

		// add a flag
		versionFlag := globals.RootFlags.BoolP("version", "v",
			false, "print version and exit")

		// add a hook to print version and quit
		globals.RootPreRuns = append(globals.RootPreRuns,
			func(rootCmd *cobra.Command, args []string) {
				if !*versionFlag {
					return
				}

				// hijack run funcs of root command
				rootCmd.Run = cmd.Run
				rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
					cmd.Run(cmd, args)
					return nil
				}
			})

		return cmd
	}
}

// ConfigCommand is a *cobra.Command constructor to print, modify and validate config.
func ConfigCommand[T any](
	log *slog.Logger,
	configProvider configfx.Provider[T],
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "print, modify or validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// show subcommand
	showCmd := &cobra.Command{
		Use:     "show",
		Aliases: []string{"print"},
		Short:   "print and show configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configProvider.Config()
			if err != nil {
				return err
			}
			v := configProvider.Viper()

			log.Info("configuration",
				slog.String("file", v.ConfigFileUsed()),
				slog.Any("parsed", cfg))
			return nil
		},
	}
	cmd.AddCommand(showCmd)

	// get subcommand
	getCmd := &cobra.Command{
		Use:   "get [key]...",
		Short: "get value(s) by key from configuration",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := configProvider.Config()
			if err != nil {
				return err
			}
			v := configProvider.Viper()

			// get values
			attrs := []any{}
			for _, key := range args {
				value := v.Get(key)
				attrs = append(attrs, slog.Any(key, value))
			}

			log.Info("read configuration", attrs...)
			return nil
		},
	}
	cmd.AddCommand(getCmd)

	// set subcommand
	setCmd := &cobra.Command{
		Use:   "set [key=value]...",
		Short: "set value(s) by key from configuration",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := configProvider.Config()
			if err != nil {
				return err
			}
			v := configProvider.Viper()

			// update state
			attrs := []any{}
			for _, arg := range args {
				key, value, found := strings.Cut(arg, "=")
				if !found {
					return fmt.Errorf("invalid syntax in %q, use key=value", arg)
				}
				v.Set(key, value)
				attrs = append(attrs, slog.Any(key, value))
			}

			// persist changes
			err = v.WriteConfig()
			if err != nil {
				return err
			}

			log.Info("updated configuration", attrs...)
			return nil
		},
	}
	cmd.AddCommand(setCmd)

	// validate subcommand
	validateCmd := &cobra.Command{
		Use:     "validate",
		Aliases: []string{"test"},
		Short:   "test or validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// validate viper parsing
			v := configProvider.Viper()
			err := v.ReadInConfig()
			if err != nil {
				return err
			}

			// more strict config parsing
			b, err := os.ReadFile(v.ConfigFileUsed())
			if err != nil {
				return err
			}
			switch t := strings.ToLower(filepath.Ext(v.ConfigFileUsed())); t {
			case "yaml":
				// more strict yaml parsing by using k8s parser:
				log.Debug("using strict yaml parser",
					slog.String("type", t))
				err := yaml.Unmarshal(b, &struct{}{})
				if err != nil {
					return err
				}
			default:
				log.Debug("missing strict parser for config",
					slog.String("type", t))
			}

			// validate config hook
			cfg, err := configProvider.Config()
			if err != nil {
				return err
			}
			if ctype, ok := any(cfg).(configfx.CustomValidator); ok {
				// T implements CustomValidator and therefore
				// has a custom func Validate(), use it:
				log.Debug("found custom config Validate()")
				if err := ctype.Validate(); err != nil {
					return err
				}
			}

			log.Info("configuration ok",
				slog.String("file", v.ConfigFileUsed()))
			return nil
		},
	}
	cmd.AddCommand(validateCmd)

	return cmd
}
