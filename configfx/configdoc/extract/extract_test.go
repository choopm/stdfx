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

package extract_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/choopm/stdfx/configfx/configdoc/extract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// update rewrites the golden file instead of comparing with it
var update = flag.Bool("update", false, "update the golden file")

func TestExtract(t *testing.T) {
	opts := extract.Options{
		Dir:          filepath.Join("testdata", "cfg"),
		Func:         "generated",
		Package:      "docs",
		SkipUntagged: true,
		Type:         "example.com/cfg.Config",
	}

	result, err := extract.Extract(opts)
	require.NoError(t, err)
	assert.Equal(t, []string{"example.com/cfg.Log.Undocumented"}, result.Missing)

	src, err := extract.Generate(opts, result.Docs)
	require.NoError(t, err)

	golden := filepath.Join("testdata", "docs_gen.golden")
	if *update {
		require.NoError(t, os.WriteFile(golden, src, 0o644))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(src))
}

func TestExtractRefuses(t *testing.T) {
	for _, typ := range []string{"example.com/cfg.Missing", "example.com/missing.Config", "noimportpath"} {
		_, err := extract.Extract(extract.Options{Dir: filepath.Join("testdata", "cfg"), Type: typ})
		assert.Error(t, err, typ)
	}
}
