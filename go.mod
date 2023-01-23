module cdr.dev/coder-cli

go 1.14

// TODO: remove the replace once this PR gets merged:
// https://github.com/pion/webrtc/pull/1946
replace github.com/pion/webrtc/v3 => github.com/deansheather/webrtc/v3 v3.1.0-beta.6.0.20210907233552-57c66b872d12

require (
	cdr.dev/slog v1.4.1
	cdr.dev/wsep v0.1.0
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/briandowns/spinner v1.16.0
	github.com/cli/safeexec v1.0.0
	github.com/fatih/color v1.14.0
	github.com/google/go-cmp v0.5.6
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/yamux v0.0.0-20210826001029-26ff87cf9493
	github.com/kirsle/configdir v0.0.0-20170128060238-e45d2f54772f
	github.com/klauspost/compress v1.13.5 // indirect
	github.com/manifoldco/promptui v0.8.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/pion/datachannel v1.4.21
	github.com/pion/dtls/v2 v2.0.9
	github.com/pion/ice/v2 v2.1.12
	github.com/pion/logging v0.2.2
	github.com/pion/turn/v2 v2.0.5
	github.com/pion/webrtc/v3 v3.1.0-beta.7
	github.com/pkg/browser v0.0.0-20210904010418-6d279e18f982
	github.com/rjeczalik/notify v0.9.2
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/net v0.0.0-20210907225631-ff17edfbf26d
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.3.0
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	nhooyr.io/websocket v1.8.7
)
