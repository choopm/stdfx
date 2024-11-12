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

package configfx

import (
	"github.com/choopm/stdfx/configfx/decoders"
	"github.com/go-viper/mapstructure/v2"
)

// CustomDecoder denotes types which implement a custom DecodeHook()
// for use with viper and mapstructure struct tags.
type CustomDecoder interface {
	// DecodeHook shall return the func to be used for decoding.
	// It will be appended to default decoders.
	DecodeHook() mapstructure.DecodeHookFunc
}

// DefaultDecoders returns common decoders to be used with config parsers
func DefaultDecoders() []mapstructure.DecodeHookFunc {
	decoders := []mapstructure.DecodeHookFunc{
		// viper defaults
		// mapstructure.StringToTimeDurationHookFunc(), // replaced
		mapstructure.StringToSliceHookFunc(","),

		// decoders from subpackage
		decoders.Duration(), // replaces StringToTimeDurationHookFunc
	}

	return decoders
}
