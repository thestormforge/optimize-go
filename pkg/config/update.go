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
	"fmt"
	"strings"

	"github.com/thestormforge/optimize-go/pkg/oauth2/registration"
	"golang.org/x/oauth2"
)

// SaveServer is a configuration change that persists the supplied server configuration. If the server exists,
// it is overwritten; otherwise a new named server is created.
func SaveServer(name string, srv *Server, env string) Change {
	return func(cfg *Config) error {
		mergeServers(cfg, []NamedServer{{Name: name, Server: *srv}})
		mergeAuthorizations(cfg, []NamedAuthorization{{Name: name}})

		// Make sure we capture the current value of the default server roots
		return defaultServerRoots(env, findServer(cfg.Servers, name))
	}
}

// SaveToken is a configuration change that persists the supplied token as a named authorization. If the authorization
// exists, it is overwritten; otherwise a new named authorization is created.
func SaveToken(name string, t *oauth2.Token) Change {
	return func(cfg *Config) error {
		az := findAuthorization(cfg.Authorizations, name)
		if az == nil {
			cfg.Authorizations = append(cfg.Authorizations, NamedAuthorization{Name: name})
			az = &cfg.Authorizations[len(cfg.Authorizations)-1].Authorization
		}

		az.Credential.ClientCredential = nil
		az.Credential.TokenCredential = &TokenCredential{
			AccessToken:  t.AccessToken,
			TokenType:    t.TokenType,
			RefreshToken: t.RefreshToken,
			Expiry:       t.Expiry,
		}
		return nil
	}
}

// SaveClientRegistration stores the supplied registration response to the named controller (creating it if it does not exist)
func SaveClientRegistration(name string, info *registration.ClientInformationResponse) Change {
	return func(cfg *Config) error {
		ctrl := findController(cfg.Controllers, name)
		if ctrl == nil {
			cfg.Controllers = append(cfg.Controllers, NamedController{Name: name})
			ctrl = &cfg.Controllers[len(cfg.Controllers)-1].Controller
		}

		mergeString(&ctrl.RegistrationClientURI, info.RegistrationClientURI)
		mergeString(&ctrl.RegistrationAccessToken, info.RegistrationAccessToken)
		return nil
	}
}

// ApplyCurrentContext is a configuration change that updates the values of a context and sets that context as the
// current context. If the context exists, non-empty values will overwrite; otherwise a new named context is created.
func ApplyCurrentContext(contextName, serverName, authorizationName, clusterName string) Change {
	return func(cfg *Config) error {
		ctx := findContext(cfg.Contexts, contextName)
		if ctx == nil {
			cfg.Contexts = append(cfg.Contexts, NamedContext{Name: contextName})
			ctx = &cfg.Contexts[len(cfg.Contexts)-1].Context
		}

		mergeString(&cfg.CurrentContext, contextName)
		mergeString(&ctx.Server, serverName)
		mergeString(&ctx.Authorization, authorizationName)
		mergeString(&ctx.Cluster, clusterName)
		return nil
	}
}

// SetExecutionEnvironment is a configuration change that updates the execution environment
func SetExecutionEnvironment(env string) Change {
	return func(cfg *Config) error {
		// Normalize and validate the execution environment name
		if env != "" {
			switch strings.ToLower(env) {
			case "production", "prod":
				env = environmentProduction
			case "staging", "stage":
				env = environmentStaging
			case "development", "dev":
				env = environmentDevelopment
			default:
				return fmt.Errorf("unknown environment: %s", env)
			}
		}

		// Do not explicitly persist the "production" value
		mergeString(&cfg.Environment, env)
		if cfg.Environment == environmentProduction {
			cfg.Environment = ""
		}
		return nil
	}
}

