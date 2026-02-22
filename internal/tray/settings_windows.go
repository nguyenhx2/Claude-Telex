//go:build windows

package tray

import "os/exec"

// openSettings opens the Settings UI in the default browser on Windows.
// To use a native webview window instead, install MinGW and build with CGO_ENABLED=1.
func openSettings(srv interface{ URL() string }) {
	_ = exec.Command("cmd", "/c", "start", srv.URL()).Start()
}
