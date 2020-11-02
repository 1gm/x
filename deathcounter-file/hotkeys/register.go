// +build windows

package hotkeys

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
)

var (
	hotKeyLookup = make(map[int16]*hotKey)

	user32         = syscall.MustLoadDLL("user32")
	registerHotKey = user32.MustFindProc("RegisterHotKey")
	peekMessageW   = user32.MustFindProc("PeekMessageW")
)

func Register(modifiers int, keyCode int, fn func()) error {
	hotKey := newHotKey(modifiers, keyCode, fn)
	hotKeyLookup[hotKey.id] = hotKey
	r1, _, err := registerHotKey.Call(0, uintptr(hotKey.id), uintptr(hotKey.modifiers), uintptr(hotKey.keyCode))
	if r1 != 1 {
		return err
	}

	return nil
}

func Poll() {
	// https://docs.microsoft.com/en-us/windows/win32/api/winuser/ns-winuser-msg
	type msg struct {
		HWND   uintptr
		UINT   uintptr
		WPARAM int16
		LPARAM int64
		DWORD  int32
		POINT  struct{ X, Y int64 }
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, os.Interrupt, syscall.SIGKILL)

	for {
		var m msg
		_, _, _ = peekMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, 1)

		if id := m.WPARAM; id != 0 {
			if hotKey, ok := hotKeyLookup[id]; ok {
				hotKey.callback()
			}
		}

		select {
		case <-time.After(50 * time.Millisecond):
		case <-c:
			log.Println("INFO: Shutting down")
			if err := user32.Release(); err != nil {
				log.Println("ERROR: failed to release user32.dll: ", err)
			}
			return
		}

	}
}
