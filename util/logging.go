package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type LogLevel uint8

const (
	PanicLevel LogLevel = iota
	FatalLevel
	ErrorLevel
	WarningLevel
	InfoLevel
	DebugLevel
)

type Logger struct {
	Out    io.Writer
	Level  LogLevel
	Prefix string
}

func NewLogger(level LogLevel, out io.Writer, prefix string) *Logger {
	return &Logger{
		Out:    out,
		Level:  level,
		Prefix: prefix,
	}
}

var logger *Logger = NewLogger(DebugLevel, os.Stdout, "DeepHealth:")

func SetLogLevel(level LogLevel) {
	logger.Level = level
}

func LogD(tag string, format string, a ...interface{}) {
	if logger.Level >= DebugLevel {
		_, fn, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s[DEBUG] %s:%d: ", time.Now().Format(time.RFC3339), filepath.Base(fn), line)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogI(tag string, format string, a ...interface{}) {
	if logger.Level >= InfoLevel {
		_, fn, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s[INFO]  %s:%d: ", time.Now().Format(time.RFC3339), filepath.Base(fn), line)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogE(tag string, format string, a ...interface{}) {
	if logger.Level >= ErrorLevel {
		_, fn, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s[ERROR] %s:%d: ", time.Now().Format(time.RFC3339), filepath.Base(fn), line)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogF(tag string, format string, a ...interface{}) {
	if logger.Level >= FatalLevel {
		_, fn, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s[FATAL] %s:%d: ", time.Now().Format(time.RFC3339), filepath.Base(fn), line)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
		os.Exit(1)
	}
}

func LogP(tag string, format string, a ...interface{}) {
	if logger.Level >= PanicLevel {
		_, fn, line, _ := runtime.Caller(1)
		prefix := fmt.Sprintf("%s[PANIC] %s:%d: ", time.Now().Format(time.RFC3339), filepath.Base(fn), line)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
		panic(fmt.Sprintf(format, a...))
	}
}
