package sync

import (
	"fmt"
	"path/filepath"
)

func setConsoleTitle(title string, isInteractiveOutput bool) {
	if !isInteractiveOutput {
		return
	}
	fmt.Printf("\033]0;%s\007", title)
}

func fmtUpdateTitle(path string) string {
	return "🚀 updating " + filepath.Base(path)
}
