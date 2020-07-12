package lobby

import (
	"errors"
)

type Config struct {
	IDLen     int `validate:"required"`
	MaxClient int `validate:"required"`
	// this field is composed of idlen
	max int
	// this field is also composed of idlen
	min int
}

var DefaultConfig = Config{
	IDLen:     4,
	MaxClient: 10,
}

var gConfig = DefaultConfig

func (c Config) Validate() error {
	if c.MaxClient == 0 {
		return errors.New("MaxClient is 0")
	}

	if c.IDLen == 0 {
		return errors.New("IDLen is 0")
	}

	return nil
}
