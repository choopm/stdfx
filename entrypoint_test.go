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
	"path/filepath"
	"testing"

	"github.com/choopm/stdfx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
)

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
