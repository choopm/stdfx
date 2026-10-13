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
	"testing"
	"time"

	"github.com/choopm/stdfx/configfx"
	"github.com/choopm/stdfx/configfx/configdoc"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pkg is the import path the types of this file are declared in
const pkg = "github.com/choopm/stdfx/configfx/configdoc_test"

type Config struct {
	Log Log `mapstructure:"log"`

	Listener `mapstructure:",squash"`

	Hidden  string            `mapstructure:"hidden"`
	Options map[string]string `mapstructure:"options" default:"{}"`
	Roles   []Role            `mapstructure:"roles"`
	TLS     *TLSConfig        `mapstructure:"tls"`

	Templates map[string]Template `mapstructure:"templates"`
	Mail      Mail                `mapstructure:"mail"`

	Derived string
	Skipped string `mapstructure:"-"`
}

type Log struct {
	Format  string        `mapstructure:"format" default:"text" validate:"oneof=text json"`
	Level   Level         `mapstructure:"level" default:"info"`
	Timeout time.Duration `mapstructure:"timeout" default:"1h"`
	Prefix  *string       `mapstructure:"prefix"`
}

type Mail struct {
	Footer *string `mapstructure:"footer"`
}

type Level string

type Listener struct {
	Host string `mapstructure:"host" default:"localhost"`
	Mode int    `mapstructure:"mode" default:"0644"`
	Port int    `mapstructure:"port" default:"25"`
}

type Role struct {
	Name   string   `mapstructure:"name"`
	Scopes []string `mapstructure:"scopes" default:"[\"read\"]"`
}

type TLSConfig struct {
	Certfile string `mapstructure:"certfile" default:"/etc/cert.pem"`
	Enabled  bool   `mapstructure:"enabled"`
}

type Template struct {
	Subject string `mapstructure:"subject"`
	Text    string `mapstructure:"text"`
}

func testDocs() *configdoc.Docs {
	return &configdoc.Docs{
		Enums: map[configdoc.TypeKey][]configdoc.Enum{
			{Name: "Level", Package: pkg}: {{Doc: "LevelDebug logs a lot", Value: "debug"}, {Value: "info"}},
		},
		Examples: map[string]string{
			"log.prefix":  "prefix: '[app] '",
			"mail.footer": "footer: regards",
			"tls":         "tls:\n  enabled: true",
		},
		Fields: map[configdoc.Key]configdoc.Doc{
			{Field: "Format", Package: pkg, Type: "Log"}:       {Text: "Format is how a line is written, see [Log.Level] too.", Line: "text, json"},
			{Field: "Host", Package: pkg, Type: "Listener"}:    {Text: "Host to listen on"},
			{Field: "Level", Package: pkg, Type: "Log"}:        {Text: "Level is the least severity logged."},
			{Field: "Name", Package: pkg, Type: "Role"}:        {Text: "Name of the role"},
			{Field: "Options", Package: pkg, Type: "Config"}:   {Text: "Options are passed on as they are."},
			{Field: "Port", Package: pkg, Type: "Listener"}:    {Text: "Port to listen on"},
			{Field: "Roles", Package: pkg, Type: "Config"}:     {Text: "Roles are granted to everyone."},
			{Field: "Templates", Package: pkg, Type: "Config"}: {Block: true, Text: "Templates by kind."},
			{Field: "Timeout", Package: pkg, Type: "Log"}:      {Block: true, Text: "Timeout bounds a write."},
			{Field: "TLS", Package: pkg, Type: "Config"}:       {Text: "TLS secures the listener."},
			{Field: "Subject", Package: pkg, Type: "Template"}: {Text: "Subject of the mail"},
		},
		Header: "maintained by a test",
		Hidden: []string{"hidden"},
		Overrides: map[string]string{
			"options": "Options are what the application says about them.",
		},
		SkipUntagged: true,
		Types: map[configdoc.TypeKey]string{
			{Name: "Log", Package: pkg}: "Log configures logging.",
		},
	}
}

