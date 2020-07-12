package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/oauth2"
)

const apiurl = "https://discordapp.com/api/v6"
const meurl = apiurl + "/users/@me"

// Endpoint is the oauth2 discord endpoint
var Endpoint = oauth2.Endpoint{
	AuthURL:  apiurl + "/oauth2/authorize",
	TokenURL: apiurl + "/oauth2/token",
}

// ErrClientNil is returned oauth2.Client == nil
var ErrClientNil = errors.New("oauth2.Client is nil")

// ErrStatusCode is returned whenever StatusCode != 200
var ErrStatusCode = errors.New("http.StatusCode is not 200")

// GetUser returns a user from TokenSource
func GetUser(t oauth2.TokenSource) (User, error) {
	var u User

	client := oauth2.NewClient(oauth2.NoContext, t)
	if client == nil {
		return u, ErrClientNil
	}

	resp, err := client.Get(meurl)
	if err != nil {
		return u, fmt.Errorf("client.Get: %w", err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return u, fmt.Errorf("ioutil.ReadAll: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return u, fmt.Errorf("%w\n body: %s\nstatus code: %d", ErrStatusCode, string(bytes), resp.StatusCode)
	}

	var unmarshal user
	err = json.Unmarshal(bytes, &unmarshal)
	if err != nil {
		return u, fmt.Errorf("json.Unmarshal: %w", err)
	}

	return User{
		ID:            unmarshal.ID,
		email:         unmarshal.Email,
		Username:      unmarshal.Username,
		Avatar:        unmarshal.Avatar,
		Discriminator: unmarshal.Discriminator,
	}, nil
}
