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

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

// Overlay defines a configuration overlay
type Overlay struct {
	// Filename is the full filepath to the overlay config
	Filename string `mapstructure:"filename" default:""`

	// From is the mapstructure path to the element which shall be used
	From string `mapstructure:"from" default:""`

	// To defines mapstructure paths where the [From] element gets injected
	To []string `mapstructure:"to" default:"[]"`

	// viper is used internally to read and parse the overlay config file
	viper *viper.Viper

	// viperWatchOnce is used to only start one watcher
	viperWatchOnce sync.Once
}

// ApplyTo loads the overlay from the filesystem and
// merges it with vip *Viper and cfg or error.
// Overlay config files are searched using full- and relative to main config file path.
func (s *Overlay) applyTo(vip *viper.Viper, cfg any) error {
	// remove file extension
	extension := filepath.Ext(s.Filename)
	filename := s.Filename[0 : len(s.Filename)-len(extension)]

	// fresh viper to read in overlay
	s.viper = viper.New()
	s.viper.SetConfigName(filename)
	s.viper.AddConfigPath(filepath.Dir(vip.ConfigFileUsed()))
	s.viper.AddConfigPath(".")
	err := s.viper.ReadInConfig()
	if err != nil {
		return fmt.Errorf("reading overlay config %q failed: %s", s.Filename, err)
	}

	// retrieve the from key
	fromPath := strings.Split(s.From, ".")
	fromSlice := s.viper.AllSettings()
	var from any
	for _, elem := range fromPath {
		// retrieve path element
		var ok bool
		from, ok = fromSlice[elem]
		if !ok {
			return fmt.Errorf("referenced from field %q in path %q not found in overlay %q", elem, s.From, s.Filename)
		}

		// check if it is a map for next iter
		if cast, ok := from.(map[string]any); ok {
			fromSlice = cast
		}
	}
	// sanity check
	if from == nil {
		return fmt.Errorf("referenced from path %q is nil in overlay %q", s.From, s.Filename)
	}

	for _, path := range s.To {
		// forge a config from values inside overlay by adding the desired path in front
		forged := from
		apath := strings.Split(path, ".")
		for i := len(apath) - 1; i >= 0; i-- {
			key := apath[i]
			if strings.Contains(key, "[") {
				// whenever we encounter [] operator we need to parse it
				// this allows for syntax like this:
				//   to:
				//   - "policy.rules.[name=replace-subject].match.header.regex.[name=test].value"
				trimmed := strings.TrimRight(strings.TrimLeft(key, "["), "]")
				a, b, ok := strings.Cut(trimmed, "=")
				if a != "name" {
					return fmt.Errorf("[] operator in %q can only be used against name field", s.Filename)
				}
				if ok {
					// [a=b]
					v, ok := forged.(map[string]any)
					if !ok {
						return fmt.Errorf("[] operator in %q can only be used on map types", s.Filename)
					}

					// add the name=selector to existing map and wrap it inside a slice
					v[a] = b
					forged = []any{
						v,
					}
				}

			} else {
				// otherwise we can easily add it as a map path
				forged = map[string]any{
					key: forged,
				}
			}
		}
		mforged, ok := forged.(map[string]any)
		if !ok {
			return fmt.Errorf("merging overlay config %q failed due to map cast", s.Filename)
		}

		// using Kubernetes strategic merge patch from forged patch documents
		patch, err := strategicpatch.StrategicMergeMapPatch(
			vip.AllSettings(), mforged, cfg)
		if err != nil {
			return fmt.Errorf("building patch of overlay config %q failed: %s", s.Filename, err)
		}

		// merge into current viper configuration
		err = vip.MergeConfigMap(patch)
		if err != nil {
			return fmt.Errorf("merging overlay config %q failed: %s", s.Filename, err)
		}
	}

	return nil
}
