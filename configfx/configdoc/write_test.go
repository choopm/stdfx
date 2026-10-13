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

package configdoc_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/choopm/stdfx/configfx/configdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.yaml")

	// a new file is not readable by others
	require.NoError(t, configdoc.WriteFile(path, []byte("first\n")))
	assertFile(t, path, "first\n", 0o640)
	assert.NoFileExists(t, path+configdoc.BackupSuffix)

	// an existing one keeps its mode, and what it held is kept
	require.NoError(t, os.Chmod(path, 0o600))
	require.NoError(t, configdoc.WriteFile(path, []byte("second\n")))
	assertFile(t, path, "second\n", 0o600)
	assertFile(t, path+configdoc.BackupSuffix, "first\n", 0o600)
	assertFile(t, path+configdoc.OriginalSuffix, "first\n", 0o600)

	// the original is written once, the backup every time
	require.NoError(t, configdoc.WriteFile(path, []byte("third\n")))
	assertFile(t, path, "third\n", 0o600)
	assertFile(t, path+configdoc.BackupSuffix, "second\n", 0o600)
	assertFile(t, path+configdoc.OriginalSuffix, "first\n", 0o600)

	// nothing temporary is left behind
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestWriteFileRefusesAMissingDirectory(t *testing.T) {
	err := configdoc.WriteFile(filepath.Join(t.TempDir(), "missing", "app.yaml"), []byte("x"))
	assert.Error(t, err)
}

func assertFile(t *testing.T, path, content string, mode fs.FileMode) {
	t.Helper()

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(b), path)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, mode, info.Mode().Perm(), path)
}
