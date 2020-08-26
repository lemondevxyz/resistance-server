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

var (
	ErrMaxClientZero = errors.New("Maxclient cannot be zero or less")
	ErrIDLengthZero  = errors.New("ID Length cannot be zero or less")
)

func (c Config) Validate() error {
	if c.MaxClient <= 0 {
		return ErrMaxClientZero
	}

	if c.IDLen <= 0 {
		return ErrIDLengthZero
	}

	return nil
}
