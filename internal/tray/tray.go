// Package tray manages the system tray icon and menu.
package tray

import (
	"fmt"
	"log"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/nguyenhx2/claude-telex/internal/icon"
	"github.com/nguyenhx2/claude-telex/internal/patcher"
	"github.com/nguyenhx2/claude-telex/internal/settings"
	"github.com/nguyenhx2/claude-telex/internal/state"
)

// Run starts the systray. Blocks until Quit is clicked.
func Run(srv *settings.Server) {
	systray.Run(func() { onReady(srv) }, nil)
}

func onReady(srv *settings.Server) {
	setIcon(state.Get().Enabled)
	systray.SetTitle("Claude TELEX")
	systray.SetTooltip("Claude TELEX — Vietnamese IME Fix")

	mSettings := systray.AddMenuItem("⚙  Mở cài đặt / Settings", "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("✕  Thoát / Quit", "")

	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				openSettings(srv)
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	go startupPatch()
}

// setIcon updates the tray icon. Uses ICO on Windows, PNG elsewhere.
func setIcon(enabled bool) {
	var iconBytes []byte
	if runtime.GOOS == "windows" {
		iconBytes = icon.ICO(enabled)
	} else {
		iconBytes = icon.PNG(enabled)
	}
	systray.SetIcon(iconBytes)

	tooltip := "Claude TELEX — Vietnamese IME Fix [TẮT]"
	if enabled {
		tooltip = "Claude TELEX — Vietnamese IME Fix [BẬT]"
	}
	systray.SetTooltip(tooltip)
}

// UpdateIcon refreshes the tray icon to match current state.
func UpdateIcon() { setIcon(state.Get().Enabled) }

func startupPatch() {
	st := state.Get()
	if !st.Enabled {
		return
	}
	path := st.PatchPath
	if path == "" {
		var err error
		path, err = patcher.FindCliJS()
		if err != nil {
			log.Printf("startup: %v", err)
			return
		}
		state.Update(func(s *state.State) { s.PatchPath = path })
	}
	if patcher.IsPatched(path) {
		ver := patcher.ClaudeVersion(path)
		if ver != st.LastPatchedVersion {
			log.Printf("version changed %s→%s, re-patching", st.LastPatchedVersion, ver)
			_ = patcher.Restore(path)
			if err := patcher.Patch(path); err == nil {
				state.Update(func(s *state.State) { s.LastPatchedVersion = ver })
			}
		}
		return
	}
	if err := patcher.Patch(path); err != nil {
		log.Printf("startup patch: %v", err)
		return
	}
	ver := patcher.ClaudeVersion(path)
	state.Update(func(s *state.State) { s.LastPatchedVersion = ver })
	log.Printf("patched %s (%s)", path, ver)
}

// TogglePatch toggles enabled state; called from global hotkey.
func TogglePatch() {
	st := state.Get()
	enabled := !st.Enabled
	path := st.PatchPath
	if path == "" {
		var err error
		path, err = patcher.FindCliJS()
		if err != nil {
			log.Printf("toggle: %v", err)
			return
		}
		state.Update(func(s *state.State) { s.PatchPath = path })
	}
	if enabled {
		_ = patcher.Patch(path)
	} else {
		_ = patcher.Restore(path)
	}
	state.Update(func(s *state.State) { s.Enabled = enabled })
	setIcon(enabled)
	msg := fmt.Sprintf("Vietnamese Fix: %s", map[bool]string{true: "BẬT ✓", false: "TẮT"}[enabled])
	systray.SetTooltip(msg)
}
