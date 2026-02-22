//go:build !windows && !darwin

package autostart

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

var desktopTmpl = template.Must(template.New("desktop").Parse(`[Desktop Entry]
Type=Application
Name=Claude TELEX
Exec={{.Exe}}
X-GNOME-Autostart-enabled=true
`))

func autostartDir() string {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		home, _ := os.UserHomeDir()
		cfg = filepath.Join(home, ".config")
	}
	return filepath.Join(cfg, "autostart")
}

const desktopName = "claude-telex.desktop"

// Enable writes/removes the XDG autostart .desktop file.
func Enable(enable bool) error {
	dir := autostartDir()
	p := filepath.Join(dir, desktopName)
	if !enable {
		return os.Remove(p)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)
	var buf bytes.Buffer
	if err := desktopTmpl.Execute(&buf, map[string]string{"Exe": exe}); err != nil {
		return fmt.Errorf("render desktop: %w", err)
	}
	_ = os.MkdirAll(dir, 0o755)
	return os.WriteFile(p, buf.Bytes(), 0o644)
}

// IsEnabled reports whether the .desktop file exists.
func IsEnabled() bool {
	_, err := os.Stat(filepath.Join(autostartDir(), desktopName))
	return err == nil
}
