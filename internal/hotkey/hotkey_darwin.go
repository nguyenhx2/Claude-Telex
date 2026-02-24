//go:build darwin

package hotkey

import "golang.design/x/hotkey"

// altMod is the Option modifier on macOS (equivalent to Alt).
var altMod = hotkey.ModOption
