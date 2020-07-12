package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/fatih/color"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/toms1441/resistance-server/internal/client"
	"github.com/toms1441/resistance-server/internal/config"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/lobby"
	"github.com/toms1441/resistance-server/internal/logger"
	"github.com/toms1441/resistance-server/internal/repo/plain"
	"github.com/toms1441/resistance-server/internal/routes"
	"golang.org/x/oauth2"

	_ "github.com/toms1441/resistance-server/internal/game"
)

func main() {
	lc := logger.DefaultConfig
	lc.PAttr = color.New(color.Italic, color.FgHiGreen)
	lc.Prefix = "main"

	main := logger.NewLogger(lc)

	c, err := config.NewConfig()
	if err != nil {
		main.Fatal("config.NewConfig: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	store := cookie.NewStore([]byte(c.SessionSecret))
	sesh := sessions.Sessions("sesh", store)

	r.Use(sesh)

	getuser := func(c *gin.Context) (discord.User, error) {
		return discord.User{}, nil
	}

	// auth-related routes
	{
		group := "/auth/discord"

		dr := routes.NewDiscordRoutes(oauth2.Config{
			ClientID:     c.Discord.ClientID,
			ClientSecret: c.Discord.ClientSecret,
			RedirectURL:  fmt.Sprintf("%s%s/login", c.Domain, group),
			Scopes:       []string{"email", "identify"},
		}, time.Hour*12)
		// cache for 12 hours

		// refresh the token whenever it's invalid
		r.Use(dr.RefreshMiddleware())

		// kinda globalize getuser to use in client
		getuser = dr.GetUser

		routergroup := r.Group(group)

		routergroup.GET("/redirect", dr.Redirect())
		routergroup.GET("/login", dr.Login())
		routergroup.GET("/logout", dr.Logout())
		routergroup.GET("/verify", func(c *gin.Context) {
			user, _ := dr.GetUser(c)

			c.JSON(200, user)
		})

		routergroup.POST("/refresh", func(c *gin.Context) {
			dr.RefreshUser(c)
		})
	}

	// lobby service init
	// make the lobby service kinda global
	// so we can use it in websocket route
	var lserv lobby.Service
	{
		lc = logger.DefaultConfig
		lc.PAttr = color.New(color.FgRed, color.Italic)
		lc.Prefix = "lobby"
		lc.Debug = true

		llog := logger.NewLogger(lc)
		lrepo := plain.NewLobbyRepository()

		var err error
		lserv, err = lobby.NewService(lrepo, llog, c.Lobby)
		if err != nil {
			main.Danger("an error occurred with creating the lobby service: %v", err)
		}
	}

	// client service init
	var cserv client.Service
	{
		lc = logger.DefaultConfig

		lc.PAttr = color.New(color.FgHiBlue, color.Italic)
		lc.Prefix = "client"
		lc.Debug = true

		clog := logger.NewLogger(lc)
		crepo := plain.NewClientRepository()

		var err error
		cserv, err = client.NewService(crepo, clog)
		if err != nil {
			main.Danger("an error occurred with creating the client service: %v", err)
		}
	}

	// websocket route init
	{
		lc := logger.DefaultConfig
		lc.PAttr = color.New(color.FgCyan, color.Italic)
		lc.Prefix = "socket"
		lc.Debug = true

		r.GET("/ws", routes.NewWebsocketRoute(
			routes.WebsocketConfig{
				LobbyService:  lserv,
				ClientService: cserv,
				Log:           logger.NewLogger(lc),
				GetUser:       getuser,
			},
		))
	}

	var port = "8080"

	uri, err := url.Parse(c.Domain)
	if err == nil {
		port = uri.Port()
	}

	main.Info("Running web server on %s. Press Ctrl-C to quit.", port)
	r.Run(":" + port)

}

/* send post request to refresh user information
	r.GET("/", func(c *gin.Context) {
		c.Data(200, "text/html; charset=utf-8", []byte(`
<form method="post" action="/auth/discord/refresh">
	<input type="submit" value="alright">
</form>`))
	})
*/
