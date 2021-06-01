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
	"net/url"
	"os"
)

// envLoader adds environment variable overrides to the configuration
func envLoader(cfg *OptimizeConfig) error {
	defaultString(&cfg.Overrides.Environment, os.Getenv("STORMFORGE_ENV"))
	defaultString(&cfg.Overrides.ServerIdentifier, os.Getenv("STORMFORGE_SERVER_IDENTIFIER"))
	defaultString(&cfg.Overrides.ServerIssuer, os.Getenv("STORMFORGE_SERVER_ISSUER"))
	defaultString(&cfg.Overrides.Credential.ClientID, os.Getenv("STORMFORGE_AUTHORIZATION_CLIENT_ID"))
	defaultString(&cfg.Overrides.Credential.ClientSecret, os.Getenv("STORMFORGE_AUTHORIZATION_CLIENT_SECRET"))
	return nil
}

// EnvironmentMapping returns an environment variable map from the specified configuration reader
func EnvironmentMapping(r Reader, includeController bool) (map[string][]byte, error) {
	env := make(map[string][]byte)

	// Record the server information
	srv, err := CurrentServer(r)
	if err != nil {
		return nil, err
	}
	env["STORMFORGE_SERVER_IDENTIFIER"] = []byte(srv.Identifier)
	env["STORMFORGE_SERVER_ISSUER"] = []byte(srv.Authorization.Issuer)

	// Record the authorization information
	az, err := CurrentAuthorization(r)
	if err != nil {
		return nil, err
	}
	if az.Credential.ClientCredential != nil {
		env["STORMFORGE_AUTHORIZATION_CLIENT_ID"] = []byte(az.Credential.ClientID)
		env["STORMFORGE_AUTHORIZATION_CLIENT_SECRET"] = []byte(az.Credential.ClientSecret)
	}

	// Optionally record environment variables from the controller configuration
	if includeController {
		ctrl, err := CurrentController(r)
		if err != nil {
			return nil, err
		}

		for i := range ctrl.Env {
			env[ctrl.Env[i].Name] = []byte(ctrl.Env[i].Value)
		}

		// The controller needs it's issuer to match the registration host
		if u, err := url.Parse(srv.Authorization.RegistrationEndpoint); err == nil {
			u.Path = "/"
			env["STORMFORGE_SERVER_ISSUER"] = []byte(u.String())
		}
	}

	// Strip out blanks
	for k, v := range env {
		if len(v) == 0 {
			delete(env, k)
		}
	}
	return env, nil
}
