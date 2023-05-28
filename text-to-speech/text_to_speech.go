package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/1gm/x/internal/log"
)

func main() {
	in := flag.String("i", "input", "directory to read input files from")
	flag.Parse()

	os.Exit(realMain(*in))
}

func realMain(inputDirectory string) int {
	log := log.New()
	defer log.Sync()

	if created, err := createDirectories(inputDirectory); err != nil {
		log.Error(err)
		return 1
	} else if created {
		log.Infof("created directory %v to process input files", inputDirectory)
	} else {
		log.Infof("watching directory %v for input files", inputDirectory)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	watchFiles := make(chan string)
	watchErr := watchDirectory(ctx, inputDirectory, watchFiles)
	for {
		select {
		case inputFile, ok := <-watchFiles:
			if !ok {
				continue
			}
			log.Infof("received %v", inputFile)
		case werr, ok := <-watchErr:
			if !ok {
				continue
			}
			log.Error(werr)
		case <-ctx.Done():
			log.Info("shutting down")
			return 0
		}
	}
}

// createDirectories will make the inputDirectory and it's subdirectory, _processed.
// If the directory was created then a (true, nil) is returned.
func createDirectories(inputDirectory string) (created bool, err error) {
	inputDirectoryFull := filepath.Join(inputDirectory, "_processed")
	if fi, err := os.Stat(inputDirectoryFull); err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(inputDirectoryFull, 0777); err != nil {
				return false, fmt.Errorf("failed to create directory: %v", err)
			}
			return true, nil
		}
	} else if !fi.IsDir() {
		return false, fmt.Errorf("%q must be a directory", inputDirectory)
	}
	return false, nil
}

func watchDirectory(ctx context.Context, inputDirectory string, resultCh chan<- string) <-chan error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		for {
			// TODO(george): Watch for new files passed into input directory
			resultCh <- time.Now().Format(time.RFC3339)
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()
	return errCh
}
