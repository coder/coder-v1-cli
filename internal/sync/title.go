package sync

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh/terminal"
)

func setConsoleTitle(title string) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return
	}
	fmt.Printf("\033]0;%s\007", title)
}


func fmtUpdateTitle(path string) string {
	return "🚀 updating " + filepath.Base(path)
}