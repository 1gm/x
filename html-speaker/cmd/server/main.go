package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/1gm/x/internal/log"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"nhooyr.io/websocket"
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

	r.Get("/ws", handleWebSocket(ctx, log))
	r.Get("/*", handleAsset(log))

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

func handleWebSocket(bgContext context.Context, log *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			log.Errorf("event websocket accept failed %s", err)
			return
		}

		log.Infof("websocket connection established")

		defer func() {
			if cerr := conn.Close(websocket.StatusInternalError, ""); cerr != nil {
				log.Errorf("failed to close websocket connection: %v", cerr)
			}
		}()

		// ignore incoming connections
		r = r.WithContext(conn.CloseRead(r.Context()))

		// write sends message with a timeout.
		writeTimeout := func(ctx context.Context, timeout time.Duration, conn *websocket.Conn, msg []byte) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			return conn.Write(ctx, websocket.MessageText, msg)
		}

		// Stream all events to outgoing websocket writer.
		for {
			select {
			case <-r.Context().Done():
				return // disconnect when HTTP connection disconnects
			case <-bgContext.Done():
				return // disconnect when the application is shutting down
			case <-time.After(time.Second * 5):
				if werr := writeTimeout(r.Context(), time.Second*3, conn, []byte(fmt.Sprintf("the time is %s", time.Now()))); werr != nil {
					log.Errorf("write timeout error: %v", werr)
					return
				}
			}
		}
	}
}
