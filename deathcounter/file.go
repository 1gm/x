package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func newFileDeathCounter(name string) (*fileDeathCounter, error) {
	var count int64
	var err error

	if fileExists(name) {
		if count, err = parseDeathCounterFromFile(name); err != nil {
			return nil, err
		}
	} else if err = createFile(name); err != nil {
		return nil, err
	}

	return &fileDeathCounter{
		name: name,
		c:    newCounter(count),
	}, nil
}

type fileDeathCounter struct {
	name string
	c    *counter
}

func (f fileDeathCounter) Increment() error {
	if err := ioutil.WriteFile(f.name, []byte(fmt.Sprintf("Deaths: %d", f.c.Increment())), 0644); err != nil {
		return err
	}
	return nil
}

func (f fileDeathCounter) Decrement() error {
	if err := ioutil.WriteFile(f.name, []byte(fmt.Sprintf("Deaths: %d", f.c.Decrement())), 0644); err != nil {
		return err
	}
	return nil
}

func parseDeathCounterFromFile(name string) (int64, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return 0, err
	}

	num := strings.TrimPrefix(strings.ToLower(string(b)), "deaths: ")
	return strconv.ParseInt(num, 10, 64)
}

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

func createFile(name string) error {
	if err := mkdirAll(filepath.Dir(name)); err != nil {
		return err
	}

	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}
