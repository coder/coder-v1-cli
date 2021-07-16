package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"cdr.dev/coder-cli/internal/x/xcobra"

	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"
)

const (
	satelliteKeyPath =  "/api/private/satellites/key"
)

type satelliteKeyResponse struct {
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
}

func satellitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "satellites",
		Short:  "Interact with Coder satellite deployments",
		Long:   "Perform operations on the Coder satellites for the platform.",
		Hidden: true,
	}

	cmd.AddCommand(
		createSatelliteCmd(),
		listSatellitesCmd(),
		deleteSatelliteCmd(),
	)
	return cmd
}

func createSatelliteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [name] [satellite_access_url]",
		Args:  xcobra.ExactArgs(2),
		Short: "create a new satellite.",
		Long:  "Create a new Coder satellite.",
		Example: `# create a new satellite

coder satellites create eu-west https://eu-west-coder.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				ctx = cmd.Context()
				name = args[0]
				accessURL = args[1]
			)

			client, err := newClient(ctx, true)
			if err != nil {
				return xerrors.Errorf("making coder client", err)
			}

			sURL, err := url.Parse(accessURL)
			if err != nil {
				return xerrors.Errorf("parsing satellite access url", err)
			}
			sURL.Path = satelliteKeyPath

			// Create the http request.
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, sURL.String(), nil)
			if err != nil {
				return xerrors.Errorf("create satellite request: %w", err)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return xerrors.Errorf("doing satellite request: %w", err)
			}
			defer func() { _ = res.Body.Close() }()

			if res.StatusCode > 299 {
				return fmt.Errorf("unexpected status code %d: %+v", res.StatusCode, res)
			}

			var keyRes satelliteKeyResponse
			if err := json.NewDecoder(res.Body).Decode(&keyRes); err != nil {
				return xerrors.Errorf("decode response body: %w", err)
			}

			if keyRes.Key == "" {
				return xerrors.Errorf("key field empty in response")
			}
			if keyRes.Fingerprint == "" {
				return xerrors.Errorf("fingerprint field empty in response")
			}

			fmt.Printf(`The following satellite will be created:
Name: %s

Public Key: 
%s

Fingerprint:
%s

Do you wish to continue? (y)
`, name, keyRes.Key, keyRes.Fingerprint)
			err = getConfirmation()
			if err != nil {
				return err
			}

			_, err = client.CreateSatellite(ctx, coder.CreateSatelliteReq{
				Name:      name,
				PublicKey: keyRes.Key,
			})
			if err != nil {
				return xerrors.Errorf("making create satellite request: %w", err)
			}

			clog.LogSuccess(fmt.Sprintf("satellite %s successfully created", name))

			return nil
		},
	}

	return cmd
}

func getConfirmation() error {
	var response string

	_, err := fmt.Scanln(&response)
	if err != nil {
		return xerrors.Errorf("scan line: %w", err)
	}

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		return xerrors.New("request canceled")
	}

	return nil
}

func listSatellitesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls",
		Short: "list satellites.",
		Long:  "List all Coder workspace satellites.",
		Example: `# list satellites
coder satellites ls`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client, err := newClient(ctx, true)
			if err != nil {
				return xerrors.Errorf("making coder client", err)
			}

			sats, err := client.Satellites(ctx)
			if err != nil {
				return xerrors.Errorf("get satellites request", err)
			}

			err = tablewriter.WriteTable(cmd.OutOrStdout(), len(sats.Data), func(i int) interface{} {
				return sats.Data[i]
			})
			if err != nil {
				return xerrors.Errorf("write table: %w", err)
			}

			return nil
		},
	}
	return cmd
}

func deleteSatelliteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [satellite_name]",
		Args:  xcobra.ExactArgs(1),
		Short: "remove a satellite.",
		Long:  "Remove an existing Coder satellite by name.",
		Example: `# remove an existing satellite by name
coder satellites rm my-satellite`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]

			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			sats, err := client.Satellites(ctx)
			if err != nil {
				return xerrors.Errorf("get satellites request", err)
			}

			for _, sat := range sats.Data {
				if sat.Name == name {
					err = client.DeleteSatelliteByID(ctx, sat.ID)
					if err != nil {
						return xerrors.Errorf("delete satellites request", err)
					}
					clog.LogSuccess(fmt.Sprintf("satellite %s successfully deleted", name))

					return nil
				}
			}

			return xerrors.Errorf("no satellite found by name '%s'", name)
		},
	}
	return cmd
}


