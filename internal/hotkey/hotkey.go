// Package hotkey registers a global Ctrl+Alt+V shortcut.
package hotkey

import (
	"log"

	"golang.design/x/hotkey"
)

// Listen registers Ctrl+Alt+V and calls fn each time it is pressed.
// It blocks until stopCh is closed.
func Listen(fn func(), stopCh <-chan struct{}) {
	hk := hotkey.New([]hotkey.Modifier{hotkey.ModCtrl, hotkey.ModAlt}, hotkey.KeyV)
	if err := hk.Register(); err != nil {
		log.Printf("hotkey register: %v", err)
		return
	}
	defer hk.Unregister()
	for {
		select {
		case <-stopCh:
			return
		case <-hk.Keydown():
			fn()
		}
	}
}
