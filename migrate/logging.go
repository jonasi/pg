package migrate

import (
	"fmt"
	"github.com/fatih/color"
)

func toFmt(msg string, c []interface{}) string {
	fmt := msg

	for i := 0; i < len(c)/2; i++ {
		fmt += " [%s=%v]"
	}

	return fmt
}

type pgxLogger struct {
	l *logger
}

func (l *pgxLogger) Debug(msg string, context ...interface{}) {
	l.l.Debugf(toFmt(msg, context), context...)
}
func (l *pgxLogger) Info(msg string, context ...interface{}) {
	l.l.Debugf(toFmt(msg, context), context...)
}
func (l *pgxLogger) Warn(msg string, context ...interface{}) {}
func (l *pgxLogger) Error(msg string, context ...interface{}) {
	l.l.Errorf(toFmt(msg, context), context...)
}

type logger struct {
	debug bool
}

func (l *logger) Debugf(fmt string, params ...interface{}) {
	if l.debug {
		color.Blue("[DEBUG] "+fmt, params...)
	}
}

func (l *logger) Infof(f string, params ...interface{}) {
	fmt.Printf("[INFO] "+f+"\n", params...)
}

func (l *logger) Errorf(fmt string, params ...interface{}) {
	color.Red("[ERROR] "+fmt, params...)
}
