package hotkeys

import (
	"bytes"
	"fmt"
	"sync/atomic"
)

func newHotKey(modifiers int, keyCode int, callback func()) *hotKey {
	return &hotKey{
		id:        int16(nextKeyID()),
		callback:  callback,
		modifiers: modifiers,
		keyCode:   keyCode,
	}
}

type hotKey struct {
	// id represents the identifier of the hot key as passed to the Register w32 function.
	id int16

	callback func()

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
