//go:build unix

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

package configdoc

import (
	"io/fs"
	"os"
	"syscall"
)

// chown gives f the owner and group of like, or only its group where
// groupOnly is true, or returns error. It does nothing where both already
// match, which is all a process not running as root could do.
func chown(f *os.File, like fs.FileInfo, groupOnly bool) error {
	stat, ok := like.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	uid := int(stat.Uid)
	if groupOnly {
		uid = os.Geteuid()
	}
	if uid == os.Getuid() && int(stat.Gid) == os.Getgid() {
		return nil
	}
	if os.Geteuid() != 0 {
		// only root gives a file away; the group of a new file follows the
		// directory or the process, either of which is what a user may write
		return nil
	}

	return f.Chown(uid, int(stat.Gid))
}
