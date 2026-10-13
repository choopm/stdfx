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
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/choopm/stdfx/loggingfx/slogfx"
	"github.com/creasty/defaults"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

// Provider defines an interface for abstract config providers
type Provider[T any] interface {
	// Config shall return the generic config or error
	Config(opts ...ConfigOption) (*T, error)
	// Viper shall return the viper instance
	Viper() *viper.Viper
}

// providerImpl implements Provider[T]
type providerImpl[T any] struct {
	source Source[T]
	log    *slog.Logger

	viper      *viper.Viper
	viperMutex sync.Mutex

	viperWatchOnce sync.Once
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
func (s *providerImpl[T]) Config(opts ...ConfigOption) (*T, error) {
	// apply any given opts
	cOpts := defaultConfigOptions()
	for _, option := range opts {
		option(cOpts)
	}

	// create fresh generic config with defaults
	s.log.Debug("setting defaults")
	t, err := newConfig[T]()
	if err != nil {
		return nil, err
	}

	// get viper instance
	v := s.Viper()
	if cOpts.onConfigChange != nil {
		v.OnConfigChange(cOpts.onConfigChange)
		s.viperWatchOnce.Do(v.WatchConfig)
	}

	if cOpts.readInConfig {
		// let viper read the config from source
		if err := v.ReadInConfig(); err != nil {
			s.releaseViper()
			return nil, fmt.Errorf("read config: %s", err)
		}
	}

	// apply any overlays
	for _, overlay := range cOpts.overlays {
		if err := overlay.applyTo(v, t); err != nil {
			return nil, fmt.Errorf("apply overlay: %s", err)
		}
		if cOpts.onConfigChange != nil {
			overlay.viper.OnConfigChange(cOpts.onConfigChange)
			overlay.viperWatchOnce.Do(overlay.viper.WatchConfig)
		}
	}

	// decode config using viper and struct tags `mapstructure:""`
	s.log.Debug("unmarshalling config using viper")
	if err := unmarshal(v, t); err != nil {
		s.releaseViper()
		return nil, err
	}

	return t, nil
}

// Decode returns the config *T decoded from b or error.
// It applies defaults, decoders and overlays as [Provider.Config] does, but it
// never consults the environment: what it returns is what b says, which is what
// an editor of the config file needs to validate before writing it.
// path names the file b was read from, if any. Its extension selects the
// format, yaml if it has none, and its directory is where overlays are found.
// Of opts only overlays apply.
func Decode[T any](path string, b []byte, opts ...ConfigOption) (*T, error) {
	cOpts := defaultConfigOptions()
	for _, option := range opts {
		option(cOpts)
	}

	t, err := newConfig[T]()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if path != "" {
		v.SetConfigFile(path)
		if ext := strings.TrimPrefix(filepath.Ext(path), "."); ext != "" {
			v.SetConfigType(ext)
		}
	}
	if err := v.ReadConfig(bytes.NewReader(b)); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	for _, overlay := range cOpts.overlays {
		if err := overlay.applyTo(v, t); err != nil {
			return nil, fmt.Errorf("apply overlay: %w", err)
		}
	}

	if err := unmarshal(v, t); err != nil {
		return nil, err
	}

	return t, nil
}

// newConfig returns a fresh *T with the defaults of its struct tags
// `default:""` set, which a config file overrides afterwards, or error.
func newConfig[T any]() (*T, error) {
	t := new(T)
	if err := defaults.Set(t); err != nil {
		return nil, fmt.Errorf("setting config defaults: %s", err)
	}

	return t, nil
}

// unmarshal decodes what v read onto t using the struct tags
// `mapstructure:""`, the [DefaultDecoders] and the DecodeHook of t if it is a
// [CustomDecoder], or returns error.
func unmarshal[T any](v *viper.Viper, t *T) error {
	decoders := DefaultDecoders()
	if ctype, ok := any(t).(CustomDecoder); ok {
		// T implements CustomDecoder and therefore
		// has a custom func DecodeHook(), use it:
		decoders = append(decoders, ctype.DecodeHook())
	}

	err := v.Unmarshal(t, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(decoders...),
	))
	if err != nil {
		return fmt.Errorf("unmarshal config: %s", err)
	}

	return nil
}

// releaseViper should be called when viper needs to be freed after errors.
// This might be the case for any decorator attempt on reading the config,
// thus blocking future parsing attempts after e.g. cobra flags have been read.
func (s *providerImpl[T]) releaseViper() {
	s.viperMutex.Lock()
	defer s.viperMutex.Unlock()

	s.viper = nil
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
