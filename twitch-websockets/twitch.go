package main

import (
	"encoding/json"
	"strconv"
	"time"
)

type Event struct {
	Type  string      `json:"type"`
	Data  []byte      `json:"data"`
	Value interface{} `json:"-"`
}

type twitchPubSubMessage struct {
	Type  string          `json:"type,omitempty"`
	Error string          `json:"error,omitempty"`
	Nonce string          `json:"nonce,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

func (ps twitchPubSubMessage) Unwrap() (*Event, error) {
	var outer struct {
		Topic   string          `json:"topic"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(ps.Data, &outer); err != nil {
		return nil, err
	}

	msg, err := strconv.Unquote(string(outer.Message))
	if err != nil {
		return nil, err
	}

	var inner struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal([]byte(msg), &inner); err != nil {
		return nil, err
	}

	t := Event{
		Type:  inner.Type,
		Data:  inner.Data,
		Value: string(inner.Data),
	}

	// can filter on different types of events here but let's just keep it simple for this example.
	if t.Type == "reward-redeemed" {
		var redeem twitchRewardRedeemedEvent
		if err = json.Unmarshal(t.Data, &redeem); err != nil {
			return nil, err
		}
		t.Value = redeem
	}

	return &t, nil
}

type twitchRewardRedeemedEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Redemption struct {
		ID   string `json:"id"`
		User struct {
			ID          string `json:"id"`
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
		} `json:"user"`
		ChannelID  string    `json:"channel_id"`
		RedeemedAt time.Time `json:"redeemed_at"`
		Reward     struct {
			ID                  string `json:"id"`
			ChannelID           string `json:"channel_id"`
			Title               string `json:"title"`
			Prompt              string `json:"prompt"`
			Cost                int    `json:"cost"`
			IsUserInputRequired bool   `json:"is_user_input_required"`
			IsSubOnly           bool   `json:"is_sub_only"`
			Image               struct {
				URL1X string `json:"url_1x"`
				URL2X string `json:"url_2x"`
				URL4X string `json:"url_4x"`
			} `json:"image"`
			DefaultImage struct {
				URL1X string `json:"url_1x"`
				URL2X string `json:"url_2x"`
				URL4X string `json:"url_4x"`
			} `json:"default_image"`
			BackgroundColor string `json:"background_color"`
			IsEnabled       bool   `json:"is_enabled"`
			IsPaused        bool   `json:"is_paused"`
			IsInStock       bool   `json:"is_in_stock"`
			MaxPerStream    struct {
				IsEnabled    bool `json:"is_enabled"`
				MaxPerStream int  `json:"max_per_stream"`
			} `json:"max_per_stream"`
			ShouldRedemptionsSkipRequestQueue bool        `json:"should_redemptions_skip_request_queue"`
			TemplateID                        interface{} `json:"template_id"`
			UpdatedForIndicatorAt             time.Time   `json:"updated_for_indicator_at"`
		} `json:"reward"`
		Status    string `json:"status"`
		UserInput string `json:"user_input"`
	} `json:"redemption"`
}
