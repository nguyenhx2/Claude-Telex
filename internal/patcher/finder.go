package patcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FindCliJS locates the cli.js file of the @anthropic-ai/claude-code npm package.
func FindCliJS() (string, error) {
	candidates := searchDirs()
	for _, dir := range candidates {
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(dir, "@anthropic-ai", "claude-code", "cli.js"))
		if err == nil && len(matches) > 0 {
			return matches[0], nil
		}
		// Walk subdirectories one level (for scoped modules nested inside node_modules)
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name(), "@anthropic-ai", "claude-code", "cli.js")
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}
	return "", ErrNotFound
}

func searchDirs() []string {
	var dirs []string

	// 1. npm root -g
	if out, err := exec.Command("npm", "root", "-g").Output(); err == nil {
		dirs = append(dirs, strings.TrimSpace(string(out)))
	}

	// 2. Platform-specific defaults
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			dirs = append(dirs, filepath.Join(appData, "npm", "node_modules"))
		}
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			dirs = append(dirs,
				filepath.Join(localAppData, "npm", "node_modules"),
				filepath.Join(localAppData, "nvm", "current", "node_modules"),
			)
		}
	case "darwin":
		home, _ := os.UserHomeDir()
		dirs = append(dirs,
			"/usr/local/lib/node_modules",
			"/opt/homebrew/lib/node_modules",
			filepath.Join(home, ".nvm", "versions"),
			filepath.Join(home, ".fnm", "node-versions"),
		)
	default: // linux
		home, _ := os.UserHomeDir()
		dirs = append(dirs,
			"/usr/lib/node_modules",
			"/usr/local/lib/node_modules",
			filepath.Join(home, ".nvm", "versions"),
			filepath.Join(home, ".local", "lib", "node_modules"),
		)
	}

	// 3. PATH-based: find node → sibling lib/node_modules
	if nodePath, err := exec.LookPath("node"); err == nil {
		nodeDir := filepath.Dir(nodePath)
		dirs = append(dirs,
			filepath.Join(nodeDir, "..", "lib", "node_modules"),
			filepath.Join(nodeDir, "node_modules"),
		)
	}

	return dirs
}
