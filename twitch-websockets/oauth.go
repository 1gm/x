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
	log *zap.SugaredLogger
	// not thread safe oauth state variable, only one rquest is expected to be sent in the usual use of
	// this application so it doesn't matter.
	oauthState string
	// accessToken is written to this channel when it's been acquired from Twitch
	accessTokenCh chan<- string

	// OAuth configuration values
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
	oauthClaims := oauth2.SetAuthURLParam(
		"claims",
		`{"id_token":{"email":null,"email_verified":null,"preferred_username":null},"userinfo":{"picture":null}}`,
	)
	authCodeURL := h.makeOAuth2Config().AuthCodeURL(h.oauthState, oauthClaims)
	h.log.Debugf("twitch auth code url %q", authCodeURL)
	http.Redirect(w, r, authCodeURL, http.StatusTemporaryRedirect)
}

func (h *OAuthHandler) handleOAuthTwitchCallback(w http.ResponseWriter, r *http.Request) {
	code, scope, state := r.FormValue("code"), r.FormValue("scope"), r.FormValue("state")
	h.log.Debugf("handling oauth callback from twitch, code = %s scope = %s, state = %s", code, scope, state)

	if state != h.oauthState {
		h.log.Errorf("state mismatch want %s got %s", h.oauthState, state)
		w.WriteHeader(401)
		w.Write([]byte("session state mismatch"))
		return
	}

	tok, err := h.makeOAuth2Config().Exchange(r.Context(), code)
	if err != nil {
		h.log.Error(err)
		w.WriteHeader(401)
		w.Write([]byte("failed to exchange oauth code for a token"))
		return
	}

	w.Write([]byte("You may now close this browser"))
	h.accessTokenCh <- tok.AccessToken
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
			"user:read:email",          // this doesn't need to be here
			"channel:read:redemptions", // read custom rewards & redemptions
		},
		Endpoint: twitch.Endpoint,
	}
}
