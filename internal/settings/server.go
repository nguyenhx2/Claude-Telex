// Package settings serves the embedded Settings UI and exposes a JSON API.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/nguyenhx2/claude-telex/assets/ui"
	"github.com/nguyenhx2/claude-telex/internal/autostart"
	"github.com/nguyenhx2/claude-telex/internal/patcher"
	"github.com/nguyenhx2/claude-telex/internal/state"
)

const preferredPort = 9315
const maxPortAttempts = 20

// Server serves the web UI on a fixed (or nearby) localhost port.
type Server struct {
	Port int
	srv  *http.Server
}

// Start binds to port 9315 (or the next free port) and starts serving.
// The chosen port is written to ~/.claude-telex/port for single-instance detection.
func Start() (*Server, error) {
	port, listener, err := bindPort()
	if err != nil {
		return nil, err
	}

	if err := writePortFile(port); err != nil {
		_ = listener.Close()
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(ui.FS)))
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/toggle", handleToggle)
	mux.HandleFunc("/api/autostart", handleAutostart)
	mux.HandleFunc("/api/patch", handlePatch)
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]bool{"ok": true})
	})

	srv := &http.Server{Handler: corsMiddleware(mux)}
	go func() { _ = srv.Serve(listener) }()

	return &Server{Port: port, srv: srv}, nil
}

// URL returns the local URL of the settings page.
func (s *Server) URL() string { return fmt.Sprintf("http://127.0.0.1:%d", s.Port) }

// Shutdown gracefully stops the server and removes the port file.
func (s *Server) Shutdown() {
	_ = s.srv.Shutdown(context.Background())
	_ = os.Remove(portFilePath())
}

// PingRunningInstance checks if another instance is already running.
// Returns the settings URL of the running instance, or "" if none.
func PingRunningInstance() string {
	portFile := portFilePath()
	data, err := os.ReadFile(portFile)
	if err != nil {
		return ""
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return ""
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	client := &http.Client{}
	resp, err := client.Get(url + "/api/health")
	if err != nil || resp.StatusCode != 200 {
		// Stale port file
		_ = os.Remove(portFile)
		return ""
	}
	return url
}

func bindPort() (int, net.Listener, error) {
	for i := 0; i < maxPortAttempts; i++ {
		port := preferredPort + i
		l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return port, l, nil
		}
	}
	return 0, nil, fmt.Errorf("no free port in range %d-%d", preferredPort, preferredPort+maxPortAttempts-1)
}

func portFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-telex", "port")
}

func writePortFile(port int) error {
	p := portFilePath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return os.WriteFile(p, []byte(strconv.Itoa(port)), 0o644)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

// --- API handlers ---

type statusResp struct {
	Enabled   bool   `json:"enabled"`
	Patched   bool   `json:"patched"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Autostart bool   `json:"autostart"`
	OS        string `json:"os"`
}

func handleStatus(w http.ResponseWriter, _ *http.Request) {
	st := state.Get()
	resp := statusResp{
		Enabled:   st.Enabled,
		Patched:   st.PatchPath != "" && patcher.IsPatched(st.PatchPath),
		Path:      st.PatchPath,
		Version:   patcher.ClaudeVersion(st.PatchPath),
		Autostart: autostart.IsEnabled(),
		OS:        osInfo(),
	}
	writeJSON(w, resp)
}

var osInfo = sync.OnceValue(func() string {
	switch runtime.GOOS {
	case "windows":
		if out, err := exec.Command("cmd", "/c", "ver").Output(); err == nil {
			line := strings.TrimSpace(string(out))
			// "Microsoft Windows [Version 10.0.22631.xxxx]"
			if i := strings.Index(line, "Version "); i != -1 {
				ver := strings.Trim(line[i+8:], "]")
				parts := strings.Split(ver, ".")
				if len(parts) >= 3 {
					build, _ := strconv.Atoi(parts[2])
					name := "10"
					if build >= 22000 {
						name = "11"
					}
					return fmt.Sprintf("Windows %s (build %s)", name, parts[2])
				}
			}
		}
		return "Windows"
	case "darwin":
		if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
			return "macOS " + strings.TrimSpace(string(out))
		}
		return "macOS"
	default:
		// Linux: try /etc/os-release
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(line[12:], `"`)
				}
			}
		}
		return "Linux"
	}
})

func handleToggle(w http.ResponseWriter, r *http.Request) {
	var req struct{ Enabled bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	st := state.Get()
	if req.Enabled {
		if err := patcher.Patch(st.PatchPath); err != nil && err != patcher.ErrAlreadyPatched {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
	} else {
		_ = patcher.Restore(st.PatchPath)
	}
	state.Update(func(s *state.State) { s.Enabled = req.Enabled })
	writeJSON(w, map[string]any{"ok": true})
}

func handleAutostart(w http.ResponseWriter, r *http.Request) {
	var req struct{ Enabled bool }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := autostart.Enable(req.Enabled); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	state.Update(func(s *state.State) { s.Autostart = req.Enabled })
	writeJSON(w, map[string]any{"ok": true})
}

func handlePatch(w http.ResponseWriter, _ *http.Request) {
	st := state.Get()
	path := st.PatchPath
	if path == "" {
		var err error
		path, err = patcher.FindCliJS()
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		state.Update(func(s *state.State) { s.PatchPath = path })
	}
	_ = patcher.Restore(path)
	if err := patcher.Patch(path); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	state.Update(func(s *state.State) { s.LastPatchedVersion = patcher.ClaudeVersion(path) })
	writeJSON(w, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
