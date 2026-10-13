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

package zerologfx_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/choopm/stdfx/loggingfx"
	"github.com/choopm/stdfx/loggingfx/zerologfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLogsIntoAFile asserts that a file named as output is created and
// appended to, with timestamps in RFC 3339 when no time format is given.
func TestNewLogsIntoAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	require.NoError(t, os.WriteFile(path, []byte("before\n"), 0o600))

	logger, err := zerologfx.New(loggingfx.Config{Format: "json", Level: "info", Output: path})
	require.NoError(t, err)
	logger.Info().Msg("written")

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	require.Len(t, lines, 2, "the file is appended to")
	assert.Equal(t, "before", lines[0])

	var line map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &line))
	assert.Equal(t, "written", line["message"])
	_, err = time.Parse(time.RFC3339, line["time"].(string))
	assert.NoError(t, err)
}
