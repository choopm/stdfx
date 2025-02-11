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

package zapfx

import (
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/loggingfx"
	"go.uber.org/zap"
)

// Decorator is a fx.Decorate constructor to decorate logger to use
// settings found in config for all configs implementing [ConfigWithLogging].
//
// The decorator will silently discard any errors since it is only decorating:
// A user could run version command without providing a valid config path.
// In such a case config file parsing would fail hence why errors are ignored.
func Decorator[T any](
	configProvider configfx.Provider[T],
	logger *zap.Logger,
) (*zap.Logger, error) {
	cfg, err := configProvider.Config()
	if err != nil {
		return logger, nil
	}

	// check if cfg implements ConfigWithLogging
	if ctype, ok := any(cfg).(loggingfx.ConfigWithLogging); ok {
		// cfg implements ConfigWithLogging and therefore
		// has a custom func LoggingConfig(), use it to decorate:
		log, err := New(ctype.LoggingConfig())
		if err != nil {
			return logger, nil
		}

		return log, nil
	}

	// not implementing, so return as it is
	return logger, nil
}
