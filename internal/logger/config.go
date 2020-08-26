package logger

import (
	"io"
	"os"

	"github.com/fatih/color"
)

// Config for the Logger interface
type Config struct {
	// Prefix is a string that gets added to the start of the print message
	Prefix string
	// PAttr is the prefix attributes
	PAttr *color.Color
	// Suffix is a string that gets added after the logtype.
	// Suffix has a max length of 20
	Suffix string
	// SAttr is the suffix attributes
	SAttr *color.Color
	// Layout is a string that specifies the time format
	Layout string
	// LogFormat is a map that contains the name for each logtype
	LogFormat map[logtype]string
	// LogColor is a map that contains all the colors for each logtype
	LogColor map[logtype]color.Attribute
	// Debug is a boolean value that determines if debug messages get printed
	Debug bool
	// Writer is the standard io.Writer
	Writer io.Writer
	// PWidth is the minimum prefix width
	PWidth int
	// SWidth is the minimum suffix width
	SWidth int
}

var DefaultConfig = Config{
	Prefix: "",
	PAttr:  color.New(),
	Suffix: "",
	SAttr:  color.New(),
	Layout: "15:04:05.000",
	LogFormat: map[logtype]string{
		debug:  "DEBUG",
		info:   "INFO",
		warn:   "WARNING",
		danger: "DANGER",
		fatal:  "FATAL",
	},
	LogColor: map[logtype]color.Attribute{
		debug:  color.FgMagenta,
		info:   color.FgBlue,
		warn:   color.FgYellow,
		danger: color.FgRed,
		fatal:  color.FgHiRed,
	},
	Writer: os.Stdout,
	PWidth: 8,
	SWidth: 20,
}
