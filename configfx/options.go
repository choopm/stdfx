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

package configfx

import "github.com/fsnotify/fsnotify"

// configOptions stores options for With*() funcs
type configOptions struct {
	readInConfig   bool
	overlays       []*Overlay
	onConfigChange func(in fsnotify.Event)
}

// ConfigOption is a func to adjust options of *configOptions for later
// usage during [Config].
type ConfigOption func(*configOptions)

// defaultConfigOptions returns the default *configOptions
func defaultConfigOptions() *configOptions {
	opts := &configOptions{
		overlays:       make([]*Overlay, 0),
		onConfigChange: nil,
	}

	WithReadInConfig(true)(opts)

	return opts
}

// WithReadInConfig will use viper.ReadInConfig during [Config] invocation.
//
// Turning this off is useful when unmarshalling a config for the second time
// after merging anything into the backing viper config
// (otherwise the MergedInConfig would be lost).
func WithReadInConfig(value bool) ConfigOption {
	return func(o *configOptions) {
		o.readInConfig = value
	}
}

// WithOverlays adds the given overlays which will be injected
// during configuration parsing.
// This allows to split a config file into multiple files.
func WithOverlays(overlays ...*Overlay) ConfigOption {
	return func(o *configOptions) {
		o.overlays = append(o.overlays, overlays...)
	}
}

// WithOnConfigChange adds the callback to all viper instances.
// This callback will be invoked whenever there is a config change.
func WithOnConfigChange(callback func(in fsnotify.Event)) ConfigOption {
	return func(o *configOptions) {
		o.onConfigChange = callback
	}
}
