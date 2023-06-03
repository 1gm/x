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

	accessTokenCh := make(chan string)
	// if access token is not specified in the config then we need to fetch it from Twitch.
	var server *http.Server
	if config.AccessToken == "" {
		oauthHandler := &OAuthHandler{
			log:           log,
			accessTokenCh: accessTokenCh,
			ClientID:      config.ClientID,
			ClientSecret:  config.ClientSecret,
			CallbackURL:   config.CallbackURL,
		}
		server = &http.Server{Handler: oauthHandler, Addr: ":8080"}
		go func() {
			log.Info("go to http://localhost:8080 to get an access token")
			if err := server.ListenAndServe(); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					log.Errorf("server closed unexpectedly: %v", err)
				}
			}
		}()
	} else {
		// send the access token immediately
		accessTokenCh <- config.AccessToken
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() { <-c; cancel() }()

	var exitCode int
	var closeConn = func() {}
LOOP:
	for {
		select {
		// if we are receiving the access token from twitch then we will be blocking here
		case accessToken, ok := <-accessTokenCh:
			log.Info("received access token")
			if ok {
				if server != nil {
					if err = server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						log.Errorf("server failed to close: %v", err)
						exitCode = 1
						break LOOP
					}
				}

				// use a background context to allow us to issue UNLISTEN commands to the twitch API
				if closeConn, err = openWebsocketConnection(context.Background(), log, accessToken, config.ChannelID); err != nil {
					err = err
					exitCode = 1
					break LOOP
				}
			}
		case <-ctx.Done():
			if err = ctx.Err(); !errors.Is(err, context.Canceled) {
				exitCode = 1
				break LOOP
			} else {
				err = nil
			}
			exitCode = 0
			break LOOP
		}
	}

	closeConn()
	if err != nil {
		log.Error(err)
	}
	return exitCode
}
