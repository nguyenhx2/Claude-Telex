//go:build linux

package hotkey

import "golang.design/x/hotkey"

// altMod is Mod1 on Linux/X11 (equivalent to Alt).
var altMod = hotkey.Mod1
