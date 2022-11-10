/*
Copyright 2022 GramLabs, Inc.

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

package command

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/square/go-jose.v2/jwt"
	"sigs.k8s.io/yaml"
)

// NewWhoAmICommand returns a command for determining the current identity associated
// with the configuration.
func NewWhoAmICommand(cfg Config) *cobra.Command {
	var (
		output  string
		pattern string
	)

	cmd := &cobra.Command{
		Use: "whoami",
	}

	cmd.Flags().StringVarP(&output, "output", "o", output, "the output `format` to use; one of: json|yaml|go-template")
	cmd.Flags().StringVar(&pattern, "template", pattern, "the template `text` used to render the claims")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx, out := cmd.Context(), cmd.OutOrStdout()

		// Fetch a token so we can inspect the claims
		tok, err := token(ctx, cfg)
		if err != nil {
			return err
		}

		// Ignore the signature, just extract the claims
		accessToken, err := jwt.ParseSigned(tok.AccessToken)
		if err != nil {
			return err
		}
		claims := map[string]interface{}{}
		if err := accessToken.UnsafeClaimsWithoutVerification(&claims); err != nil {
			return err
		}

		// Choose a template
		switch output {
		case "json", "":
			pattern = "{{ toJson . }}"
		case "yaml":
			pattern = "{{ toYaml . }}"
		case "go-template":
			if pattern == "" {
				return fmt.Errorf("missing template")
			}
		default:
			return fmt.Errorf("unknown format: %s", output)
		}

		// Send it through a template
		tmpl, err := template.New("out").
			Funcs(map[string]interface{}{
				"toJson": func(v interface{}) (string, error) {
					var buf strings.Builder
					enc := json.NewEncoder(&buf)
					enc.SetIndent("", "  ")
					err := enc.Encode(v)
					return buf.String(), err
				},
				"toYaml": func(v interface{}) (string, error) {
					data, err := yaml.Marshal(v)
					return string(data), err
				},
			}).
			Parse(pattern)
		if err != nil {
			return err
		}
		return tmpl.Execute(out, claims)
	}
	return cmd
}

// token returns an access token obtained using the supplied configuration.
func token(ctx context.Context, cfg Config) (*oauth2.Token, error) {
	// Check that the configuration can produce a token source
	tcfg, ok := cfg.(interface {
		TokenSource(ctx2 context.Context) oauth2.TokenSource
	})
	if !ok {
		return nil, fmt.Errorf("unable to obtain token to ascertain identity")
	}

	// If there is a token source, use it to obtain a token
	ts := tcfg.TokenSource(ctx)
	if ts == nil {
		return nil, fmt.Errorf("not logged in")
	}
	return ts.Token()
}
