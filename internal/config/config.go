package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator"
	"github.com/spf13/viper"
	"github.com/toms1441/resistance-server/internal/discord"
	"github.com/toms1441/resistance-server/internal/lobby"
)

var validate *validator.Validate

type Config struct {
	Domain        string         `validate:"required" mapstructure:"domain"`
	SessionSecret string         `validate:"required,len=32" mapstructure:"session_secret"`
	Discord       discord.Config `validate:"required" mapstructure:"discord"`
	Lobby         lobby.Config   `validate:"required" mapstructure:"lobby"`
}

func NewConfig() (c Config, err error) {

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.SetDefault("domain", "http://localhost:8080")
	viper.SetDefault("discord", discord.DefaultConfig)
	viper.SetDefault("lobby", lobby.DefaultConfig)

	// Initiate viper for our config
	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			os.Create("config.yaml")
			viper.WriteConfig()
		} else {
			return c, fmt.Errorf("viper.ReadInConfig: %w", err)
		}
	}

	// Unmarshal the config
	err = viper.Unmarshal(&c)
	if err != nil {
		return c, fmt.Errorf("viper.Unmarshal: %w", err)
	}

	err = c.Validate()
	if err != nil {
		return c, fmt.Errorf("c.Validate: %w", err)
	}

	viper.WriteConfig()
	return c, nil
}

func (c Config) Validate() error {
	if validate == nil {
		validate = validator.New()
	}

	return validate.Struct(c)
}
