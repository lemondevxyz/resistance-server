package lobby

import (
	"testing"

	"github.com/toms1441/resistance-server/internal/lobby"
)

var config = lobby.Config{}
var invalidconfig = lobby.Config{}

func TestConfigValidate(t *testing.T) {
	if err := config.Validate(); err == nil {
		t.Fatalf("Invalid config returns no error. config.IDLen == 0, config.MaxClient == 0")
	}

	config.MaxClient = 1
	if err := config.Validate(); err == nil {
		t.Fatalf("Invalid config returns no error. config.IDLen == 0")
	}

	config.MaxClient = 0
	config.IDLen = 1
	if err := config.Validate(); err == nil {
		t.Fatalf("Invalid config returns no error. config.MaxClient == 0")
	}

	config.MaxClient = 1
	config.IDLen = 2
	if err := config.Validate(); err != nil {
		t.Fatalf("config.Validate: %v", err)
	}
}
