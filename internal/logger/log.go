package logger

import (
	"fmt"
	"io"
	"os"
)

type Level int

const (
	NoneLevel Level = iota
	WarningLevel
	InfoLevel
	DebugLevel
)

type Logger struct {
	w     io.Writer
	level Level
}

func New(w io.Writer, l Level) *Logger {
	return &Logger{
		level: l,
		w:     w,
	}
}

func (l *Logger) Writer() io.Writer {
	return l.w
}

func (l *Logger) SetWriter(w io.Writer) {
	l.w = w
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) Level() Level {
	return l.level
}

func (l *Logger) IsDebugLevel() bool {
	return l.level >= DebugLevel
}

func (l *Logger) IsInfoLevel() bool {
	return l.level >= InfoLevel
}

func (l *Logger) IsWarningLevel() bool {
	return l.level >= WarningLevel
}

func (l *Logger) Debug(a ...any) (int, error) {
	if !l.IsDebugLevel() {
		return 0, nil
	}
	return fmt.Fprint(l.w, a...)
}

func (l *Logger) Debugln(a ...any) (int, error) {
	if !l.IsDebugLevel() {
		return 0, nil
	}
	return fmt.Fprintln(l.w, a...)
}

func (l *Logger) Debugf(format string, a ...any) (int, error) {
	if !l.IsDebugLevel() {
		return 0, nil
	}
	return fmt.Fprintf(l.w, format, a...)
}

func (l *Logger) Warning(a ...any) (int, error) {
	if !l.IsWarningLevel() {
		return 0, nil
	}
	return fmt.Fprint(l.w, a...)
}

func (l *Logger) Warningln(a ...any) (int, error) {
	if !l.IsWarningLevel() {
		return 0, nil
	}
	return fmt.Fprintln(l.w, a...)
}

func (l *Logger) Warningf(format string, a ...any) (int, error) {
	if !l.IsWarningLevel() {
		return 0, nil
	}
	return fmt.Fprintf(l.w, format, a...)
}

func (l *Logger) Info(a ...any) (int, error) {
	if !l.IsInfoLevel() {
		return 0, nil
	}
	return fmt.Fprint(l.w, a...)
}

func (l *Logger) Infoln(a ...any) (int, error) {
	if !l.IsInfoLevel() {
		return 0, nil
	}
	return fmt.Fprintln(l.w, a...)
}

func (l *Logger) Infof(format string, a ...any) (int, error) {
	if !l.IsInfoLevel() {
		return 0, nil
	}
	return fmt.Fprintf(l.w, format, a...)
}

var std = New(os.Stderr, InfoLevel)

func Writer() io.Writer {
	return std.Writer()
}

func SetWriter(w io.Writer) {
	std.SetWriter(w)
}

func SetLevel(level Level) {
	std.SetLevel(level)
}

func IsDebugLevel() bool {
	return std.IsDebugLevel()
}

func IsInfoLevel() bool {
	return std.IsInfoLevel()
}

func IsWarningLevel() bool {
	return std.IsWarningLevel()
}

func Default() *Logger {
	return std
}

func Debug(a ...any) {
	_, _ = std.Debug(a...)
}

func Debugln(a ...any) {
	_, _ = std.Debugln(a...)
}

func Debugf(format string, a ...any) {
	_, _ = std.Debugf(format, a...)
}

func Warning(a ...any) {
	_, _ = std.Warning(a...)
}

func Warningln(a ...any) {
	_, _ = std.Warningln(a...)
}

func Warningf(format string, a ...any) {
	_, _ = std.Warningf(format, a...)
}

func Info(a ...any) {
	_, _ = std.Info(a...)
}

func Infoln(a ...any) {
	_, _ = std.Infoln(a...)
}

func Infof(format string, a ...any) {
	_, _ = std.Infof(format, a...)
}
