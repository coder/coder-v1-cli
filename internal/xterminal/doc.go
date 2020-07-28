// Package xterminal provides functions to change termios or console attributes
// and restore them later on. It supports Unix and Windows.
//
// This does the same thing as x/crypto/ssh/terminal on Linux. On Windows, it
// sets the same console modes as the terminal package but also sets
// `ENABLE_VIRTUAL_TERMINAL_INPUT` and `ENABLE_VIRTUAL_TERMINAL_PROCESSING` to
// allow for VT100 sequences in the console. This is important, otherwise Linux
// apps (with colors or ncurses) that are run through SSH or wsep get
// garbled in a Windows console.
//
// More details can be found out about Windows console modes here:
// https://docs.microsoft.com/en-us/windows/console/setconsolemode
package xterminal
