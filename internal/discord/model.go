package discord

import (
	"fmt"
	"strconv"
)

// User is the user object we recieve from meurl.
type User struct {
	ID       string `json:"id"`
	email    string
	Username string `json:"username"`
	// Discriminator is the number(#4444) that you get after the username.
	Discriminator string `json:"discriminator"`
	// Avatar is a hash that can be used to get the real avatar, u.GetAvatar() to get the real avatar.
	Avatar string `json:"avatar"`
}

type user struct {
	User
	Email string `json:"email"`
}

// GetEmail returns the email for the user
func (u User) GetEmail() string {
	return u.email
}

// IsValid returns a boolean value if the user is valid or not.
func (u User) IsValid() bool {
	return u != (User{})
}

// GetAvatar returns the avatar link, something like https://cdn.discordapp.com/embed/avatars/2.png
func (u User) GetAvatar() string {
	avatar := "https://cdn.discordapp.com/"
	if len(u.Avatar) > 0 {
		ext := "png"
		if u.Avatar[0] == 'a' {
			ext = "gif"
		}

		avatar += fmt.Sprintf("avatars/%s/%s.%s", u.ID, u.Avatar, ext)
	} else {
		disc, _ := strconv.Atoi(u.Discriminator)
		avatar += fmt.Sprintf("embed/avatars/%d.png", disc%5)
	}

	return avatar
}
