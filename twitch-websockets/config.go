package main

import (
	"encoding/json"
	"errors"
	"os"
)

func LoadConfig(filename string) (Config, error) {
	var config Config
	if filename == "" {
		return config, errors.New("must specify a config file path")
	} else if b, err := os.ReadFile(filename); err != nil {
		return config, err
	} else if err = json.Unmarshal(b, &config); err != nil {
		return config, err
	}
	return config, nil
}

type Config struct {
	AccessToken  string `json:"accessToken"`
	ChannelID    string `json:"channelId"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	CallbackURL  string `json:"callbackUrl"`
}

func (c Config) Validate() error {
	if c.ChannelID == "" {
		return errors.New("channel ID required")
	}

	if c.AccessToken != "" {
		return nil
	}

	if c.ClientID == "" {
		return errors.New("client ID must be specified if access token is not")
	}

	if c.ClientSecret == "" {
		return errors.New("client secret must be specified if access token is not")
	}

	if c.CallbackURL == "" {
		return errors.New("callback url must be specified if access token is not")
	}

	return nil
}
