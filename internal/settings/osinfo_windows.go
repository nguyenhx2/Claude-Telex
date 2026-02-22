//go:build windows

package settings

import (
	"fmt"
	"strconv"

	"golang.org/x/sys/windows/registry"
)

// osInfoPlatform reads OS version directly from the Windows registry,
// avoiding any exec.Command("cmd", ...) calls that would flash a console window.
func osInfoPlatform() string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows"
	}
	defer k.Close()

	buildStr, _, err := k.GetStringValue("CurrentBuildNumber")
	if err != nil {
		return "Windows"
	}

	build, _ := strconv.Atoi(buildStr)
	name := "10"
	if build >= 22000 {
		name = "11"
	}

	// Try to get display version like "23H2"
	if dv, _, err := k.GetStringValue("DisplayVersion"); err == nil && dv != "" {
		return fmt.Sprintf("Windows %s %s (build %s)", name, dv, buildStr)
	}

	return fmt.Sprintf("Windows %s (build %s)", name, buildStr)
}
