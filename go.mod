module cdr.dev/coder-cli

go 1.14

replace github.com/pion/turn/v2 => github.com/deansheather/turn/v2 v2.0.6-0.20210908222112-8e1286eedccd

require (
	cdr.dev/slog v1.4.1
	cdr.dev/wsep v0.1.0
	github.com/briandowns/spinner v1.16.0
	github.com/cli/safeexec v1.0.0
	github.com/fatih/color v1.12.0
	github.com/google/go-cmp v0.5.6
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/yamux v0.0.0-20210316155119-a95892c5f864
	github.com/kirsle/configdir v0.0.0-20170128060238-e45d2f54772f
	github.com/klauspost/compress v1.10.8 // indirect
	github.com/manifoldco/promptui v0.8.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/pion/datachannel v1.5.0
	github.com/pion/dtls/v2 v2.0.10
	github.com/pion/ice/v2 v2.1.13
	github.com/pion/logging v0.2.2
	github.com/pion/turn/v2 v2.0.5
	github.com/pion/webrtc/v3 v3.1.7
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/rjeczalik/notify v0.9.2
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/goleak v1.1.12
	golang.org/x/net v0.0.0-20211020060615-d418f374d309
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211025201205-69cdffdb9359
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	nhooyr.io/websocket v1.8.7
)
