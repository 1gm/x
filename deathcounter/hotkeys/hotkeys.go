// +build windows

package hotkeys

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

func Register(modifiers int, keyCode int, callback func() error) error {
	hotKey := &hotKey{
		id:        int16(nextKeyID()),
		callback:  callback,
		modifiers: modifiers,
		keyCode:   keyCode,
	}
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
				if err := hotKey.callback(); err != nil {
					errorHandler(err)
				}
			}
		}

		select {
		case <-time.After(50 * time.Millisecond):
		case <-c:
			log.Println("INFO: Shutting down")
			if err := user32.Release(); err != nil {
				errorHandler(err)
			}
			return
		}

	}
}

func SetErrorHandler(fn func(error)) {
	errorHandler = fn
}

var errorHandler = func(err error) {
	log.Println("ERROR: ", err.Error())
}

var (
	hotKeyLookup = make(map[int16]*hotKey)

	user32         = syscall.MustLoadDLL("user32")
	registerHotKey = user32.MustFindProc("RegisterHotKey")
	peekMessageW   = user32.MustFindProc("PeekMessageW")
)

type hotKey struct {
	// id represents the identifier of the hot key as passed to the RegisterHotKey function.
	id int16

	callback func() error

	modifiers int
	keyCode   int
}

func (h hotKey) String() string {
	var buf bytes.Buffer
	if h.modifiers&ModAlt != 0 {
		buf.WriteString("Alt+")
	}
	if h.modifiers&ModCtrl != 0 {
		buf.WriteString("Ctrl+")
	}
	if h.modifiers&ModShift != 0 {
		buf.WriteString("Shift+")
	}
	if h.modifiers&ModWin != 0 {
		buf.WriteString("Win+")
	}
	return fmt.Sprintf("HotKey[%s%c]", buf.String(), h.keyCode)
}

var keyID int32

func nextKeyID() int32 {
	return atomic.AddInt32(&keyID, 1)
}
