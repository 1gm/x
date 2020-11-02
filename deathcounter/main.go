// +build windows

package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"github.com/1gm/x/deathcounter/hotkeys"
)

var deathCounterFile = flag.String("input", "deathcounter.txt", "path to death counter file")

func main() {
	flag.Parse()
	exitCode := realMain()
	os.Exit(exitCode)
}

func realMain() int {
	fileDeathCounter, err := newFileDeathCounter(*deathCounterFile)
	if check(err) {
		return 1
	}

	if check(hotkeys.Register(hotkeys.ModCtrl, hotkeys.VK_NUMPAD0, fileDeathCounter.Increment)) {
		return 1
	}
	if check(hotkeys.Register(hotkeys.ModCtrl, hotkeys.VK_NUMPAD1, fileDeathCounter.Decrement)) {
		return 1
	}
	hotkeys.Poll()

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
