//go:build !windows && !darwin

package settings

import (
	"os"
	"strings"
)

// osInfoPlatform returns Linux distro from /etc/os-release.
func osInfoPlatform() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(line[12:], `"`)
		}
	}
	return "Linux"
}
