package cmd

import (
	"fmt"
	"strings"
	"time"

	"cdr.dev/coder-cli/coder-sdk"
	"cdr.dev/coder-cli/pkg/clog"
	"cdr.dev/coder-cli/wsnet"
	"github.com/fatih/color"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/cobra"
	"nhooyr.io/websocket"
)

func pingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ping [workspace_name]",
		Short:   "Ping a Coder workspace",
		Example: `coder ping my-dev`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			client, err := newClient(ctx, true)
			if err != nil {
				return err
			}
			workspace, err := findWorkspace(ctx, client, args[0], coder.Me)
			if err != nil {
				return err
			}
			if workspace.LatestStat.ContainerStatus != coder.WorkspaceOn {
				return clog.Error("workspace not available",
					fmt.Sprintf("current status: \"%s\"", workspace.LatestStat.ContainerStatus),
					clog.BlankLine,
					clog.Tipf("use \"coder workspaces rebuild %s\" to rebuild this workspace", workspace.Name),
				)
			}
			servers, err := client.ICEServers(ctx)
			if err != nil {
				return err
			}
			url := client.BaseURL()
			connectionStart := time.Now()
			dialer, err := wsnet.DialWebsocket(ctx, wsnet.ConnectEndpoint(&url, workspace.ID, client.Token()), &wsnet.DialOptions{
				ICEServers:         servers,
				TURNProxyAuthToken: client.Token(),
				TURNRemoteProxyURL: &url,
				TURNLocalProxyURL:  &url,
			}, &websocket.DialOptions{})
			if err != nil {
				return err
			}
			connectionMS := float64(time.Since(connectionStart).Microseconds()) / 1000
			candidates, err := dialer.Candidates()
			if err != nil {
				return err
			}
			relay := candidates.Local.Typ == webrtc.ICECandidateTypeRelay
			tunneled := false
			properties := []string{}
			candidateURLs := []string{}

			for _, server := range servers {
				if server.Username == wsnet.TURNProxyICECandidate().Username {
					candidateURLs = append(candidateURLs, fmt.Sprintf("turns:%s", url.Host))
					if !relay {
						continue
					}
					tunneled = true
					continue
				}

				candidateURLs = append(candidateURLs, server.URLs...)
			}
			properties = append(properties, fmt.Sprintf("candidates=%s", strings.Join(candidateURLs, ",")))

			connectionText := "direct via STUN"
			if relay {
				connectionText = "proxied via TURN"
			}
			if tunneled {
				connectionText = fmt.Sprintf("proxied via %s", url.Host)
			}

			fmt.Printf("%s %s %s (%s) %s\n",
				color.New(color.Bold, color.FgWhite).Sprint("PING"),
				workspace.Name,
				color.New(color.Bold, color.FgGreen).Sprintf("connected in %.2fms", connectionMS),
				connectionText,
				strings.Join(properties, " "),
			)

			ticker := time.NewTicker(time.Second)
			seq := 1
			for {
				select {
				case <-ticker.C:
					start := time.Now()
					err := dialer.Ping(ctx)
					if err != nil {
						return err
					}
					pingMS := float64(time.Since(start).Microseconds()) / 1000
					connectionText = "you ↔ workspace"
					if tunneled {
						connectionText = fmt.Sprintf("you ↔ %s ↔ workspace", url.Host)
					}

					fmt.Printf("%.2fms (%s) seq=%d\n",
						pingMS,
						connectionText,
						seq)
					seq++
				case <-ctx.Done():
					return nil
				}
			}
		},
	}

	return cmd
}
