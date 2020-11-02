// +build windows

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

func mkdirAll(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0666); err != nil {
			return err
		}
	}
	return nil
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		return false
	}

	return true
}

func touchFile(name string) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

func deathCounter(name string) (increment func(), decrement func(), err error) {
	var deathCounter int64

	if fileExists(name) {
		b, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, nil, err
		}

		num := strings.TrimPrefix(strings.ToLower(string(b)), "deaths: ")
		deathCounter, err = strconv.ParseInt(num, 10, 64)
		if err != nil {
			return nil, nil, err
		}
	} else {
		if err = mkdirAll(filepath.Dir(name)); err != nil {
			return nil, nil, err
		}
		if err = touchFile(name); err != nil {
			return nil, nil, err
		}
	}

	increment = func() {
		newCounter := atomic.AddInt64(&deathCounter, 1)
		if err := ioutil.WriteFile(name, []byte(fmt.Sprintf("Deaths: %d", newCounter)), 0644); err != nil {
			log.Println("ERROR: ", err)
		}
	}

	decrement = func() {
		newCounter := atomic.AddInt64(&deathCounter, -1)
		if err := ioutil.WriteFile(name, []byte(fmt.Sprintf("Deaths: %d", newCounter)), 0644); err != nil {
			log.Println("ERROR: ", err)
		}
	}

	return increment, decrement, nil
}
