package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/1gm/x/internal/log"
)

func main() {
	configFilePath := flag.String("c", "config.json", "config file path")
	flag.Parse()

	os.Exit(realMain(*configFilePath))
}

func realMain(configFilePath string) int {
	log := log.New()
	defer log.Sync()

	config, err := LoadConfig(configFilePath)
	if err != nil {
		log.Error(err)
		return 1
	} else if err = config.Validate(); err != nil {
		log.Error(err)
		return 1
	}

	accessToken := config.AccessToken
	// if access token is not specified in the config then we need to fetch it from Twitch.
	if accessToken == "" {
		// TODO(george): Implement this.
		accessTokenCh := make(chan string)
		oauthHandler := &OAuthHandler{
			log:           log,
			accessTokenCh: accessTokenCh,
			ClientID:      config.ClientID,
			ClientSecret:  config.ClientSecret,
			CallbackURL:   config.CallbackURL,
		}
		server := http.Server{Handler: oauthHandler, Addr: ":8080"}
		go func() {
			if err := server.ListenAndServe(); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Errorf("server closed unexpectedly: %v", err)
				}
			}
		}()

		// block until we receive the access token
		accessToken = <-accessTokenCh
		if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf("server failed to close: %v", err)
			return 1
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	closeConn, err := openWebsocketConnection(ctx, log, accessToken, config.ChannelID)
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
