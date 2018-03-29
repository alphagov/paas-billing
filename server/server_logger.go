package server

import (
	"errors"
	"fmt"
	"io"

	"code.cloudfoundry.org/lager"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
)

// Logger wraps a lager.Logger in the echo.Logger interface
type Logger struct {
	lvl    log.Lvl
	lager  lager.Logger
	action string
}

func (l *Logger) Debug(i ...interface{}) {
	l.lager.Debug(l.action, lager.Data{
		"detail": fmt.Sprint(i...),
	})
}

func (l *Logger) Debugf(format string, i ...interface{}) {
	l.lager.Debug(l.action, lager.Data{
		"detail": fmt.Sprintf(format, i...),
	})
}

func (l *Logger) Debugj(j log.JSON) {
	l.lager.Debug(l.action, lager.Data(j))
}

func (l *Logger) Warn(i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprint(i...),
	})
}

func (l *Logger) Warnf(format string, i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprintf(format, i...),
	})
}

func (l *Logger) Warnj(j log.JSON) {
	l.lager.Info(l.action, lager.Data(j))
}

func (l *Logger) Error(i ...interface{}) {
	l.lager.Error(l.action, errors.New(fmt.Sprint(i...)), lager.Data{})
}

func (l *Logger) Errorf(format string, i ...interface{}) {
	l.lager.Error(l.action, fmt.Errorf(format, i...), lager.Data{})
}

func (l *Logger) Errorj(j log.JSON) {
	l.lager.Error(l.action, errors.New("error"), lager.Data(j))
}

func (l *Logger) Fatal(i ...interface{}) {
	l.lager.Fatal(l.action, errors.New(fmt.Sprint(i...)), lager.Data{})
}

func (l *Logger) Fatalf(format string, i ...interface{}) {
	l.lager.Fatal(l.action, fmt.Errorf(format, i...), lager.Data{})
}

func (l *Logger) Fatalj(j log.JSON) {
	l.lager.Fatal(l.action, errors.New("fatal"), lager.Data(j))
}

func (l *Logger) Info(i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprint(i...),
	})
}

func (l *Logger) Infof(format string, i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprintf(format, i...),
	})
}

func (l *Logger) Infoj(j log.JSON) {
	l.lager.Info(l.action, lager.Data(j))
}

func (l *Logger) Print(i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprint(i...),
	})
}

func (l *Logger) Printf(format string, i ...interface{}) {
	l.lager.Info(l.action, lager.Data{
		"detail": fmt.Sprintf(format, i...),
	})
}

func (l *Logger) Printj(j log.JSON) {
	l.lager.Info(l.action, lager.Data(j))
}

func (l *Logger) Panic(i ...interface{}) {
	l.lager.Error(l.action, errors.New("panic"), lager.Data{
		"detail": fmt.Sprint(i...),
	})
	panic(fmt.Sprint(i...))
}

func (l *Logger) Panicf(format string, i ...interface{}) {
	l.lager.Error(l.action, errors.New("panic"), lager.Data{
		"detail": fmt.Sprintf(format, i...),
	})
	panic(fmt.Sprintf(format, i...))
}

func (l *Logger) Panicj(j log.JSON) {
	l.lager.Error(l.action, errors.New("panic"), lager.Data(j))
	panic(fmt.Sprintf("%v", j))
}

func (l *Logger) Level() log.Lvl {
	return l.lvl
}

func (l *Logger) SetLevel(newLvl log.Lvl) {
	l.lvl = newLvl
}

func (l *Logger) Prefix() string {
	panic("notimp")
}

func (l *Logger) SetPrefix(p string) {
	panic("notimp")
}

func (l *Logger) Output() io.Writer {
	panic("notimp")
}

func (l *Logger) SetOutput(w io.Writer) {
	panic("notimp")
}

func NewLogger(logger lager.Logger) echo.Logger {
	return &Logger{
		lager:  logger,
		action: "request",
	}
}
