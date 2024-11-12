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
	"fmt"
	"log/slog"
	"sync"

	"github.com/choopm/stdfx/loggingfx/slogfx"
	"github.com/creasty/defaults"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Provider defines an interface for abstract config providers
type Provider[T any] interface {
	// Config shall return the generic config or error
	Config() (*T, error)
	// Viper shall return the viper instance
	Viper() *viper.Viper
}

// providerImpl implements Provider[T]
type providerImpl[T any] struct {
	source Source[T]
	log    *slog.Logger

	viper      *viper.Viper
	viperMutex sync.Mutex
}

// ensure providerImpl[T] implements Provider[T]
var _ Provider[any] = &providerImpl[any]{}

// NewProvider returns a config provider to fetch the config.
// Internally the config source is provided by viper and parsed the
// moment one does call Provider[T].Config().
func NewProvider[T any](
	source Source[T], // construct using [NewSourceFile]
	log *slog.Logger, // logger for use with viper of source
) Provider[T] {
	return &providerImpl[T]{
		source: source,
		log:    log.With(slog.String("context", "config-provider")),
	}
}

// Config returns the decoded config *T or error.
// Config decoding can be tuned by implementing [CustomConfigDecoder].
// Internally it requests a Viper instance from the ConfigSource[T]
// to then unmarshall it onto *T using mapstructure and default tags.
func (s *providerImpl[T]) Config() (*T, error) {
	// create fresh generic config
	t := new(T)

	// set default values by struct tags `default:""` on t
	// viper will override what is present afterwards
	s.log.Debug("setting defaults")
	if err := defaults.Set(t); err != nil {
		return nil, fmt.Errorf("setting config defaults: %s", err)
	}

	// build default decoders
	decoders := DefaultDecoders()
	// check if T implements CustomDecoder
	if ctype, ok := any(t).(CustomDecoder); ok {
		// T implements CustomDecoder and therefore
		// has a custom func DecodeHook(), use it:
		s.log.Debug("found custom config DecodeHook()")
		decoders = append(decoders, ctype.DecodeHook())
	}

	// get viper instance
	v := s.Viper()

	// let viper read the config from source
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %s", err)
	}

	// decode config using viper and struct tags `mapstructure:""`
	s.log.Debug("unmarshalling config using viper")
	err := v.Unmarshal(t, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(decoders...),
	))
	if err != nil {
		return nil, fmt.Errorf("unmarshal config: %s", err)
	}

	return t, nil
}

// Viper returns the viper instance.
// Internally it requests a Viper instance from the ConfigSource[T]
// if it was missing in s before.
func (s *providerImpl[T]) Viper() *viper.Viper {
	s.viperMutex.Lock()
	defer s.viperMutex.Unlock()

	// check if it was constructed before
	if s.viper != nil {
		return s.viper
	}

	// build viper instance using default options
	// for all config sources
	vOpts := []viper.Option{
		// viper logs using Info by default, therefore we wrap
		// it into a separate logger which logs to debug instead
		viper.WithLogger(slogfx.AtLevel(
			s.log.With(slog.String("context", "viper")),
			slog.LevelDebug,
		)),
	}
	s.viper = s.source.Viper(vOpts...)

	return s.viper
}
