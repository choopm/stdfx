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
	"errors"
	"log/slog"
	"os"
)

var (
	// ErrRunningAsRoot can be returned by [Unprivileged]
	ErrRunningAsRoot = errors.New("running as root is dangerous and prohibited")
)

// Unprivileged returns an error if being run as root.
// This takes effect whenever the real or effective user id
// of the current user process is 0.
func Unprivileged() error {
	if os.Getuid() == 0 || os.Geteuid() == 0 {
		return ErrRunningAsRoot
	}
	return nil
}

// UnprivilegedWarn warns if being run as root.
// This takes effect whenever the real or effective user id
// of the current user process is 0.
func UnprivilegedWarn(log *slog.Logger) {
	if Unprivileged() != nil {
		log.Warn("running as root is dangerous")
	}
}
