// +build windows

package main

import (
	"flag"
	"os"
	"strings"

	"github.com/1gm/x/deathcounter/hotkeys"
	"github.com/1gm/x/internal/log"
)

func main() {
	var counterFile string
	flag.StringVar(&counterFile, "file", "deathcounter.txt", "path to death counter file")
	flag.StringVar(&counterFile, "f", "deathcounter.txt", "path to death counter file (short)")
	flag.Parse()

	exitCode := realMain(counterFile)
	os.Exit(exitCode)
}

func realMain(counterFilename string) int {
	l := log.New()
	defer l.Sync()
	l.Infof("using %s as counter file", counterFilename)

	fileDeathCounter, err := newFileDeathCounter(counterFilename)
	if check(err) {
		return 1
	}

	hotkeys.SetErrorHandler(func(err error) {
		l.Error(err)
	})

	if check(hotkeys.Register(hotkeys.ModCtrl, hotkeys.VK_NUMPAD0, fileDeathCounter.Increment)) {
		return 1
	}
	l.Info("Registered (CTRL + Numpad 0) to fileDeathCounter.Increment")
	if check(hotkeys.Register(hotkeys.ModCtrl, hotkeys.VK_NUMPAD1, fileDeathCounter.Decrement)) {
		return 1
	}
	l.Info("Registered (CTRL + Numpad 1) to fileDeathCounter.Decrement")

	l.Info("Begin polling...")
	if check(hotkeys.Poll()) {
		return 1
	}

	l.Info("End polling...")
	return 0
}

func check(err error, msgs ...string) bool {
	if err != nil {
		if len(msgs) > 0 {
			log.Println("ERROR: ", strings.Join(msgs, " "), " ", err)
		} else {
			log.Println("ERROR: ", err)
		}
		return true
	}
	return false
}