// SetProperty is a configuration change that updates a single property using a dotted name notation.
func SetProperty(name, value string) Change {
	// TODO This is a giant hack. Consider not even supporting `config set` generically
	return func(cfg *Config) error {
		path := strings.Split(name, ".")
		switch path[0] {

		case "env":
			return SetExecutionEnvironment(value)(cfg)

		case "current-context":
			cfg.CurrentContext = value
			return nil

		case "cluster":
			if len(path) == 3 {
				switch path[2] {
				case "context":
					mergeClusters(cfg, []NamedCluster{{
						Name:    path[1],
						Cluster: Cluster{Context: value},
					}})
					return nil
				case "bin":
					mergeClusters(cfg, []NamedCluster{{
						Name:    path[1],
						Cluster: Cluster{Bin: value},
					}})
					return nil
				case "controller":
					mergeClusters(cfg, []NamedCluster{{
						Name:    path[1],
						Cluster: Cluster{Controller: value},
					}})
					return nil
				}
			}

		case "controller":
			if len(path) == 4 {
				switch path[2] {
				case "env":
					mergeControllers(cfg, []NamedController{{
						Name:       path[1],
						Controller: Controller{Env: []ControllerEnvVar{{Name: path[3], Value: value}}},
					}})
					return nil
				case "resources":
					mergeControllers(cfg, []NamedController{{
						Name: path[1],
						Controller: Controller{Resources: &ControllerResources{
							Requests: map[string]string{path[3]: value},
							Limits:   map[string]string{path[3]: value},
						}},
					}})
					return nil
				}
			}

		case "context":
			if len(path) == 3 {
				var context *Context
				switch path[2] {
				case "server":
					if findServer(cfg.Servers, value) == nil {
						return fmt.Errorf("unknown server reference: %s", value)
					}
					context = &Context{Server: value}
				case "authorization":
					if findAuthorization(cfg.Authorizations, value) == nil {
						return fmt.Errorf("unknown authorization reference: %s", value)
					}
					context = &Context{Authorization: value}
				case "cluster":
					if findCluster(cfg.Clusters, value) == nil {
						return fmt.Errorf("unknown cluster reference: %s", value)
					}
					context = &Context{Cluster: value}
				}

				if context != nil {
					mergeContexts(cfg, []NamedContext{{
						Name:    path[1],
						Context: *context,
					}})
					return nil
				}
			}
		}

		return fmt.Errorf("unknown config property: %s", name)
	}
}

// UnsetProperty eliminates a configuration property. Note that in general, just
// setting a property to an empty value will not overwrite the existing value.
func UnsetProperty(name string) Change {
	// TODO This is just as bad a hack as SetProperty...what are we doing here?
	return func(cfg *Config) error {
		path := strings.Split(name, ".")
		switch path[0] {

		case "env":
			cfg.Environment = ""
			return nil

		case "current-context":
			cfg.CurrentContext = ""
			return nil

		case "cluster":
			if len(path) == 3 {
				if cstr := findCluster(cfg.Clusters, path[1]); cstr != nil {
					switch path[2] {
					case "context":
						cstr.Context = ""
						return nil
					case "bin":
						cstr.Bin = ""
						return nil
					case "controller":
						cstr.Controller = ""
						return nil
					}
				}
			}

		case "controller":
			if len(path) == 4 {
				if ctrl := findController(cfg.Controllers, path[1]); ctrl != nil {
					switch path[2] {
					case "env":
						if len(ctrl.Env) == 1 && ctrl.Env[0].Name == path[3] {
							ctrl.Env = nil
						} else if len(ctrl.Env) > 1 {
							j := 0
							for i := 0; i < len(ctrl.Env); i++ {
								if ctrl.Env[i].Name != path[3] {
									ctrl.Env[j] = ctrl.Env[i]
									j++
								}
							}
							ctrl.Env = ctrl.Env[0:j]
						}
						return nil
					case "resources":
						if ctrl.Resources != nil {
							delete(ctrl.Resources.Requests, path[3])
							delete(ctrl.Resources.Limits, path[3])
							if len(ctrl.Resources.Requests)+len(ctrl.Resources.Limits) == 0 {
								ctrl.Resources = nil
							}
						}
						return nil
					}
				}
			}

		case "context":
			if len(path) == 3 {
				if ctx := findContext(cfg.Contexts, path[1]); ctx != nil {
					switch path[2] {
					case "server":
						ctx.Server = ""
						return nil
					case "authorization":
						ctx.Authorization = ""
						return nil
					case "cluster":
						ctx.Cluster = ""
						return nil
					}
				}
			}

		}

		return fmt.Errorf("unknown config property: %s", name)
	}
}
