/*
Copyright 2025 Christoph Hoopmann

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
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/choopm/stdfx/globals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/fx"
)

// ContainerEntrypointDefaultTools are the default tools for [ContainerEntrypoint]
var ContainerEntrypointDefaultTools = []string{"sh", "/bin/sh", "bash", "/bin/bash"}

// ContainerEntrypoint might be used with [fx.Invoke] and tooling when the calling
// go program is packaged into a container where it is used as the entrypoint.
// This will execute any value of `tools` when given as the first argument and if found in $PATH.
// If tools is empty it will use a default list: [ContainerEntrypointDefaultTools].
// A special value of '*' allows for any tool.
// There is extra handling when the first argument is the binary name itself:
// For such cases that argument is silbently shifted out and execution continues.
// A first argument naming a command registered with [AutoRegister], or one of
// its aliases, is never executed as a tool: a host may well have a tool of the
// same name in $PATH (RHEL ships /usr/bin/setup), and the command is what the
// user asked for.
//
// The first argument is the first one following the root flags given ahead of
// it, as an image entrypoint `myproject -c /etc/myproject` gives them: they are
// skipped to find it, kept when the binary name is shifted out, and dropped when
// a tool runs, as they belong to the binary. A flag of [globals.RootFlags]
// takes the argument following it as its value unless it is boolean or given
// as --flag=value; any other flag is taken to have no value.
//
// Example usage:
//   - fx.Invoke(stdfx.ContainerEntrypoint())
//   - fx.Invoke(stdfx.ContainerEntrypoint("sh", "bash", "whoami"))
//   - fx.Invoke(stdfx.ContainerEntrypoint("*"))
//
// Resulting container image invocations:
//   - docker run --rm -it ghcr.io/choopm/myproject:latest sh -c 'echo hello world'
//   - docker run --rm -it ghcr.io/choopm/myproject:latest bash -i
//   - docker run --rm -it ghcr.io/choopm/myproject:latest whoami
//   - docker run --rm -it ghcr.io/choopm/myproject:latest myproject -c ...
func ContainerEntrypoint(tools ...string) func(entrypointParams) {
	// use default tools if nothing was provided
	if len(tools) == 0 {
		tools = ContainerEntrypointDefaultTools
	}

	// return constructor
	return func(p entrypointParams) {
		// the first argument, following the root flags given ahead of it
		first := 1 + leadingFlags(os.Args[1:])
		if first >= len(os.Args) {
			// only care when atleast one argument was given to cli
			return
		}
		arg := os.Args[first]

		// a registered command takes precedence over a tool of the same name
		if slices.ContainsFunc(p.Commands, func(cmd *cobra.Command) bool {
			return cmd.Name() == arg || cmd.HasAlias(arg)
		}) {
			return
		}

		wildcardTool := slices.Contains(tools, "*")

		// container image argument handling
		switch {
		case arg == filepath.Base(os.Args[0]):
			// First argument is the same as binary name -> remove it, keeping
			// the flags ahead of it, continue
			os.Args = slices.Delete(os.Args, first, first+1)

		case wildcardTool || slices.Contains(tools, arg):
			// Chain to the first argument given by looking it up in $PATH.
			path, err := exec.LookPath(arg)
			if err != nil && wildcardTool {
				// wildcard tool is allowed, so the failing lookup might be
				// caused by first argument not being any tool, continue
				break
			} else if err != nil {
				panic(err)
			}
			// the flags ahead of it are the binary's, never the tool's
			err = syscall.Exec(path, os.Args[first:], syscall.Environ())
			if err != nil {
				panic(err)
			}
		}
	}
}

// leadingFlags returns how many of args are root flags and their values given
// ahead of the first argument which is neither, see [ContainerEntrypoint].
func leadingFlags(args []string) int {
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "-" || arg == "--" || !strings.HasPrefix(arg, "-") {
			return i
		}
		i++

		// --flag=value and -fvalue carry their value in the same argument
		if strings.Contains(arg, "=") || (!strings.HasPrefix(arg, "--") && len(arg) > 2) {
			continue
		}

		var flag *pflag.Flag
		if name, long := strings.CutPrefix(arg, "--"); long {
			flag = globals.RootFlags.Lookup(name)
		} else {
			flag = globals.RootFlags.ShorthandLookup(arg[1:])
		}

		// a boolean flag has a value to fall back to when given without one
		if flag != nil && flag.NoOptDefVal == "" && i < len(args) {
			i++
		}
	}

	return i
}

// entrypointParams are the dependencies of [ContainerEntrypoint]
type entrypointParams struct {
	fx.In

	// Commands are the commands registered using [AutoRegister]
	Commands []*cobra.Command `group:"commands"`
}
