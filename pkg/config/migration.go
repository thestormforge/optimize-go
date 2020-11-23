/*
Copyright 2020 GramLabs, Inc.

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

package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// migrationLoader will take the meaningful bits from a legacy config file and delete that file once the changes are persisted
func migrationLoader(cfg *RedSkyConfig) error {
	// Migrate the really old `~/.redsky` file
	if err := migrateDotRedSky(cfg); err != nil {
		return err
	}

	// Migrate the old `~/.config/redsky/config` file
	if err := migrateXDGRedSky(cfg); err != nil {
		return err
	}

	return nil
}

// migrateDotRedSky migrates the really old '~/.redsky' file into the supplied configuration.
func migrateDotRedSky(cfg *RedSkyConfig) error {
	filename := filepath.Join(os.Getenv("HOME"), ".redsky")

	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = f.Close() }()

	// legacyConfig contains only the parts of the legacy configuration object that we can migrate; the address and
	// credentials were all invalidated when the remote server switched to a single endpoint
	legacyConfig := &struct {
		Manager struct {
			Env []ControllerEnvVar `json:"env"`
		} `json:"manager"`
	}{}

	if err := yaml.NewYAMLOrJSONDecoder(bufio.NewReader(f), 4096).Decode(legacyConfig); err != nil {
		return err
	}
	if len(legacyConfig.Manager.Env) == 0 {
		return nil
	}

	legacyControllers := []NamedController{
		{Name: bootstrapClusterName(), Controller: Controller{Env: legacyConfig.Manager.Env}},
	}

	// We can't use cfg.Update here because we only want to remove the file as part of cfg.Write
	mergeControllers(&cfg.data, legacyControllers)
	cfg.unpersisted = append(cfg.unpersisted, func(cfg *Config) error {
		mergeControllers(cfg, legacyControllers)
		return os.Remove(filename)
	})

	return nil
}

// migrateXDGRedSky migrates the old '~/.config/redsky/config' file into the supplied configuration.
func migrateXDGRedSky(cfg *RedSkyConfig) error {
	filename, _ := configFilename("redsky/config")

	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = f.Close() }()

	// The old file was basically the same as the new file with one exception: the name of the API server section. We
	// need to decode into an unstructured type so we can perform the necessary rename.
	legacyConfig := map[string]interface{}{}

	if err := yaml.NewYAMLOrJSONDecoder(bufio.NewReader(f), 4096).Decode(&legacyConfig); err != nil {
		return err
	}

	if servers, ok := legacyConfig["servers"].([]interface{}); ok {
		for i := range servers {
			if namedServer, ok := servers[i].(map[string]interface{}); ok {
				if server, ok := namedServer["server"].(map[string]interface{}); ok {
					if rs, ok := server["redsky"]; ok {
						server["api"] = rs
					}
				}
			}
		}
	}

	// Now that the unstructured data is in the right format we can round trip it into the correct structure
	data := &Config{}
	if b, err := json.Marshal(legacyConfig); err != nil {
		return err
	} else if err := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 4096).Decode(data); err != nil {
		return err
	}

	// We can't use cfg.Update here because we only want to remove the file as part of cfg.Write
	mergeConfig(&cfg.data, data)
	cfg.unpersisted = append(cfg.unpersisted, func(cfg *Config) error {
		mergeConfig(cfg, data)
		if err := os.Remove(filename); err != nil {
			return err
		}
		_ = os.Remove(filepath.Dir(filename))
		return nil
	})

	return nil
}