func testSchema(t *testing.T) *configdoc.Schema {
	t.Helper()

	schema, err := configdoc.NewSchema[Config](testDocs())
	require.NoError(t, err)

	return schema
}

func TestRenderEmpty(t *testing.T) {
	out, report, err := testSchema(t).Render(nil)
	require.NoError(t, err)

	assert.Equal(t, `# maintained by a test

# log configures logging.
log:
  # prefix: '[app] '
  #
  # format is how a line is written, see level too.
  format: text # text, json
  # level is the least severity logged.
  level: info

  # timeout bounds a write.
  timeout: 1h

# host to listen on
host: localhost

mode: 0644

# port to listen on
port: 25

# options are what the application says about them.
options: {}

# roles are granted to everyone.
roles: []

# tls secures the listener.
#
# tls:
#   enabled: true
#
# templates by kind.
templates: {}

# footer: regards
mail: {}
`, string(out))
	assert.Equal(t, []string{"log.format", "log.level", "log.timeout", "host", "mode", "port", "options", "roles", "templates"}, report.Added)
	assert.Empty(t, report.Dropped)
	assert.Empty(t, report.Mismatched)
}

func TestRenderKeepsValues(t *testing.T) {
	in := `# a comment of the operator, which is replaced
port: 2525 # so is this
LOG:
  timeout: 7d
  format: 'json'
options:
  b: "2"
  a: 1
roles:
  - name: admin
    scopes: [read, write]
    typo: x
  - name: viewer
tls:
  enabled: true
templates:
  welcome:
    text: |
      hello
      world
unknown: 1
hidden: x
`
	out, report, err := testSchema(t).Render([]byte(in))
	require.NoError(t, err)

	assert.Equal(t, `# maintained by a test

# log configures logging.
log:
  # prefix: '[app] '
  #
  # format is how a line is written, see level too.
  format: 'json' # text, json
  # level is the least severity logged.
  level: info

  # timeout bounds a write.
  timeout: 7d

# host to listen on
host: localhost

mode: 0644

# port to listen on
port: 2525

# options are what the application says about them.
options:
  b: "2"
  a: 1

# roles are granted to everyone.
roles:
  # name of the role
  - name: admin
    scopes: [read, write]
  - name: viewer

# tls secures the listener.
#
# tls:
#   enabled: true
tls:
  certfile: ''
  enabled: true

# templates by kind.
templates:
  welcome:
    text: |
      hello
      world

# footer: regards
mail: {}
`, string(out))
	assert.Equal(t, []string{"log.level", "host", "mode", "tls.certfile"}, report.Added)
	assert.Equal(t, []string{"roles.0.typo", "unknown", "hidden"}, report.Dropped)
	assert.Empty(t, report.Mismatched)
}

func TestRenderIsIdempotent(t *testing.T) {
	schema := testSchema(t)
	for _, in := range []string{"", "roles: [{name: a}]\nlog: {level: debug}\n", "tls: {enabled: true}\noptions: {x: y}\n"} {
		once, _, err := schema.Render([]byte(in))
		require.NoError(t, err)
		twice, report, err := schema.Render(once)
		require.NoError(t, err)

		assert.Equal(t, string(once), string(twice), in)
		assert.Empty(t, report.Added, in)
		assert.Empty(t, report.Dropped, in)
	}
}

func TestRenderKeepsTheDecodedConfig(t *testing.T) {
	schema := testSchema(t)
	for _, in := range []string{
		"",
		"port: 2525\nlog: {timeout: 2d, format: json}\n",
		"roles:\n  - name: admin\n  - {name: viewer, scopes: [read]}\n",
		"tls:\n  enabled: true\n",
		"templates:\n  welcome:\n    text: hi\n",
		"base: &base {name: a}\nroles: [*base, {<<: *base, scopes: [x]}]\n",
	} {
		before, err := configfx.Decode[Config]("", []byte(in))
		require.NoError(t, err, in)

		out, _, err := schema.Render([]byte(in))
		require.NoError(t, err, in)
		after, err := configfx.Decode[Config]("", out)
		require.NoError(t, err, in)

		assert.Empty(t, cmp.Diff(before, after, cmpopts.EquateEmpty()), in)
	}
}

