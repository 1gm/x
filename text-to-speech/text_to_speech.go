package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
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

	watchFiles := make(chan fileInfo)
	watchErr := watchDirectory(ctx, inputDirectory, watchFiles)
	for {
		select {
		case inputFile, ok := <-watchFiles:
			if !ok {
				continue
			}
			log.Info(inputFile)
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

// watchDirectory writes results to resultCh which are read from inputDirectory, errors are returned through a channel
// but are not necessarily indicative of a fatal error (haven't really figured these out yet).
func watchDirectory(ctx context.Context, inputDirectory string, resultCh chan<- fileInfo) <-chan error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		dirFS := os.DirFS(inputDirectory)
		var latestModTime int64

		for {
			results, err := getFileInfosSince(dirFS, latestModTime)
			if err != nil {
				errCh <- err
				continue
			}

			for _, result := range results {
				if result.mod > latestModTime {
					latestModTime = result.mod
				}
				resultCh <- result
			}

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				return
			}
		}
	}()
	return errCh
}

// fileInfo captures some information about files (not sure if all of it is relevant).
type fileInfo struct {
	path string
	name string
	ext  string
	mod  int64
	fi   fs.FileInfo
}

func (fi fileInfo) String() string {
	return fmt.Sprintf(`path: %s name: %s ext: %s mod: %d`, fi.path, fi.name, fi.ext, fi.mod)
}

// getFileInfosSince gets all the file infos in dirFS with mod times later than sinceModTime.
func getFileInfosSince(dirFS fs.FS, sinceModTime int64) ([]fileInfo, error) {
	var fis []fileInfo
	err := fs.WalkDir(dirFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}

		modTime := fi.ModTime().UnixNano()
		if modTime <= sinceModTime {
			return nil
		}

		if !d.IsDir() {
			fis = append(fis, fileInfo{
				path: path,
				name: fi.Name(),
				ext:  filepath.Ext(fi.Name()),
				mod:  modTime,
				fi:   fi,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fis, nil
}
