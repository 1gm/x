package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

type OAuthHandler struct {
	log        *zap.SugaredLogger
	oauthState string
	// accessToken is written to this channel when it's been acquired from Twitch
	accessTokenCh chan<- string

	ClientID     string
	ClientSecret string
	CallbackURL  string
}

func (h *OAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.handleOAuthTwitch(w, r)
	} else if r.URL.Path == "/oauth/twitch/callback" {
		h.handleOAuthTwitchCallback(w, r)
	} else {
		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("path %v not found", r.URL.Path)))
	}
}

func (h *OAuthHandler) handleOAuthTwitch(w http.ResponseWriter, r *http.Request) {
	// Generate new OAuth state for the session to prevent CSRF attacks.
	var err error
	h.oauthState, err = h.makeOAuthState()
	if err != nil {
		h.log.Error(err)
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	var oauthClaims = oauth2.SetAuthURLParam("claims", `{"id_token":{"email":null,"email_verified":null,"preferred_username":null},"userinfo":{"picture":null}}`)
	authCodeURL := h.makeOAuth2Config().AuthCodeURL(h.oauthState, oauthClaims)
	h.log.Debugf("twitch auth code url %q", authCodeURL)
	http.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) handleOAuthTwitchCallback(w http.ResponseWriter, r *http.Request) {

}

func (h *OAuthHandler) makeOAuthState() (string, error) {
	state := make([]byte, 64)
	if _, err := io.ReadFull(rand.Reader, state); err != nil {
		return "", fmt.Errorf("makeOAuthState: %v", err)
	}
	return hex.EncodeToString(state), nil
}

func (h *OAuthHandler) makeOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     h.ClientID,
		ClientSecret: h.ClientSecret,
		RedirectURL:  h.CallbackURL,
		Scopes: []string{
			"openid",
			"user:read:email",
			"channel:manage:redemptions", // create custom rewards
			"channel:read:redemptions",   // read custom rewards & redemptions
		},
		Endpoint: twitch.Endpoint,
	}
}
