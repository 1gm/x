package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"

	"github.com/1gm/x/internal/log"
)

func main() {
	accessToken := flag.String("a", "", "twitch access token")
	channelID := flag.String("c", "", "twitch channel ID")
	flag.Parse()

	os.Exit(realMain(*accessToken, *channelID))
}

func realMain(accessToken string, channelID string) int {
	log := log.New()
	defer log.Sync()

	if accessToken == "" {
		log.Error("access token is required (specified with -a flag)")
		return 1
	}

	if channelID == "" {
		log.Error("channel ID is required (specified with -c flag)")
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	closeConn, err := openWebsocketConnection(ctx, log, accessToken, channelID)
	if err != nil {
		log.Error(err)
		return 1
	}
	defer closeConn()

	for {
		select {
		case <-ctx.Done():
			if err = ctx.Err(); !errors.Is(err, context.Canceled) {
				log.Error(err)
				return 1
			}
			return 0
		}
	}
}
