// claude-telex: Vietnamese IME fix for Claude Code CLI
package main

import (
	"log"
	"runtime"

	"github.com/nguyenhx2/claude-telex/internal/hotkey"
	"github.com/nguyenhx2/claude-telex/internal/patcher"
	"github.com/nguyenhx2/claude-telex/internal/settings"
	"github.com/nguyenhx2/claude-telex/internal/state"
	"github.com/nguyenhx2/claude-telex/internal/tray"
)

func main() {
	// Must lock OS thread for systray (macOS requires main thread)
	runtime.LockOSThread()

	// Single-instance: if another claude-telex is already running,
	// focus its Settings UI and exit silently (no cmd.exe flash).
	if existingURL := settings.PingRunningInstance(); existingURL != "" {
		log.Println("Another instance is running, opening its Settings UI...")
		tray.OpenURL(existingURL)
		return
	}

	// Find cli.js on first run
	st := state.Get()
	if st.PatchPath == "" {
		if path, err := patcher.FindCliJS(); err == nil {
			state.Update(func(s *state.State) { s.PatchPath = path })
		} else {
			log.Printf("warning: %v", err)
		}
	}

	// Start Settings HTTP server
	srv, err := settings.Start()
	if err != nil {
		log.Fatalf("settings server: %v", err)
	}
	defer srv.Shutdown()
	log.Printf("settings UI at %s", srv.URL())

	// Global hotkey Ctrl+Alt+V in background
	stopHK := make(chan struct{})
	go hotkey.Listen(tray.TogglePatch, stopHK)
	defer close(stopHK)

	// Tray blocks until Quit
	tray.Run(srv)
}
