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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// migrationLoader will take the meaningful bits from a legacy config file and delete that file once the changes are persisted
func migrationLoader(cfg *OptimizeConfig) error {
	// Migrate the really old `~/.redsky` file
	if err := migrateDotRedSky(cfg); err != nil {
		return err
	}

	// Migrate the old `~/.config/redsky/config` file
	if err := migrateXDGRedSky(cfg); err != nil {
		return err
	}

	// Migrate the old environment variables
	if err := migrateRedSkyEnv(cfg); err != nil {
		return err
	}

	// Migrate the server identifier to drop the /v1/
	if err := migrateServerIdentifier(cfg); err != nil {
		return err
	}

	// Migrate the old "carbonrelay" hostnames
	if err := migrateCarbonRelayHostnames(cfg); err != nil {
		return err
	}

	return nil
}

// migrateDotRedSky migrates the really old '~/.redsky' file into the supplied configuration.
func migrateDotRedSky(cfg *OptimizeConfig) error {
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
func migrateXDGRedSky(cfg *OptimizeConfig) error {
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

// migrateRedSkyEnv migrates the old environment variables.
func migrateRedSkyEnv(cfg *OptimizeConfig) error {
	// This should be consistent with the expected behavior because migrations
	// run after environment loading and we are only applying defaults to overrides
	defaultString(&cfg.Overrides.Environment, os.Getenv("REDSKY_ENV"))
	defaultString(&cfg.Overrides.ServerIdentifier, os.Getenv("REDSKY_SERVER_IDENTIFIER"))
	defaultString(&cfg.Overrides.ServerIssuer, os.Getenv("REDSKY_SERVER_ISSUER"))
	defaultString(&cfg.Overrides.Credential.ClientID, os.Getenv("REDSKY_AUTHORIZATION_CLIENT_ID"))
	defaultString(&cfg.Overrides.Credential.ClientSecret, os.Getenv("REDSKY_AUTHORIZATION_CLIENT_SECRET"))
	return nil
}

// migrateServerIdentifier removes the `/v1/` suffix from the server identifier.
func migrateServerIdentifier(cfg *OptimizeConfig) error {
	// Require both path separators, but leave the trailing slash in place
	trimV1 := func(s string) string {
		if strings.HasSuffix(s, "/v1/") {
			return strings.TrimSuffix(s, "v1/")
		}
		return s
	}

	// Update the overrides for stale environment variables
	cfg.Overrides.ServerIdentifier = trimV1(cfg.Overrides.ServerIdentifier)

	// Check to see if we need to make a change to any persisted server identifiers
	for _, svr := range cfg.data.Servers {
		if svr.Server.Identifier == trimV1(svr.Server.Identifier) {
			continue
		}

		// Update all servers with a `/v1/` suffix
		return cfg.Update(func(cfg *Config) error {
			for i := range cfg.Servers {
				cfg.Servers[i].Server.Identifier = trimV1(cfg.Servers[i].Server.Identifier)
			}
			return nil
		})
	}
	return nil
}

// migrateCarbonRelayHostnames updates the old "carbonrelay" hostnames to use "stormforge" instead.
func migrateCarbonRelayHostnames(cfg *OptimizeConfig) error {
	// Helper to get all the URLs in the configuration
	allEndpoints := func(cfg *Config) []*string {
		var locators []*string
		for i := range cfg.Servers {
			locators = append(locators,
				&cfg.Servers[i].Server.Identifier,
				&cfg.Servers[i].Server.API.ApplicationsEndpoint,
				&cfg.Servers[i].Server.API.ExperimentsEndpoint,
				&cfg.Servers[i].Server.API.AccountsEndpoint,
				&cfg.Servers[i].Server.API.PerformanceTokenEndpoint,
				&cfg.Servers[i].Server.Authorization.Issuer,
				&cfg.Servers[i].Server.Authorization.AuthorizationEndpoint,
				&cfg.Servers[i].Server.Authorization.TokenEndpoint,
				&cfg.Servers[i].Server.Authorization.RevocationEndpoint,
				&cfg.Servers[i].Server.Authorization.RegistrationEndpoint,
				&cfg.Servers[i].Server.Authorization.DeviceAuthorizationEndpoint,
				&cfg.Servers[i].Server.Authorization.JSONWebKeySetURI,
			)
		}
		for i := range cfg.Controllers {
			locators = append(locators,
				&cfg.Controllers[i].Controller.RegistrationClientURI,
			)
		}
		return locators
	}

	// Check to see if we need to make a change to any endpoints
	for _, s := range allEndpoints(&cfg.data) {
		if *s == "" || !strings.Contains(*s, ".carbonrelay.") {
			continue
		}

		// Update all endpoints with an outdated hostname
		return cfg.Update(func(cfg *Config) error {
			for _, s := range allEndpoints(cfg) {
				if *s == "" {
					continue
				}

				u, err := url.Parse(*s)
				if err != nil {
					// Silently ignore the error and try a direct string replacement instead
					*s = strings.ReplaceAll(*s, ".carbonrelay.", ".stormforge.")
				} else {
					// For valid URLs, only replace the value in the host field
					u.Host = strings.ReplaceAll(u.Host, ".carbonrelay.", ".stormforge.")
					*s = u.String()
				}
			}
			return nil
		})
	}
	return nil
}
