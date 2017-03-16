package util

import (
	"fmt"
	"io"
	"log"
	"os"
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
	Out   io.Writer
	Level LogLevel
	l     *log.Logger
}

func NewLogger(level LogLevel, out io.Writer, prefix string) *Logger {
	return &Logger{
		Out:   out,
		Level: level,
		l:     log.New(out, prefix, log.Lshortfile),
	}
}

var logger *Logger = NewLogger(DebugLevel, os.Stdout, "DeepHealth:")

func LogD(tag string, format string, a ...interface{}) {
	if logger.Level >= DebugLevel {
		prefix := fmt.Sprintf("DEBUG[%s] ", tag)
		logger.l.Printf(prefix+format+"\n", a...)
	}
}

func LogI(tag string, format string, a ...interface{}) {
	if logger.Level >= InfoLevel {
		prefix := fmt.Sprintf("INFO[%s] ", tag)
		logger.l.Printf(prefix+format+"\n", a...)
	}
}

func LogE(tag string, format string, a ...interface{}) {
	if logger.Level >= ErrorLevel {
		prefix := fmt.Sprintf("ERROR[%s] ", tag)
		logger.l.Printf(prefix+format+"\n", a...)
	}
}

func LogF(tag string, format string, a ...interface{}) {
	if logger.Level >= FatalLevel {
		prefix := fmt.Sprintf("FATAL[%s] ", tag)
		logger.l.Fatalf(prefix+format+"\n", a...)
	}
}

func LogP(tag string, format string, a ...interface{}) {
	if logger.Level >= FatalLevel {
		prefix := fmt.Sprintf("PANIC[%s] ", tag)
		logger.l.Panicf(prefix+format+"\n", a...)
	}
}
