package routes

import (
	"encoding/gob"
	"errors"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"github.com/toms1441/resistance-server/internal/discord"
	"golang.org/x/oauth2"
)

// SessionUser is a struct that caches discord user information, it helps us not get rate limited by discord
type SessionUser struct {
	Expiry      time.Time
	DiscordUser discord.User
}

// Valid returns the validity of the user. i.e if the user has expired
func (s SessionUser) Valid() bool {
	if time.Now().After(s.Expiry) {
		return false
	}

	return true
}

// Routes is an interface that has 3 routes: redirect, login and logout.
type DiscordRoutes interface {
	// RefreshUser refreshes the current user whether it's has expired or not
	RefreshUser(c *gin.Context)
	// GetUser returns the discord user
	GetUser(c *gin.Context) (discord.User, error)
	// RefreshMiddleware refreshes the token whenever it's invalid
	RefreshMiddleware() gin.HandlerFunc
	// Redirect to discord page that authorizes user credenitals
	Redirect() gin.HandlerFunc
	// The route that gets redirected from discord authorization page
	Login() gin.HandlerFunc
	// Deletes the token from the session
	Logout() gin.HandlerFunc
}

type discordRoute struct {
	config oauth2.Config
	expiry time.Duration
}

func NewDiscordRoutes(config oauth2.Config, expiry time.Duration) DiscordRoutes {
	gob.Register(SessionUser{})
	gob.Register(&oauth2.Token{})
	gob.Register(time.Time{})

	config.Endpoint = discord.Endpoint

	dr := discordRoute{
		config: config,
		expiry: expiry,
	}

	return dr
}

func (r discordRoute) RefreshUser(c *gin.Context) {
	sesh := sessions.Default(c)
	if sesh != nil {
		token, ok := sesh.Get("token").(*oauth2.Token)
		if ok {
			discorduser, err := discord.GetUser(r.config.TokenSource(oauth2.NoContext, token))
			if err == nil {

				user := SessionUser{
					Expiry:      time.Now().Add(r.expiry),
					DiscordUser: discorduser,
				}

				sesh.Set("user", user)
				sesh.Save()

			} else {
				sesh.Delete("token")
				sesh.Delete("user")

				sesh.Save()
			}

			c.Set("discordusererr", err)
			c.Set("discorduser", discorduser)
		}
	}

}

func (r discordRoute) GetUser(c *gin.Context) (discord.User, error) {
	sesh := sessions.Default(c)
	if sesh != nil {
		token, ok := sesh.Get("token").(*oauth2.Token)
		if ok && token != nil {
			user, ok := sesh.Get("user").(SessionUser)
			if ok {
				if user.Valid() {
					return user.DiscordUser, nil
				}
			}

			r.RefreshUser(c)

			val1, exists1 := c.Get("discorduser")
			val2, exists2 := c.Get("discordusererr")
			if exists1 && exists2 {
				discorduser, _ := val1.(discord.User)
				discordusererr, _ := val2.(error)

				return discorduser, discordusererr
			}
		}
	}

	return discord.User{}, errors.New("User is not logged in")
}

func (r discordRoute) RefreshMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sesh := sessions.Default(c)
		if sesh != nil {
			token, ok := sesh.Get("token").(*oauth2.Token)
			if ok && token != nil {
				user, err := discord.GetUser(r.config.TokenSource(oauth2.NoContext, token))
				// delete the token if we cant get the user
				if user == (discord.User{}) || err != nil {
					sesh.Delete("token")
					sesh.Save()
				} else {
					c.Set("loggedin", true)
				}
			}
		}
	}
}

func (r discordRoute) Redirect() gin.HandlerFunc {
	return func(c *gin.Context) {
		sesh := sessions.Default(c)
		if sesh != nil {
			state := randstr.Hex(16)
			sesh.Set("state", state)
			if sesh.Save() == nil {
				c.Redirect(307, r.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "code")))
			}
		}
	}
}

func (r discordRoute) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		sesh := sessions.Default(c)
		if sesh != nil {
			_, ok := sesh.Get("token").(*oauth2.Token)
			if !ok {
				state, ok := sesh.Get("state").(string)
				if ok {
					if state == c.Query("state") {
						code := c.Query("code")
						if len(code) > 0 {
							token, err := r.config.Exchange(oauth2.NoContext, code)
							if err == nil {
								sesh.Set("token", token)
							} else {
								c.AbortWithStatus(http.StatusBadRequest)
							}

							sesh.Delete("state")
							sesh.Save()
						}
					}
				}
			} else {
				c.AbortWithStatus(http.StatusForbidden)
			}
		}
	}
}

func (r discordRoute) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		sesh := sessions.Default(c)
		if sesh != nil {
			sesh.Delete("token")
			sesh.Save()
		}
	}
}
