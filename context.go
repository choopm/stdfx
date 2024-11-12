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
	"context"
	"errors"

	"go.uber.org/fx"
)

type shutdownerContextKeyType struct{}
type shutdownerContextValue fx.Shutdowner

// shutdownerContextKey is used to inject fx.Shutdowner into Context
var shutdownerContextKey = &shutdownerContextKeyType{}

// ErrContextMissingShutdowner can be returned by [Shutdown]
var ErrContextMissingShutdowner = errors.New("context is missing shutdowner")

// withShutdowner injects shutdowner into ctx for use with [Shutdown]
func withShutdowner(
	ctx context.Context,
	shutdowner fx.Shutdowner,
) context.Context {
	return context.WithValue(
		ctx,
		shutdownerContextKey,
		shutdownerContextValue(shutdowner),
	)
}

// shutdownerFromContext returns a fx.Shutdowner from ctx or error
func shutdownerFromContext(ctx context.Context) (fx.Shutdowner, error) {
	v := ctx.Value(shutdownerContextKey)
	if v == nil {
		return nil, ErrContextMissingShutdowner
	}
	val, ok := v.(shutdownerContextValue)
	if !ok {
		return nil, ErrContextMissingShutdowner
	}
	return val, nil
}

// Shutdown uses fx.Shutdowner from ctx to shutdown a fx.App using exitCode.
// This works when [Commander] was used to start it.
// You can use this to shutdown an application unaware of fx during runtime.
// It might return [ErrContextMissingShutdowner] in which case it is up to you
// to directly call [os.Exit] or panic.
func Shutdown(ctx context.Context, exitCode int) error {
	shutdowner, err := shutdownerFromContext(ctx)
	if err != nil {
		return err
	}
	return shutdowner.Shutdown(fx.ExitCode(exitCode))
}
