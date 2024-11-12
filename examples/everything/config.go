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

package everything

import (
	"fmt"

	"github.com/choopm/stdfx/loggingfx"
	"github.com/go-viper/mapstructure/v2"
)

// Config struct stores all config data.
type Config struct {
	Logging loggingfx.Config `mapstructure:"log"`

	// Webserver defines the http server config
	Webserver WebserverConfig `mapstructure:"webserver"`

	// Routes defines the webserver routes
	Routes []*Route `mapstructure:"routes" default:"[]"`
}

// Validate validates the Config
func (c *Config) Validate() error {
	if err := c.Webserver.Validate(); err != nil {
		return err
	}
	for i, route := range c.Routes {
		if err := route.Validate(); err != nil {
			return fmt.Errorf("route %d (%s): %s", i, route.Path, err)
		}
	}

	return nil
}

// WebserverConfig holds the webserver config
type WebserverConfig struct {
	// Host is the listening host to use when starting a server
	Host string `mapstructure:"host" default:"0.0.0.0"`

	// Port is the listening port to use when starting a server
	Port int `mapstructure:"port" default:"8080"`
}

// Validate validates the HTTPConfig
func (c *WebserverConfig) Validate() error {
	if len(c.Host) == 0 {
		return fmt.Errorf("missing webserver.host")
	}
	if c.Port == 0 {
		return fmt.Errorf("missing webserver.port")
	}

	return nil
}

// Route maps paths to content
type Route struct {
	// Path is the webserver path to register
	Path string `mapstructure:"path"`

	// Content is the content to deliver on this path
	Content any `mapstructure:"content"`
}

// Validate validates the config
func (c *Route) Validate() error {
	if len(c.Path) == 0 {
		return fmt.Errorf("missing path")
	}
	if c.Content == nil {
		return fmt.Errorf("missing content")
	}

	return nil
}

// DecodeHook returns the composite decoding hook for decoding Config
func (c *Config) DecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
	// knx group addresses listed as an example
	// knxGroupAddressDecoder(),
	)
}

// // knxGroupAddressDecoder returns a decoder for knx group addresses.
// // It parses strings of "1/2/3" into cemi.GroupAddr.
// func knxGroupAddressDecoder() mapstructure.DecodeHookFunc {
// 	// groupAddressDecoder returns a DecodeHookFunc that converts
// 	// string to cemi.GroupAddress or error.
// 	return func(
// 		f reflect.Type,
// 		t reflect.Type,
// 		data interface{},
// 	) (interface{}, error) {
// 		if f.Kind() != reflect.String {
// 			return data, nil
// 		}
// 		if t != reflect.TypeOf(cemi.GroupAddr(0)) {
// 			return data, nil
// 		}

// 		// Convert it by parsing
// 		return cemi.NewGroupAddrString(data.(string))
// 	}
// }

// LoggingConfig returns the loggingfx.Config.
// This implements an interface to support log decorators.
func (c *Config) LoggingConfig() loggingfx.Config {
	return c.Logging
}
