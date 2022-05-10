/*
Copyright 2021 GramLabs, Inc.

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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	applications "github.com/thestormforge/optimize-go/pkg/api/applications/v2"
	experiments "github.com/thestormforge/optimize-go/pkg/api/experiments/v1alpha1"
	"github.com/thestormforge/optimize-go/pkg/command"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	cfg := &config{
		address: os.Getenv("STORMFORGE_SERVER"),
	}

	cmd := &cobra.Command{
		Use:          "optimize",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cc := clientcredentials.Config{
				ClientID:     os.Getenv("STORMFORGE_CLIENT_ID"),
				ClientSecret: os.Getenv("STORMFORGE_CLIENT_SECRET"),
				TokenURL:     os.Getenv("STORMFORGE_ISSUER") + "oauth/token",
				AuthStyle:    oauth2.AuthStyleInParams,
				EndpointParams: map[string][]string{
					"audience": {cfg.address},
				},
			}

			dt := http.DefaultTransport
			http.DefaultTransport = &oauth2.Transport{
				Source: oauth2.ReuseTokenSource(nil, cc.TokenSource(cmd.Context())),
				Base:   dt,
			}
		},
	}

	// Aggregate the CREATE commands
	createCmd := &cobra.Command{
		Use: "create",
	}

	createCmd.AddCommand(
		command.NewCreateApplicationCommand(cfg, &printer{format: `created application %q.`}),
		command.NewCreateScenarioCommand(cfg, &printer{format: `created scenario %q.`}),
		command.NewCreateTrialCommand(cfg, &printer{format: `created trial %q.`}),
	)

	// Aggregate the EDIT commands
	editCmd := &cobra.Command{
		Use: "edit",
	}

	editCmd.AddCommand(
		command.NewEditApplicationCommand(cfg, &printer{format: `updated application %q.`}),
		command.NewEditScenarioCommand(cfg, &printer{format: `updated scenario %q.`}),
		command.NewEditExperimentCommand(cfg, &printer{format: `updated experiment %q.`}),
		command.NewEditTrialCommand(cfg, &printer{format: `updated trial %q.`}),
		command.NewEditClusterCommand(cfg, &printer{format: `updated cluster %q.`}),
	)

	// Aggregate the GET commands
	getCmd := &cobra.Command{
		Use: "get",
	}

	getCmd.AddCommand(
		command.NewGetApplicationsCommand(cfg, &printer{}),
		command.NewGetScenariosCommand(cfg, &printer{}),
		command.NewGetRecommendationsCommand(cfg, &printer{}),
		command.NewGetExperimentsCommand(cfg, &printer{}),
		command.NewGetTrialsCommand(cfg, &printer{}),
		command.NewGetClustersCommand(cfg, &printer{}),
	)

	// Aggregate the DELETE commands
	deleteCmd := &cobra.Command{
		Use: "delete",
	}

	deleteCmd.AddCommand(
		command.NewDeleteApplicationsCommand(cfg, &printer{format: `deleted application %q.`}),
		command.NewDeleteScenariosCommand(cfg, &printer{format: `deleted scenario %q.`}),
		command.NewDeleteExperimentsCommand(cfg, &printer{format: `deleted experiment %q.`}),
		command.NewDeleteTrialsCommand(cfg, &printer{format: `deleted trial %q.`}),
		command.NewDeleteClustersCommand(cfg, &printer{format: `deleted cluster %q.`}),
	)

	// Aggregate the WATCH commands
	watchCmd := &cobra.Command{
		Use: "watch",
	}

	watchCmd.AddCommand(
		command.NewWatchActivityCommand(cfg),
	)

	// Add the aggregate commends to the root
	cmd.AddCommand(
		createCmd,
		editCmd,
		getCmd,
		deleteCmd,
		watchCmd,
	)

	// Create a context for the command
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   10 * time.Second,
	})

	// Run the command
	err := cmd.ExecuteContext(ctx)
	cancel()
	if err != nil {
		os.Exit(1)
	}
}

type config struct {
	address string
}

func (c *config) Address() string {
	return c.address
}

type printer struct {
	format string
}

func (p *printer) Fprint(w io.Writer, obj interface{}) error {
	if p.format != "" {
		var err error
		switch obj := obj.(type) {
		case *applications.ApplicationItem:
			_, err = fmt.Fprintf(w, p.format, obj.Name)
		case *experiments.ExperimentItem:
			_, err = fmt.Fprintf(w, p.format, obj.Name)
		case *experiments.TrialItem:
			_, err = fmt.Fprintf(w, p.format, experiments.JoinTrialName(obj.Experiment, obj.Number))
		}
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(obj)
}
