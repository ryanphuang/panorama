package util

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

var logger *Logger = NewLogger(InfoLevel, os.Stdout, "DeepHealth:")

const (
	LogTimeFormat = "2006-01-02 15:04:05.000000"
)

func SetLogLevel(level LogLevel) {
	logger.Level = level
}

func SetLogLevelString(level string) {
	level = strings.ToLower(level)
	switch level {
	case "debug":
		logger.Level = DebugLevel
	case "info":
		logger.Level = InfoLevel
	case "warn":
		logger.Level = WarningLevel
	case "error":
		logger.Level = ErrorLevel
	case "fatal":
		logger.Level = FatalLevel
	case "panic":
		logger.Level = PanicLevel
	}
}

func LogD(tag string, format string, a ...interface{}) {
	if logger.Level >= DebugLevel {
		pc, fn, _, _ := runtime.Caller(1)
		srcName := filepath.Base(fn)
		funcName := runtime.FuncForPC(pc).Name()
		prefix := fmt.Sprintf("%s DEBUG  [%s] %s:%s: ", time.Now().Format(LogTimeFormat), tag, srcName, funcName)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogI(tag string, format string, a ...interface{}) {
	if logger.Level >= InfoLevel {
		pc, fn, _, _ := runtime.Caller(1)
		srcName := filepath.Base(fn)
		funcName := runtime.FuncForPC(pc).Name()
		prefix := fmt.Sprintf("%s INFO  [%s]  %s:%s: ", time.Now().Format(LogTimeFormat), tag, srcName, funcName)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogE(tag string, format string, a ...interface{}) {
	if logger.Level >= ErrorLevel {
		pc, fn, _, _ := runtime.Caller(1)
		srcName := filepath.Base(fn)
		funcName := runtime.FuncForPC(pc).Name()
		prefix := fmt.Sprintf("%s ERROR  [%s] %s:%s: ", time.Now().Format(LogTimeFormat), tag, srcName, funcName)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
	}
}

func LogF(tag string, format string, a ...interface{}) {
	if logger.Level >= FatalLevel {
		pc, fn, _, _ := runtime.Caller(1)
		srcName := filepath.Base(fn)
		funcName := runtime.FuncForPC(pc).Name()
		prefix := fmt.Sprintf("%s FATAL  [%s] %s:%s: ", time.Now().Format(LogTimeFormat), tag, srcName, funcName)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
		os.Exit(1)
	}
}

func LogP(tag string, format string, a ...interface{}) {
	if logger.Level >= PanicLevel {
		pc, fn, _, _ := runtime.Caller(1)
		srcName := filepath.Base(fn)
		funcName := runtime.FuncForPC(pc).Name()
		prefix := fmt.Sprintf("%s PANIC  [%s] %s:%s: ", time.Now().Format(LogTimeFormat), tag, srcName, funcName)
		fmt.Fprintf(logger.Out, prefix+format+"\n", a...)
		panic(fmt.Sprintf(format, a...))
	}
}
