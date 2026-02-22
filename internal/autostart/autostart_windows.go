//go:build windows

package autostart

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const regKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
const regName = "ClaudeTelex"

// Enable adds or removes the app from Windows Startup.
func Enable(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry: %w", err)
	}
	defer k.Close()

	if !enable {
		return k.DeleteValue(regName)
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)
	return k.SetStringValue(regName, fmt.Sprintf(`"%s"`, exe))
}

// IsEnabled reports whether the app is in Windows Startup.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(regName)
	return err == nil
}
