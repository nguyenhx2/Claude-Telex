//go:build !windows

package tray

import (
	"os/exec"
	"runtime"
)

// openSettings opens the Settings UI in the default browser on non-Windows platforms.
func openSettings(srv interface{ URL() string }) {
	OpenURL(srv.URL())
}

// OpenURL opens a URL in the default browser.
func OpenURL(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	default:
		_ = exec.Command("xdg-open", url).Start()
	}
}
