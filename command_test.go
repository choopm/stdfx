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

package stdfx_test

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/choopm/stdfx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// The tests below replace os.Args, which is why none of them runs in parallel.

// startWindow is the time frame Commander holds a running command to, which
// is unexported.
const startWindow = time.Second

// commanderApp returns an app running run as the command it is invoked with,
// as a user would invoke it.
func commanderApp(t *testing.T, run func(cmd *cobra.Command, args []string) error) *fx.App {
	t.Helper()

	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{os.Args[0], "run"}

	return fx.New(
		fx.NopLogger,
		fx.Provide(
			stdfx.AutoRegister(func() *cobra.Command {
				cmd := &cobra.Command{Use: "run", RunE: run, SilenceErrors: true}
				cmd.SetOut(io.Discard)
				cmd.SetErr(io.Discard)
				return cmd
			}),
			stdfx.AutoCommand,
		),
		fx.Invoke(stdfx.Commander),
	)
}

// exitCode returns the exit code app is shut down with, or fails t when it is
// not shut down in time.
func exitCode(t *testing.T, app *fx.App) int {
	t.Helper()

	select {
	case sig := <-app.Wait():
		return sig.ExitCode
	case <-time.After(5 * time.Second):
		t.Fatal("app was not shut down")
		return -1
	}
}

func TestCommanderReturnsOnceCommandFinished(t *testing.T) {
	app := commanderApp(t, func(*cobra.Command, []string) error {
		return nil
	})

	began := time.Now()
	require.NoError(t, app.Start(t.Context()))
	assert.Less(t, time.Since(began), startWindow/2,
		"a finished command ends the wait for its start")

	assert.Equal(t, 0, exitCode(t, app))
	require.NoError(t, app.Stop(t.Context()))
}

func TestCommanderFailsStartOfFailingCommand(t *testing.T) {
	app := commanderApp(t, func(*cobra.Command, []string) error {
		return errors.New("initial work failed")
	})

	began := time.Now()
	err := app.Start(t.Context())
	require.ErrorContains(t, err, "initial work failed")
	assert.Less(t, time.Since(began), startWindow/2)
}

func TestCommanderHoldsRunningCommandToWindow(t *testing.T) {
	app := commanderApp(t, func(cmd *cobra.Command, _ []string) error {
		<-cmd.Context().Done()
		return nil
	})

	began := time.Now()
	require.NoError(t, app.Start(t.Context()))
	assert.GreaterOrEqual(t, time.Since(began), startWindow)

	require.NoError(t, app.Stop(t.Context()))
}

func TestCommanderExitsWithFailureAfterWindow(t *testing.T) {
	app := commanderApp(t, func(cmd *cobra.Command, _ []string) error {
		select {
		case <-cmd.Context().Done():
			return nil
		case <-time.After(startWindow + startWindow/2):
			return errors.New("failed while running")
		}
	})

	require.NoError(t, app.Start(t.Context()))
	assert.Equal(t, 1, exitCode(t, app))
	assert.ErrorContains(t, app.Stop(t.Context()), "failed while running")
}
