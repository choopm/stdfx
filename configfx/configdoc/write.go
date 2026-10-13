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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

const (
	// BackupSuffix names the copy of a file as it was before it was last
	// written
	BackupSuffix = ".bak"

	// OriginalSuffix names the copy of a file as it was before it was first
	// written, which keeps the comments an operator wrote
	OriginalSuffix = ".orig"

	// newFileMode is the mode of a config file written for the first time.
	// It is not readable by others, because a config file holds secrets.
	newFileMode fs.FileMode = 0o640
)

// WriteFile replaces the file at path by data, or returns error.
//
// The file is replaced as a whole or not at all: data goes into a temporary
// file next to it which is then renamed over it, so that neither a crash nor a
// process reading the file meanwhile ever sees half of it. The new file keeps
// the mode, owner and group of the old one. A file which cannot be replaced
// because it is a mount point of its own, as a single file mounted into a
// container is, is written in place instead.
//
// What the file held before is kept as path+[BackupSuffix], and the first
// time it is written also as path+[OriginalSuffix].
func WriteFile(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("write config: %w", err)
	}

	if info != nil {
		old, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if _, err := os.Stat(path + OriginalSuffix); errors.Is(err, fs.ErrNotExist) {
			if err := replace(path+OriginalSuffix, old, info); err != nil {
				return err
			}
		}
		if err := replace(path+BackupSuffix, old, info); err != nil {
			return err
		}
	}

	return replace(path, data, info)
}

// replace replaces the file at path by data with the mode and owner of like,
// or returns error. A nil like is a file written for the first time.
func replace(path string, data []byte, like fs.FileInfo) (err error) {
	mode := newFileMode
	if like != nil {
		mode = like.Mode().Perm()
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tmp.Name()) // a leftover is all a failure here leaves
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close() // the write error is the one to report
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close() // the chmod error is the one to report
		return fmt.Errorf("write %s: %w", path, err)
	}
	group := like
	if group == nil {
		// a new file belongs to the group of its directory, which is who
		// reads it - a service user, as root:julia of /opt/julia/etc says
		if dirInfo, err := os.Stat(dir); err == nil {
			group = dirInfo
		}
	}
	if group != nil {
		if err := chown(tmp, group, like == nil); err != nil {
			_ = tmp.Close() // the chown error is the one to report
			return fmt.Errorf("write %s: %w", path, err)
		}
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close() // the sync error is the one to report
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		if !errors.Is(err, syscall.EBUSY) && !errors.Is(err, syscall.EXDEV) {
			return fmt.Errorf("write %s: %w", path, err)
		}

		// a mount point is replaced by writing it
		if err := os.WriteFile(path, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		return os.Remove(tmp.Name())
	}

	return syncDir(dir)
}

// syncDir makes the rename of a file in dir durable, or returns error
func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("sync %s: %w", dir, err)
	}
	defer func() { _ = d.Close() }() // read-only, nothing to lose

	if err := d.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		// some filesystems cannot sync a directory, which is fine
		return fmt.Errorf("sync %s: %w", dir, err)
	}

	return nil
}
