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

package configfx_test

import (
	"testing"
	"time"

	"github.com/choopm/stdfx/configfx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type decodeConfig struct {
	Host    string        `mapstructure:"host" default:"localhost"`
	Names   []string      `mapstructure:"names"`
	Port    int           `mapstructure:"port" default:"25"`
	Timeout time.Duration `mapstructure:"timeout" default:"1m"`
}

func TestDecode(t *testing.T) {
	// an environment override applies to a provider, never to Decode
	t.Setenv("HOST", "from-environment")
	t.Setenv("DECODE_HOST", "from-environment")

	tests := []struct {
		name string
		path string
		in   string
		want decodeConfig
	}{
		{
			name: "defaults",
			in:   ``,
			want: decodeConfig{Host: "localhost", Port: 25, Timeout: time.Minute},
		},
		{
			name: "values",
			path: "/nonexistent/decode.yaml",
			in:   "host: mx\nport: 2525\ntimeout: 7d\nnames: a,b\n",
			want: decodeConfig{Host: "mx", Names: []string{"a", "b"}, Port: 2525, Timeout: 7 * 24 * time.Hour},
		},
		{
			name: "json by extension",
			path: "decode.json",
			in:   `{"host": "mx"}`,
			want: decodeConfig{Host: "mx", Port: 25, Timeout: time.Minute},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configfx.Decode[decodeConfig](tt.path, []byte(tt.in))
			require.NoError(t, err)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestDecodeRefusesInvalidInput(t *testing.T) {
	for _, in := range []string{"host: [unterminated", "timeout: forever"} {
		_, err := configfx.Decode[decodeConfig]("", []byte(in))
		assert.Error(t, err, in)
	}
}
