package logger

import (
	"strings"
	"testing"
)

var instance Logger

// tests need to be written
func TestNewLogger(t *testing.T) {
	instance = NewLogger(DefaultConfig)
	for i := 0; i <= 25; i++ {
		instance.SetSuffix(strings.Repeat("A", i))
		instance.Info("awddwa")
	}
}
