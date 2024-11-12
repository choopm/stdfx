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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/choopm/stdfx/globals"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

const (
	// startBackoff defines the time frame to capture errors during startup
	// when using [Commander]
	startBackoff = 1 * time.Second
)

// AutoRegister annotates a *cobra.Command constructor f to be
// automatically registered as a sub command in NewRootCommand.
// Usage example:
//
//	fx.Provide(
//		stdfx.AutoRegister(firstCommandConstructor),
//		stdfx.AutoRegister(secondCommandConstructor),
//		stdfx.AutoCommand,
//	),
//	fx.Invoke(stdfx.Commander),
func AutoRegister(f any) any {
	return fx.Annotate(
		f,
		fx.ResultTags(`group:"commands"`),
	)
}

// AutoCommand is an annotated version of NewRootCommand which
// passes anything previously called with AutoRegister to an
// annotated version of NewRootCommand.
// Usage example:
//
//	fx.Provide(
//		stdfx.AutoRegister(firstCommandConstructor),
//		stdfx.AutoRegister(secondCommandConstructor),
//		stdfx.AutoCommand,
//	),
//	fx.Invoke(stdfx.Commander),
var AutoCommand = fx.Annotate(
	newRootCommand,
	fx.ParamTags(`group:"commands"`),
)

// newRootCommand provides a root command which adds any provided
// commands as child commands.
// Starting the root command will print the help page.
// Any globalFlags from ConfigSource implementations will be merged.
// It is up to the developer to provide meaningful subcommands.
func newRootCommand(commands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// add global RootFlags, can be filled by ConfigSource
	cmd.PersistentFlags().AddFlagSet(globals.RootFlags)

	// add global PreRuns, can be filled by commands
	cmd.PreRun = func(cmd *cobra.Command, args []string) {
		for _, cb := range globals.RootPreRuns {
			cb(cmd, args)
		}
	}

	// add commands
	for _, c := range commands {
		cmd.AddCommand(c)
	}

	return cmd
}

// Commander can be used as a *cobra.Command invoker for fx.
// It will start cmd with Context.Background() in a goroutine.
// It is typically used as last Invoke option in an fx.App to actually
// start the application using a previously provided root *cobra.Command.
// The started *cobra.Command shall use cmd.Context() to watch for Done().
// The ctx of cmd.Context() will be cancelled when it is time to shutdown.
// Failure to track cmd.Context() will kill your application after
// [fx.DefaultTimeout] - 15 seconds.
// fx.Lifecycle and fx.Shutdowner are injected into cmd.Context()
// and can be retrieved by calling [ExtractFromContext].
func Commander(
	lc fx.Lifecycle,
	shutdowner fx.Shutdowner,
	cmd *cobra.Command,
) {

	// errgroup and ctx to start/stop the *cobra.Command
	ctx := withShutdowner(context.Background(), shutdowner)
	ctx, cancel := context.WithCancel(ctx)
	g, ctx := errgroup.WithContext(ctx)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			// start the *cobra.Command using the errgroup and its ctx
			g.Go(func() error {
				_, err := cmd.ExecuteContextC(ctx)
				if err != nil && !errors.Is(err, context.Canceled) {
					defer shutdowner.Shutdown(fx.ExitCode(1)) // nolint:errcheck
					return fmt.Errorf("failed to run: %s", err)
				}
				return shutdowner.Shutdown()
			})

			// wait up to startBackoff for any error to be captured in ctx
			// otherwise the goroutine is considered up and running
			select {
			case <-ctx.Done():
				return g.Wait()

			case <-time.After(startBackoff):
				return nil
			}
		},
		OnStop: func(_ context.Context) error {
			// cancel the errgroup and wait for shutdown to finish
			cancel()
			return g.Wait()
		},
	})
}
