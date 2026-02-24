// Package tray manages the system tray icon and menu.
package tray

import (
	"log"
	"runtime"
	"time"

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
	refreshIcon()
	systray.SetTitle("Claude TELEX")
	systray.SetTooltip("Claude TELEX - Vietnamese IME Fix")

	mSettings := systray.AddMenuItem("\u2699  Settings", "Open Settings UI")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("\u2715  Quit", "")

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

// refreshIcon updates the tray icon and tooltip.
func refreshIcon() {
	st := state.Get()

	iconState := icon.StateOff
	tooltip := "Claude TELEX - Vietnamese IME Fix [OFF]"

	if st.Enabled {
		if st.PatchPath != "" && !patcher.IsPatched(st.PatchPath) {
			iconState = icon.StateUpdate
			tooltip = "Claude TELEX - Needs Re-patch (Update Detected)"
		} else {
			iconState = icon.StateOn
			tooltip = "Claude TELEX - Vietnamese IME Fix [ON]"
		}
	}

	var iconBytes []byte
	if runtime.GOOS == "windows" {
		iconBytes = icon.ICO(iconState)
	} else {
		iconBytes = icon.PNG(iconState)
	}
	systray.SetIcon(iconBytes)
	systray.SetTooltip(tooltip)
}

// UpdateIcon refreshes the tray icon to match current state.
func UpdateIcon() { refreshIcon() }

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
		return
	}

	// First time install auto-patch (LastPatchedVersion is empty)
	if st.LastPatchedVersion == "" {
		if err := patcher.Patch(path); err != nil {
			log.Printf("startup patch: %v", err)
			return
		}
		ver := patcher.ClaudeVersion(path)
		state.Update(func(s *state.State) { s.LastPatchedVersion = ver })
		log.Printf("patched %s (%s)", path, ver)
		refreshIcon()
	} else {
		// Enabled = true, IsPatched = false, LastPatchedVersion != ""
		// This means Claude Code updated. Wait for user to repatch manually.
		log.Printf("Update detected (missing patch). Prompting user.")
		refreshIcon()
	}
}

func backgroundUpdateCheck() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		refreshIcon()
	}
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
	refreshIcon()
}
