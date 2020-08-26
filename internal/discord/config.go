package discord

type Config struct {
	ClientID     string `validate:"required,len=18" mapstructure:"client_id"`     // Client ID for discord
	ClientSecret string `validate:"required,len=32" mapstructure:"client_secret"` // Client Secret for discord
}

var DefaultConfig = Config{}
