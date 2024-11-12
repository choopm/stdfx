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

package decoders

import (
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/xhit/go-str2duration/v2"
)

// Duration returns a mapstructure.DecodeHookFunc which supports
// deoding time.Duration from strings in a format such as "4d3h2m1s".
// It extends the known mapstructure.StringToTimeDurationHookFunc
// to support days and weeks aswell.
func Duration() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(time.Duration(0)) {
			return data, nil
		}

		dur, err := str2duration.ParseDuration(data.(string))
		if err != nil {
			return nil, err
		}

		return dur, nil
	}
}
