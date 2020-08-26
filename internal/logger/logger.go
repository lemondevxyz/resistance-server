package logger

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/fatih/color"
)

// Logger is an interface for a logging service
type Logger interface {
	// SetPrefix sets the text that gets added before the log message
	SetPrefix(string)
	// GetPrefix returns the text that gets added before the log message
	GetPrefix() string
	// SetSuffix sets the text that gets added after the log message
	SetSuffix(string)
	// GetSuffix returns the text that gets added after the log message
	GetSuffix() string
	// Debug logs information that is useless when not debugging and crucial when debugging. Color: magenta
	Debug(string, ...interface{})
	// Info logs information that is used to indicate success of a function that is important. Color: light blue
	Info(string, ...interface{})
	// Warn logs information that could indicate a future error, or a non-important error. Color: yellow
	Warn(string, ...interface{})
	// Danger logs non-fatal errors in a formatted string.
	Danger(string, ...interface{})
	// Fatal logs fatal-errors and then closes the application.
	Fatal(string, ...interface{})
	// Replicate creates a new logger but with the same config as the current logger
	Replicate() Logger
}

type log struct {
	config       Config
	colors       map[logtype]*color.Color
	logtypewidth int
}

const (
	debug logtype = iota
	info
	warn
	danger
	fatal
)

type logtype uint8

var reset = color.New(color.Reset)

func NewLogger(config Config) Logger {
	colors := map[logtype]*color.Color{}
	for k, v := range config.LogColor {
		colors[k] = color.New(v)
	}

	instance := &log{
		config: config,
		colors: colors,
	}

	instance.SetPrefix(config.Prefix)
	instance.SetSuffix(config.Suffix)

	for _, v := range config.LogFormat {
		if len(v) > instance.logtypewidth {
			instance.logtypewidth = len(v)
		}
	}

	return instance
}

func NullLogger() Logger {
	lc := DefaultConfig
	lc.Writer = ioutil.Discard
	return NewLogger(lc)
}

func (l *log) logtypestr(lt logtype) string {
	str := l.config.LogFormat[lt]
	return l.colors[lt].Sprint(l.calc(str, l.logtypewidth))
}

func (l *log) calc(str string, space int) string {
	size := space - len(str)
	if len(str) > space {
		if space > 0 {
			str = str[:space-4] + "..."
		}
	} else if size >= 0 {
		str = str + strings.Repeat(" ", size)
	}

	return str
}

func (l *log) SetPrefix(str string) {
	l.config.Prefix = l.calc(str, l.config.PWidth)
}

func (l *log) GetPrefix() string {
	return l.config.Prefix
}

func (l *log) SetSuffix(str string) {
	l.config.Suffix = l.calc(str, l.config.SWidth)
}

func (l *log) GetSuffix() string {
	return l.config.Suffix
}

func (l *log) log(typestr logtype, format string, values ...interface{}) {

	str := fmt.Sprintf("%s %s", l.logtypestr(typestr), format)
	if len(l.config.Suffix) > 0 {
		str = fmt.Sprintf("%s %s %s", l.logtypestr(typestr), l.config.SAttr.Sprint(l.config.Suffix), format)
	}

	if len(l.config.Prefix) > 0 {
		str = l.config.PAttr.Sprint(l.config.Prefix) + str
	}

	fmt.Fprintf(l.config.Writer, str+"\n", values...)
}

func (l *log) Debug(format string, values ...interface{}) {
	if l.config.Debug {
		l.log(debug, format, values...)
	}
}

func (l *log) Info(format string, values ...interface{}) {
	l.log(info, format, values...)
}

func (l *log) Warn(format string, values ...interface{}) {
	l.log(warn, format, values...)
}

func (l *log) Danger(format string, values ...interface{}) {
	l.log(danger, format, values...)
}

func (l *log) Fatal(format string, values ...interface{}) {
	l.log(fatal, format, values...)
	os.Exit(1)
}

func (l *log) Replicate() Logger {
	newl := &log{}
	*newl = *l
	return newl
}