func TestRenderRefuses(t *testing.T) {
	schema := testSchema(t)
	for name, in := range map[string]string{
		"duplicate key":  "port: 1\nPORT: 2\n",
		"no mapping":     "- a\n",
		"invalid syntax": "port: [\n",
	} {
		_, _, err := schema.Render([]byte(in))
		assert.Error(t, err, name)
	}
}

func TestRenderReportsMismatches(t *testing.T) {
	_, report, err := testSchema(t).Render([]byte("log: [a]\nport: {a: b}\nroles: x\n"))
	require.NoError(t, err)

	assert.Equal(t, []string{"log", "port", "roles"}, report.Mismatched)
}

func TestSchema(t *testing.T) {
	schema := testSchema(t)

	level := schema.Lookup("log", "level")
	require.NotNil(t, level)
	assert.Equal(t, configdoc.Scalar, level.Kind)
	assert.Equal(t, "log.level", level.Path)
	assert.Equal(t, []configdoc.Enum{{Doc: "LevelDebug logs a lot", Value: "debug"}, {Value: "info"}}, level.Enum)

	format := schema.Lookup("LOG", "format")
	require.NotNil(t, format)
	assert.Equal(t, []configdoc.Enum{{Value: "text"}, {Value: "json"}}, format.Enum)

	name := schema.Lookup("roles", "3", "name")
	require.NotNil(t, name)
	assert.Equal(t, "roles.*.name", name.Path)

	assert.Nil(t, schema.Lookup("derived"))
	assert.Nil(t, schema.Lookup("skipped"))
	assert.True(t, schema.Lookup("hidden").Hidden)

	var paths []string
	schema.Walk(func(s *configdoc.Setting) bool {
		paths = append(paths, s.Path)
		return s.Path != "log"
	})
	assert.Equal(t, []string{
		"log", "host", "mode", "port", "hidden", "options", "options.*", "roles", "roles.*",
		"roles.*.name", "roles.*.scopes", "roles.*.scopes.*", "tls", "tls.certfile", "tls.enabled",
		"templates", "templates.*", "templates.*.subject", "templates.*.text", "mail", "mail.footer",
	}, paths)
}

func TestNewElement(t *testing.T) {
	role, err := testSchema(t).Lookup("roles").NewElement()
	require.NoError(t, err)

	assert.Equal(t, Role{Scopes: []string{"read"}}, role.Interface())
}

func TestNewElementNode(t *testing.T) {
	node, err := testSchema(t).Lookup("roles").NewElementNode()
	require.NoError(t, err)

	assert.Equal(t, "name: ''\nscopes: [read]", configdoc.Text(node))
}

func TestDescribe(t *testing.T) {
	schema := testSchema(t)

	assert.Equal(t, "format is how a line is written,\nsee level too.",
		schema.Describe(schema.Lookup("log", "format"), 40))
	assert.Equal(t, "log configures logging.", schema.Describe(schema.Lookup("log"), 80))
	assert.Empty(t, schema.Describe(schema.Lookup("roles", "*", "scopes"), 80))
}

func TestFormatDuration(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                          "0s",
		time.Hour:                  "1h",
		336 * time.Hour:            "336h",
		90 * time.Minute:           "1h30m",
		2 * time.Minute:            "2m",
		90 * time.Second:           "1m30s",
		500 * time.Millisecond:     "500ms",
		time.Hour + 30*time.Second: "1h0m30s",
	} {
		assert.Equal(t, want, configdoc.FormatDuration(d), d)
	}
}
