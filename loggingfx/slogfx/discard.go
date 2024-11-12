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

package slogfx

import (
	"context"
	"log/slog"
)

// DiscardHandler returns a slog.Handler to discard loggings.
//
// TODO:
// replace discardHandler when this is implemented:
// https://github.com/golang/go/issues/62005
// https://go-review.googlesource.com/c/go/+/626486
// subject to go 1.24
func DiscardHandler() slog.Handler {
	return &discardHandler{}
}

// discardHandler discards all log output.
// discardHandler.Enabled returns false for all Levels.
type discardHandler struct{}

func (dh discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (dh discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (dh discardHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return dh }
func (dh discardHandler) WithGroup(_ string) slog.Handler               { return dh }
