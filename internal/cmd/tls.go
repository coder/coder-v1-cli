package cmd

import (
	"fmt"
	"os"

	"github.com/cdr/grip"
	"github.com/spf13/cobra"
	"golang.org/x/xerrors"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/internal/coderutil"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/pkg/tablewriter"
)

type ErrMissingRequiredParameter struct {
	Name string
}

func (e ErrMissingRequiredParameter) Error() string {
	return fmt.Sprintf("Missing required parameter '%s'", e.Name)
}

func tlsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tls",
		Short: "Manage Coder TLS configuration",
		Long:  "Manage Coder TLS configuration via self-signed certificates, custom certificates, or Let's Encrypt certificates",
	}

	cmd.AddCommand(
		tlsSelfSign(),
		tlsCustom(),
		tlsACME(),
		tlsDisable(),
	)
	return cmd
}

func tlsSelfSign() *cobra.Command {
	var (
		hosts []string
	)
	cmd := &cobra.Command{
		Use:     "self-sign",
		Short:   "Generate self-signed certificate for Coder",
		Long:    "Generate self-signed certificates for Coder. Self-signed certificates are automatically renewed by Coder",
		Example: "tls self-sign --hosts a.example.com --hosts b.example.com --hosts c.example.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			if len(hosts) == 0 {
				return ErrMissingRequiredParameter{"hosts"}
			}

			resp, err := client.GenerateSelfSignedCertificate(ctx, coder.SelfSignedCertificateRequest{
				Hosts: hosts,
			})
			if err != nil {
				return err
			}

			if !resp.Success {
				return xerrors.New("Failed to generate self-signed certificate")
			}

			clog.LogSuccess("Generated self-signed certificate")
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&hosts, "hosts", []string{}, "Hostnames and/or IPs to generate self-signed certificate for")

	return cmd
}

func tlsCustom() *cobra.Command {
	var (
		certFilepath string
		keyFilepath  string
	)

	cmd := &cobra.Command{
		Use:     "custom",
		Short:   "Upload custom PEM encoded certificate and key files",
		Example: "tls custom --cert /path/to/cert.pem --key /path/to/key.pem",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			cert, err := os.ReadFile(certFilepath)
			if err != nil {
				return err
			}

			key, err := os.ReadFile(keyFilepath)
			if err != nil {
				return err
			}

			req := coder.UploadCertificateRequest{
				Certificate: cert,
				PrivateKey:  key,
			}

			resp, err := client.UploadCustomCertificate(ctx, req)
			if err != nil {
				return err
			}

			if !resp.Success {
				return xerrors.New("Failed to apply custom TLS certificate")
			}

			clog.LogSuccess("Uploaded custom TLS certificate")
			return nil
		},
	}

	cmd.Flags().StringVar(&certFilepath, "cert", "", "Full path to a PEM encoded certificate file")
	cmd.Flags().StringVar(&keyFilepath, "key", "", "Full path to a PEM encoded private key file")
	return cmd
}

func tlsACME() *cobra.Command {
	var (
		showProviderInfo bool

		agreeTOS    bool
		email       string
		domains     []string
		dnsProvider string
		credentials map[string]string
	)
	cmd := &cobra.Command{
		Use:   "acme",
		Short: "Generate certificate via Let's Encrypt",
		Example: `
tls acme --info
tls acme --email me@example.com --domains a.example.com --domains b.example.com --provider route53 --credentials AWS_ACCESS_KEY_ID=your-key-id --credentials AWS_SECRET_ACCESS_KEY=your-secret-key --credentials AWS_REGION=your-region`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			if showProviderInfo {
				providers, err := client.GetLetsEncryptProviders(ctx)
				if err != nil {
					return err
				}
				if err = tablewriter.WriteTable(cmd.OutOrStdout(), len(providers.Providers), func(i int) interface{} {
					return coderutil.ProviderInfosTable(providers.Providers[i])
				}); err != nil {
					return err
				}
				return nil
			}

			catcher := grip.NewBasicCatcher()
			if !agreeTOS {
				catcher.Add(ErrMissingRequiredParameter{"agree-tos"})
			}
			if email == "" {
				catcher.Add(ErrMissingRequiredParameter{"email"})
			}
			if len(domains) == 0 {
				catcher.Add(ErrMissingRequiredParameter{"domains"})
			}
			if dnsProvider == "" {
				catcher.Add(ErrMissingRequiredParameter{"provider"})
			}
			if catcher.HasErrors() {
				return catcher.Resolve()
			}

			req := coder.GenerateLetsEncryptCertRequest{
				Email:       email,
				Domains:     domains,
				DNSProvider: dnsProvider,
				Credentials: credentials,
			}

			resp, err := client.GenerateLetsEncryptCert(ctx, req)
			if err != nil {
				return err
			}

			if !resp.Success {
				return xerrors.New("Failed to generate certificate via Let's Encrypt")
			}

			clog.LogSuccess("Generating certificate via Let's Encrypt. This may take a few minutes to complete.")
			return nil
		},
	}

	cmd.Flags().BoolVar(&showProviderInfo, "info", false, "Show supported DNS providers and required credentials for each")
	cmd.Flags().BoolVar(&agreeTOS, "agree-tos", false, "Agree to ACME Terms of Service")
	cmd.Flags().StringVar(&email, "email", "e", "Email to use for ACME account")
	cmd.Flags().StringArrayVar(&domains, "domains", []string{}, "Domains to request certificates for")
	cmd.Flags().StringVar(&dnsProvider, "provider", "", "DNS provider hosting your domains")
	cmd.Flags().StringToStringVar(&credentials, "credentials", map[string]string{}, "DNS provider credentials")
	return cmd
}

func tlsDisable() *cobra.Command {
	var (
		force bool
	)
	cmd := &cobra.Command{
		Use:     "disable",
		Short:   "Delete TLS certificates from Coder, effectively disabling https access",
		Example: "tls disable",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}

			resp, err := client.DisableTLS(ctx, force)
			if err != nil {
				return err
			}

			catcher := grip.NewBasicCatcher()
			if !resp.DeletedCertificate {
				catcher.Add(xerrors.New("Failed to delete certificates from Coder"))
			}
			if resp.CARevocationFailed {
				catcher.Add(xerrors.New("Revocation at the Certificate Authority failed"))
			}
			if catcher.HasErrors() {
				return catcher.Resolve()
			}

			clog.LogSuccess("Deleted TLS certificates from Coder")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "For Let's Encrypt certificates only: delete the certificate from Coder even if revocation at the Certificate Authority fails")

	return cmd
}
