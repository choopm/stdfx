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

// AtLevel takes a *slog.Logger and returns a new *slog.Logger
// which logs everything to the requested level instead.
func AtLevel(log *slog.Logger, level slog.Level) *slog.Logger {
	return AtLevelMap(
		log,
		map[slog.Level]slog.Level{
			slog.LevelDebug: level,
			slog.LevelInfo:  level,
			slog.LevelWarn:  level,
			slog.LevelError: level,
		},
	)
}

// AtLevelMap takes a *slog.Logger and returns a new *slog.Logger
// which logs everything to the level mapped by level instead.
func AtLevelMap(log *slog.Logger, levels map[slog.Level]slog.Level) *slog.Logger {
	return slog.New(&slogLevelRedirect{
		Logger: log,
		m:      levels,
	})
}

// slogLevelRedirect wraps a *slog.Logger rewriting log levels of logged messages
type slogLevelRedirect struct {
	*slog.Logger
	m map[slog.Level]slog.Level
}

func (s *slogLevelRedirect) Enabled(ctx context.Context, level slog.Level) bool {
	return s.Logger.Handler().Enabled(ctx, level)
}

func (s *slogLevelRedirect) Handle(ctx context.Context, record slog.Record) error {
	// rewrite level
	record.Level = s.m[record.Level]
	return s.Logger.Handler().Handle(ctx, record)
}

func (s *slogLevelRedirect) WithAttrs(attrs []slog.Attr) slog.Handler {
	return s.Logger.Handler().WithAttrs(attrs)
}

func (s *slogLevelRedirect) WithGroup(name string) slog.Handler {
	return s.Logger.Handler().WithGroup(name)
}
