//go:build darwin

package settings

import (
	"os/exec"
	"strings"
)

// osInfoPlatform returns macOS version via sw_vers.
func osInfoPlatform() string {
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		return "macOS " + strings.TrimSpace(string(out))
	}
	return "macOS"
}
