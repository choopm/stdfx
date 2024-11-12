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
	"log/slog"

	"github.com/choopm/stdfx/configfx"
)

// ConfigFile provides your fx.App with a ConfigProvider[T] constructor.
// The provider - when being constructed - can be used to search for,
// read and unmarshal your config file to a struct of type *T or error.
// [configName] shall be the name of a config file without extension.
// Internally this curries both functions [config.NewSourceFile] and
// [config.NewProvider] for syntactic sugar.
// Usage example:
//
//	fx.Provide(stdfx.Config[mypkg.ConfStruct]("configname")),
//
// After providing as described above, you will be able to request
// this provider to fetch the actual config *struct like this:
//
//	func buildCommand(
//		provider stdfx.ConfigProvider[mypkg.ConfStruct],
//	) *cobra.Command {
//		// fetch the config of type *mypkg.ConfStruct
//		cfg, err := provider.Config()
//		if err != nil {
//			// ...
//		}
//		// do something using cfg (construct a *cobra.Command for example)
//	}
func ConfigFile[T any](
	configName string,
) func(log *slog.Logger) configfx.Provider[T] {
	return func(log *slog.Logger) configfx.Provider[T] {
		buildSource := configfx.NewSourceFile[T](configName)
		return configfx.NewProvider[T](
			buildSource(log),
			log,
		)
	}
}
