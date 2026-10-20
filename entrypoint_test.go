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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/globals"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

// Root flags as a config source registers them, given ahead of the first
// argument as an image entrypoint gives them: -d takes a value, -q does not.
var (
	_ = globals.RootFlags.StringP("entrypoint-dir", "d", "", "a root flag taking a value")
	_ = globals.RootFlags.BoolP("entrypoint-quiet", "q", false, "a boolean root flag")
)

// TestContainerEntrypointShiftsBinaryNameAfterFlags asserts that the binary
// name is found and shifted out behind root flags, which are kept.
func TestContainerEntrypointShiftsBinaryNameAfterFlags(t *testing.T) {
	for name, flags := range map[string][]string{
		"shorthand":         {"-d", "/etc/app"},
		"long":              {"--entrypoint-dir", "/etc/app"},
		"long with value":   {"--entrypoint-dir=/etc/app"},
		"shorthand joined":  {"-d/etc/app"},
		"boolean":           {"-q"},
		"boolean and value": {"-q", "-d", "/etc/app"},
	} {
		t.Run(name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			binary := oldArgs[0]
			os.Args = append(append([]string{binary}, flags...), filepath.Base(binary), "version")

			app := fx.New(fx.NopLogger, fx.Invoke(stdfx.ContainerEntrypoint("*")))
			require.NoError(t, app.Err())
			assert.Equal(t, append(append([]string{binary}, flags...), "version"), os.Args)
		})
	}
}

// TestContainerEntrypointPrefersCommandsAfterFlags asserts that a registered
// command behind root flags is not executed as the tool of the same name.
func TestContainerEntrypointPrefersCommandsAfterFlags(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "setup"), []byte("#!/bin/sh\nexit 3\n"), 0o755))
	t.Setenv("PATH", dir)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{oldArgs[0], "-d", "/etc/app", "setup"}

	app := fx.New(
		fx.NopLogger,
		fx.Provide(stdfx.AutoRegister(func() *cobra.Command {
			return &cobra.Command{Use: "setup"}
		})),
		fx.Invoke(stdfx.ContainerEntrypoint("*")),
	)
	require.NoError(t, app.Err())
	assert.Equal(t, []string{oldArgs[0], "-d", "/etc/app", "setup"}, os.Args)
}

// TestContainerEntrypointRunsToolWithoutFlags asserts that a tool behind root
// flags runs with its own arguments alone. Executing it replaces the process,
// so the test runs itself again as that process.
func TestContainerEntrypointRunsToolWithoutFlags(t *testing.T) {
	if os.Getenv("STDFX_ENTRYPOINT_TOOL") != "" {
		os.Args = []string{os.Args[0], "-d", "/etc/app", "-q", "tool", "foo", "bar"}
		fx.New(fx.NopLogger, fx.Invoke(stdfx.ContainerEntrypoint("*")))
		os.Exit(2) // reached only where no tool was executed
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "out")
	script := "#!/bin/sh\necho \"$@\" > " + out + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), []byte(script), 0o755))

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestContainerEntrypointRunsToolWithoutFlags$")
	cmd.Env = append(os.Environ(), "STDFX_ENTRYPOINT_TOOL=1", "PATH="+dir)
	require.NoError(t, cmd.Run())

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "foo bar\n", string(got))
}

// TestContainerEntrypointPrefersCommands asserts that a registered command or
// alias is not executed as the tool of the same name found in $PATH.
// Were it executed, the tool would replace the test binary and exit non-zero.
func TestContainerEntrypointPrefersCommands(t *testing.T) {
	for _, arg := range []string{"setup", "configure"} {
		t.Run(arg, func(t *testing.T) {
			dir := t.TempDir()
			for _, tool := range []string{"setup", "configure"} {
				script := filepath.Join(dir, tool)
				require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nexit 3\n"), 0o755))
			}
			t.Setenv("PATH", dir)

			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = []string{oldArgs[0], arg}

			app := fx.New(
				fx.NopLogger,
				fx.Provide(stdfx.AutoRegister(func() *cobra.Command {
					return &cobra.Command{Use: "setup", Aliases: []string{"configure"}}
				})),
				fx.Invoke(stdfx.ContainerEntrypoint("*")),
			)
			require.NoError(t, app.Err())
			assert.Equal(t, []string{oldArgs[0], arg}, os.Args)
		})
	}
}
