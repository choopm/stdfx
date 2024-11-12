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

package globals

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// RootFlags stores the global flags to be used in newRootCommand.
	// These flags might be filled by configfx.Source[T] implementations.
	RootFlags = pflag.NewFlagSet("root", pflag.ContinueOnError)

	// RootPreRuns will be added to root commands PreRun.
	// Use this to inject any code precommand start.
	RootPreRuns []func(cmd *cobra.Command, args []string)

	// RootFlagConfigPathDefault is the default value for config-path.
	// It is defined here to be modified during tests to fake arguments being passed.
	RootFlagConfigPathDefault = ""
)
