package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

func openWebsocketConnection(ctx context.Context, log *zap.SugaredLogger, accessToken string, channelID string) (closeFunc func(), err error) {
	dial := func(ctx context.Context, timeout time.Duration) (*websocket.Conn, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		ws, _, err := websocket.Dial(ctx, "wss://pubsub-edge.twitch.tv", nil)
		if err != nil {
			return nil, fmt.Errorf("dial failed to connect: %v", err)
		}
		return ws, nil
	}

	ws, err := dial(ctx, time.Second*5)
	if err != nil {
		return nil, err
	}
	log.Info("connected to twitch pubsub")

	c := &wsconn{
		ctx:         ctx,
		accessToken: accessToken,
		channelID:   channelID,
		log:         log,
		ws:          ws,
		stop:        make(chan chan struct{}),
	}

	if err = c.authn("LISTEN"); err != nil {
		return nil, err
	}

	go c.read()
	go c.write()

	return func() {
		if err := c.Close(); err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				c.log.Infof("twitch client close error " + err.Error())
			}
		}
	}, nil
}

type wsconn struct {
	accessToken string
	channelID   string

	stop    chan chan struct{}
	badAuth bool
	log     *zap.SugaredLogger

	ws       *websocket.Conn
	closeErr error
	ctx      context.Context
}

func (c *wsconn) Close() error { return c.close(websocket.StatusNormalClosure, "") }

func (c *wsconn) close(code websocket.StatusCode, reason string) error {
	if c.stop != nil {
		writerStop := make(chan struct{})
		// notify the writer we are stopping and issue the UNLISTEN command to Twitch
		c.stop <- writerStop

		// wait to hear back from writer on stopping
		<-writerStop
		c.stop = nil

		return c.ws.Close(code, reason)
	}

	return nil
}

func (c *wsconn) read() {
	for {
		var msg twitchPubSubMessage
		if err := wsjson.Read(c.ctx, c.ws, &msg); err != nil {
			if errors.Is(err, context.Canceled) || websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			c.log.Error("read from socket: " + err.Error())
			return
		}

		if msg.Error != "" {
			c.log.Info(fmt.Sprintf("received an error: %s (nonce=%s)", msg.Error, msg.Nonce))
			if msg.Error == "ERR_BADAUTH" {
				c.badAuth = true
				if err := c.close(websocket.StatusInternalError, "internal server error [BAD AUTH]"); err != nil {
					c.log.Error("websocket close: " + err.Error())
				}
				return
			}
			continue
		}

		c.log.Debug("received ", msg.Type)

		if msg.Type == "MESSAGE" {
			unwrapped, err := msg.Unwrap()
			if err != nil {
				c.log.Info("failed to unwrap message: " + err.Error())
				break
			}
			if unwrapped.Type == "reward-redeemed" {
				if redeem, ok := unwrapped.Value.(twitchRewardRedeemedEvent); ok {
					c.log.Infow("reward redeemed", "redeem", redeem)
				} else {
					c.log.Error("reward-redeemed type message cannot be cast to twitchRewardRedeemedEvent")
				}
			}
		}
	}
}

func (c *wsconn) write() {
	ping := []byte(`{"type":"PING"}`)

	for {
		if err := c.ws.Write(c.ctx, websocket.MessageText, ping); err != nil {
			c.log.Error("PING: " + err.Error())
			return
		}

		select {
		case stopped := <-c.stop:
			c.ctx = c.ws.CloseRead(c.ctx)
			if !c.badAuth {
				if err := wsjson.Write(c.ctx, c.ws, c.authn("UNLISTEN")); err != nil {
					c.log.Error("failed to disconnect from twitch pubsub: ", err.Error())
				} else {
					c.log.Info("successfully disconnected from twitch pubsub")
				}
			}
			stopped <- struct{}{}
			return
		case <-time.After(time.Second * 30):
		}
	}
}

func (c *wsconn) authn(typ string) (err error) {
	data := struct {
		AuthToken string   `json:"auth_token"`
		Topics    []string `json:"topics"`
	}{
		AuthToken: c.accessToken,
		Topics:    []string{"channel-points-channel-v1." + c.channelID},
	}
	msg := twitchPubSubMessage{Type: typ}
	nonce := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	msg.Nonce = hex.EncodeToString(nonce)
	if msg.Data, err = json.Marshal(&data); err != nil {
		return err
	}
	if err = wsjson.Write(c.ctx, c.ws, msg); err != nil {
		return fmt.Errorf("twitch pubsub client failed to auth: %s", err)
	}
	return nil
}
