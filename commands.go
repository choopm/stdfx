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
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/configfx/configdoc"
	"github.com/choopm/stdfx/globals"
	"github.com/earthboundkid/versioninfo/v2"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"sigs.k8s.io/yaml"
)

// AppVersion is the version given to [VersionCommand]
var AppVersion = "unknown"

// VersionCommand a version *cobra.Command constructor to print version information.
// Supply your build tag as version and it will add runtime and compiler details.
func VersionCommand(version string) func(log *slog.Logger) *cobra.Command {
	if version != "" {
		AppVersion = version
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

// ConfigCommandParams are the dependencies of [ConfigCommand].
type ConfigCommandParams[T any] struct {
	fx.In

	// Docs document the config file. Without them, set and format write
	// the file without comments.
	Docs *configdoc.Docs `optional:"true"`

	// Log is the logger to use
	Log *slog.Logger

	// Provider provides the config
	Provider configfx.Provider[T]
}

// ConfigCommand is a *cobra.Command constructor to print, modify and validate config.
// Supply [configdoc.Docs] to fx to have set and format comment the file:
//
//	fx.Supply(mypkg.Docs()),
func ConfigCommand[T any](p ConfigCommandParams[T]) *cobra.Command {
	log, configProvider := p.Log, p.Provider
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
		Long: "set value(s) by key from configuration.\n" +
			"A value is yaml, so a list is written as [a, b].\n" +
			"The file is rewritten in the layout of the config, keeping its values;\n" +
			"environment overrides are never written into it.",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, doc, err := readConfigDocument(configProvider)
			if err != nil {
				return err
			}

			// update state
			attrs := []any{}
			for _, arg := range args {
				key, value, found := strings.Cut(arg, "=")
				if !found {
					return fmt.Errorf("invalid syntax in %q, use key=value", arg)
				}
				node, err := configdoc.ParseValue(value)
				if err != nil {
					return fmt.Errorf("value of %s: %w", key, err)
				}
				if err := doc.Set(node, strings.Split(key, ".")...); err != nil {
					return err
				}
				attrs = append(attrs, slog.Any(key, value))
			}

			// persist changes
			if _, err := writeConfigDocument[T](log, p.Docs, path, doc, false); err != nil {
				return err
			}

			log.Info("updated configuration", attrs...)
			return nil
		},
	}
	cmd.AddCommand(setCmd)

	// format subcommand
	print := false
	formatCmd := &cobra.Command{
		Use:   "format",
		Short: "rewrite configuration in the layout of the config",
		Long: "rewrite configuration in the layout of the config.\n" +
			"Settings follow the order of the config, each under its documentation;\n" +
			"values are kept as written, missing settings are added with their default\n" +
			"and keys nothing reads are dropped.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, doc, err := readConfigDocument(configProvider)
			if err != nil {
				return err
			}

			out, err := writeConfigDocument[T](log, p.Docs, path, doc, print)
			if err != nil {
				return err
			}
			if print {
				_, err := cmd.OutOrStdout().Write(out)
				return err
			}

			log.Info("formatted configuration", slog.String("file", path))
			return nil
		},
	}
	formatCmd.Flags().BoolVar(&print, "print", print, "print the result instead of writing it")
	cmd.AddCommand(formatCmd)

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
			// the extension carries its dot, the name of a type does not
			switch t := strings.TrimPrefix(strings.ToLower(filepath.Ext(v.ConfigFileUsed())), "."); t {
			case "yaml", "yml":
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

// readConfigDocument returns the path of the config file of provider and its
// document, or error. What the environment overrides is not part of it.
func readConfigDocument[T any](provider configfx.Provider[T]) (string, *configdoc.Document, error) {
	v := provider.Viper()
	if err := v.ReadInConfig(); err != nil {
		return "", nil, err
	}

	path := v.ConfigFileUsed()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	doc, err := configdoc.Parse(b)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", path, err)
	}

	return path, doc, nil
}

// writeConfigDocument renders doc as docs say, makes sure it decodes into a T
// and writes it to path unless dryRun is true, and returns what it rendered,
// or error. A file which would not change is not written.
func writeConfigDocument[T any](log *slog.Logger, docs *configdoc.Docs, path string, doc *configdoc.Document, dryRun bool) ([]byte, error) {
	schema, err := configdoc.NewSchema[T](docs)
	if err != nil {
		return nil, err
	}

	out, report, err := schema.RenderDocument(doc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(report.Dropped) > 0 {
		log.Warn("dropping keys nothing reads",
			slog.String("file", path),
			slog.Any("keys", report.Dropped))
	}
	if len(report.Mismatched) > 0 {
		log.Warn("keeping values of an unexpected shape",
			slog.String("file", path),
			slog.Any("keys", report.Mismatched))
	}

	if _, err := configfx.Decode[T](path, out); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if dryRun {
		return out, nil
	}

	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, out) {
		return out, nil
	}

	return out, configdoc.WriteFile(path, out)
}
