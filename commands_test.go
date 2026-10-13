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

package stdfx_test

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choopm/stdfx"
	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/configfx/configdoc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileProvider provides the config of one file, without the flags a
// [configfx.SourceFile] registers globally
type fileProvider struct {
	v *viper.Viper
}

func (s *fileProvider) Config(opts ...configfx.ConfigOption) (*Config, error) {
	return nil, nil
}

func (s *fileProvider) Viper() *viper.Viper {
	return s.v
}

// runConfigCommand runs the config command with args against a file holding
// content and returns the file afterwards, what was printed and the error
func runConfigCommand(t *testing.T, content string, args ...string) (string, string, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "server.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	v := viper.New()
	v.SetConfigFile(path)
	cmd := stdfx.ConfigCommand(stdfx.ConfigCommandParams[Config]{
		Docs: &configdoc.Docs{
			Fields: map[configdoc.Key]configdoc.Doc{
				{Field: "Port", Package: "github.com/choopm/stdfx_test", Type: "WebserverConfig"}: {Text: "Port is the listening port."},
			},
		},
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Provider: &fileProvider{v: v},
	})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()

	b, readErr := os.ReadFile(path)
	require.NoError(t, readErr)

	return string(b), out.String(), err
}

func TestConfigFormat(t *testing.T) {
	file, _, err := runConfigCommand(t, "# mine\nwebserver:\n  port: 9090\nunknown: 1\n", "format")
	require.NoError(t, err)

	assert.Contains(t, file, "  # port is the listening port.\n  port: 9090\n")
	assert.Contains(t, file, "routes: []\n")
	assert.NotContains(t, file, "unknown")
	assert.NotContains(t, file, "# mine")
}

func TestConfigFormatPrints(t *testing.T) {
	in := "webserver:\n  port: 9090\n"
	file, out, err := runConfigCommand(t, in, "format", "--print")
	require.NoError(t, err)

	assert.Equal(t, in, file)
	assert.Contains(t, out, "port: 9090\n")
}

func TestConfigSet(t *testing.T) {
	file, _, err := runConfigCommand(t, "webserver:\n  port: 9090\n", "set", "webserver.host=example.com", "routes=[]")
	require.NoError(t, err)

	assert.Contains(t, file, "  host: example.com\n")
	assert.Contains(t, file, "  port: 9090\n")
}

func TestConfigSetRefusesWhatDoesNotDecode(t *testing.T) {
	in := "webserver:\n  port: 9090\n"
	file, _, err := runConfigCommand(t, in, "set", "webserver.port=many")
	require.Error(t, err)

	assert.Equal(t, in, file)
}
