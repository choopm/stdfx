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

	"github.com/choopm/stdfx/configfx/configdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	doc, err := configdoc.Parse([]byte("log:\n  Level: info\nroles:\n  - name: a\n"))
	require.NoError(t, err)

	// keys match regardless of case, indexes select elements
	assert.Equal(t, "info", doc.Get("log", "level").Value)
	assert.Equal(t, "a", doc.Get("roles", "0", "name").Value)
	assert.Nil(t, doc.Get("roles", "1"))
	assert.Nil(t, doc.Get("missing", "key"))

	// a set replaces a value and creates what leads to a new one
	require.NoError(t, doc.Set(configdoc.StringNode("debug"), "log", "level"))
	require.NoError(t, doc.Set(configdoc.StringNode(""), "tls", "certfile"))
	require.NoError(t, doc.Set(configdoc.StringNode("b"), "roles", "1", "name"))
	assert.Error(t, doc.Set(configdoc.StringNode("x"), "roles", "5"))
	assert.Error(t, doc.Set(configdoc.StringNode("x"), "log", "level", "deeper"))
	assert.Equal(t, "log:\n  Level: debug\nroles:\n  - name: a\n  - name: b\ntls:\n  certfile: ''", configdoc.Text(doc.Root()))

	// a clone shares nothing
	clone := doc.Clone()
	require.True(t, clone.Delete("roles", "0"))
	assert.True(t, clone.Delete("tls"))
	assert.False(t, clone.Delete("tls"))
	assert.Equal(t, "a", doc.Get("roles", "0", "name").Value)
	assert.Equal(t, "log:\n  Level: debug\nroles:\n  - name: b", configdoc.Text(clone.Root()))
}

func TestParseValue(t *testing.T) {
	for in, want := range map[string]string{
		"'json'":      "'json'",
		"[a, b]":      "[a, b]",
		"":            "",
		"7d":          "7d",
		"{a: 1}":      "{a: 1}",
		"|\n  text\n": "|\n  text",
	} {
		node, err := configdoc.ParseValue(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, configdoc.Text(node), in)
	}
}

func TestParseRefusesWhatIsNoMapping(t *testing.T) {
	for _, in := range []string{"- a", "scalar", "a: *missing"} {
		_, err := configdoc.Parse([]byte(in))
		assert.Error(t, err, in)
	}

	doc, err := configdoc.Parse([]byte("# only a comment\n"))
	require.NoError(t, err)
	assert.Empty(t, doc.Root().Content)
}
