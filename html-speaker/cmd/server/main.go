package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/1gm/x/internal/log"
	"github.com/go-chi/chi"
)

func main() {
	audioDir := flag.String("d", "testdata", "path to directory containing audio files")
	httpAddr := flag.Int("p", 8081, "http port to listen on (leave empty for random port assignmnt)")
	flag.Parse()

	os.Exit(realMain(*audioDir, fmt.Sprintf(":%d", *httpAddr)))
}

func realMain(audioDir string, httpAddr string) int {
	log := log.New()
	defer log.Sync()

	log.Infof("audio data directory is %s", audioDir)

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	r := chi.NewRouter()
	// server sent events
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	closeCh := make(chan bool)
	go func() {
		log.Infof("starting html-speaker server on %s", httpAddr)
		if err := http.ListenAndServe(httpAddr, r); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("error occured during http server startup: %v", err)
		}
		closeCh <- true
	}()

	var exitCode int

	select {
	case <-ctx.Done():
		if err := ctx.Err(); !errors.Is(err, context.Canceled) {
			log.Error("context canceled with invalid error: ", err)
			exitCode = 1
		}
		close(closeCh)
	case <-closeCh:
		log.Info("exiting...")
	}

	return exitCode
}
