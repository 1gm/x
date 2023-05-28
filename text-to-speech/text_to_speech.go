package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/1gm/x/internal/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awspolly "github.com/aws/aws-sdk-go/service/polly"
	"github.com/aws/aws-sdk-go/service/polly/pollyiface"
	"go.uber.org/zap"
)

func main() {
	in := flag.String("i", "input", "directory to read input files from")
	flag.Parse()

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	polly := awspolly.New(sess)

	os.Exit(realMain(*in, polly))
}

func realMain(inputDirectory string, polly pollyiface.PollyAPI) int {
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

	// we pipe the results from the watchDirectory worker (watchFiles) into the text processor.
	watchFiles, watchErr := watchDirectory(ctx, inputDirectory)
	pollyResults, pollyErr := processText(ctx, log, polly, watchFiles)

	var err error
	var exitCode int
L:
	for {
		if err != nil {
			break L
		}

		select {
		case result, ok := <-pollyResults:
			if !ok {
				continue
			}
			log.Infof("processed %v", result.name)
		case werr, ok := <-watchErr:
			if !ok {
				continue
			}
			err = werr
			exitCode = 1
		case perr, ok := <-pollyErr:
			if !ok {
				continue
			}
			err = perr
			exitCode = 1
		case <-ctx.Done():
			err = ctx.Err()
		}
	}

	if !errors.Is(err, context.Canceled) {
		log.Error(err)
	}

	log.Info("shutting down")
	return exitCode
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
func watchDirectory(ctx context.Context, inputDirectory string) (<-chan fileInfo, <-chan error) {
	errCh := make(chan error)
	resultCh := make(chan fileInfo)

	go func() {
		defer close(errCh)
		defer close(resultCh)

		var latestModTime int64

		for {
			results, err := getFileInfosSince(inputDirectory, latestModTime)
			if err != nil {
				errCh <- err
				return
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
	return resultCh, errCh
}

// fileInfo captures some information about files (not sure if all of it is relevant).
type fileInfo struct {
	absolutePath string
	path         string
	name         string
	ext          string
	mod          int64
	fi           fs.FileInfo
}

func (fi fileInfo) String() string {
	return fmt.Sprintf(`absolutePath: %s path: %s name: %s ext: %s mod: %d`, fi.absolutePath, fi.path, fi.name, fi.ext, fi.mod)
}

// getFileInfosSince gets all the file infos in inputDirectory with mod times later than sinceModTime while skipping
// files in the _processed directory.
func getFileInfosSince(inputDirectory string, sinceModTime int64) ([]fileInfo, error) {
	var fis []fileInfo
	dirFS := os.DirFS(inputDirectory)
	err := fs.WalkDir(dirFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}

		if d.Name() == "_processed" {
			return fs.SkipDir
		}

		modTime := fi.ModTime().Unix()
		if modTime <= sinceModTime {
			return nil
		}

		if !d.IsDir() {
			realPath := filepath.Join(inputDirectory, path)
			absolutePath, err := filepath.Abs(realPath)
			if err != nil {
				return err
			}

			fis = append(fis, fileInfo{
				absolutePath: absolutePath,
				path:         realPath,
				name:         fi.Name(),
				ext:          filepath.Ext(fi.Name()),
				mod:          modTime,
				fi:           fi,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return fis, nil
}

func processText(ctx context.Context, log *zap.SugaredLogger, polly pollyiface.PollyAPI, inputFiles <-chan fileInfo) (<-chan fileInfo, <-chan error) {
	errCh := make(chan error)
	resultCh := make(chan fileInfo)
	go func() {
		defer close(errCh)
		defer close(resultCh)

		for {
			select {
			case inputFile, ok := <-inputFiles:
				if !ok {
					return
				}
				log.Infof("polly processor received: %s", inputFile)

				// TODO(george): Invoke polly here - rough idea:
				// 1. read the file contents
				inputBytes, err := os.ReadFile(inputFile.absolutePath)
				if err != nil {

					errCh <- fmt.Errorf("processText read input file: %v", err)
					return
				}
				inputText := string(inputBytes)

				// 2. invoke polly
				pollyInput := &awspolly.SynthesizeSpeechInput{
					OutputFormat: aws.String("mp3"),
					Text:         &inputText,
					VoiceId:      aws.String("Joanna"),
				}

				pollyOutput, err := polly.SynthesizeSpeech(pollyInput)
				if err != nil {
					errCh <- fmt.Errorf("processText synthesize text: %v", err)
					return
				}

				// 3. save the result from polly into a file in the '_processed' with a .mp3 suffix
				outputFilePath := strings.TrimSuffix(inputFile.absolutePath, inputFile.name)
				outputFilePath = filepath.Join(outputFilePath, "_processed", inputFile.name+".mp3")
				outputFile, err := os.Create(outputFilePath)
				if err != nil {
					errCh <- fmt.Errorf("processText failed to create output file: %v", err)
					return
				}

				_, err = io.Copy(outputFile, pollyOutput.AudioStream)
				if err != nil {
					errCh <- fmt.Errorf("processText copy audio stream to output file: %v", err)
					return
				}

				if err = outputFile.Close(); err != nil {
					errCh <- fmt.Errorf("processText failed to close output file: %v", err)
					return
				}

				resultCh <- inputFile
			case <-ctx.Done():
				return
			}
		}
	}()
	return resultCh, errCh
}
