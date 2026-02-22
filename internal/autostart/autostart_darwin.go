//go:build darwin

package autostart

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const plistPath = `Library/LaunchAgents/com.claude-telex.plist`

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.claude-telex</string>
  <key>ProgramArguments</key>
  <array><string>{{.Exe}}</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><false/>
</dict>
</plist>
`))

// Enable adds the LaunchAgent plist for macOS autostart.
func Enable(enable bool) error {
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, plistPath)
	if !enable {
		return os.Remove(p)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, _ = filepath.Abs(exe)
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, map[string]string{"Exe": exe}); err != nil {
		return fmt.Errorf("render plist: %w", err)
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.WriteFile(p, buf.Bytes(), 0o644)
}

// IsEnabled reports whether the LaunchAgent exists.
func IsEnabled() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, plistPath))
	return err == nil
}
