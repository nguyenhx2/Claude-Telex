//go:build windows

package tray

import (
	"syscall"
	"unsafe"
)

var (
	shell32       = syscall.NewLazyDLL("shell32.dll")
	shellExecuteW = shell32.NewProc("ShellExecuteW")
)

// openSettings opens the Settings UI in the default browser using ShellExecuteW.
// This avoids spawning cmd.exe (which causes a visible console flash).
func openSettings(srv interface{ URL() string }) {
	OpenURL(srv.URL())
}

// OpenURL opens a URL in the default browser without spawning a cmd.exe window.
func OpenURL(url string) {
	urlPtr, _ := syscall.UTF16PtrFromString(url)
	verbPtr, _ := syscall.UTF16PtrFromString("open")
	//nolint:errcheck
	shellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(urlPtr)),
		0, 0,
		uintptr(syscall.SW_SHOWNORMAL),
	)
}
