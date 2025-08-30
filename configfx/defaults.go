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

package configfx

import (
	"os"
	"path/filepath"
	"strings"
)

// DefaultEnvironmentPrefix returns the default environment prefix.
// It searches all environment variable names for a prefix of CONFIGNAME.
// If such variable exists, this prefix will be used as the default
// environment prefix for vipers autoenv feature.
func DefaultEnvironmentPrefix(configName string) string {
	upperConfigName := strings.ToUpper(configName)
	for _, envVar := range os.Environ() {
		if strings.HasPrefix(envVar, upperConfigName) {
			return upperConfigName
		}
	}

	return ""
}

// DefaultFileSearchPaths returns default config file search paths.
// In order of decreasing priority the following paths are searched
// for a file <configName> with any supported extension by default:
// - <working directory>/
// - $HOME/.config/
// - $HOME/.<configName>/
// - $HOME/.local/etc/
// - $HOME/.local/etc/<configName>/
// - $HOME/
// - /opt/<configName>/
// - /opt/<configName>/etc/
// - /opt/<configName>/etc/<configName>/
// - /usr/local/etc/
// - /usr/local/etc/<configName>/
// - /etc/
// - /etc/<configName>/
func DefaultFileSearchPaths(configName string) []string {
	// working dir
	paths := []string{
		".",
	}

	// home folder dirs
	if home := os.Getenv("HOME"); len(home) > 0 {
		paths = append(paths, []string{
			filepath.Join(home, ".config", configName),
			filepath.Join(home, "."+configName),
			filepath.Join(home, ".local/etc"),
			filepath.Join(home, ".local/etc", configName),
			home,
		}...)
	}

	// common system dirs
	paths = append(paths, []string{
		filepath.Join("/opt", configName),
		filepath.Join("/opt", configName, "etc"),
		filepath.Join("/opt", configName, "etc", configName),
		"/usr/local/etc",
		filepath.Join("/usr/local/etc", configName),
		"/etc",
		filepath.Join("/etc", configName),
	}...)

	return paths
}
