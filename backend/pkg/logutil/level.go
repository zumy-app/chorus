package logutil

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var currentLevel = LevelInfo

var levelNames = map[Level]string{
	LevelDebug: "DEBUG",
	LevelInfo:  "INFO",
	LevelWarn:  "WARN",
	LevelError: "ERROR",
}

func SetLevelFromString(s string) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		currentLevel = LevelDebug
	case "INFO":
		currentLevel = LevelInfo
	case "WARN", "WARNING":
		currentLevel = LevelWarn
	case "ERROR":
		currentLevel = LevelError
	}
}

func logf(level Level, format string, args ...any) {
	if level < currentLevel {
		return
	}
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s", levelNames[level], msg)
}

func Debugf(format string, args ...any) {
	logf(LevelDebug, format, args...)
}

func Infof(format string, args ...any) {
	logf(LevelInfo, format, args...)
}

func Warnf(format string, args ...any) {
	logf(LevelWarn, format, args...)
}

func Errorf(format string, args ...any) {
	logf(LevelError, format, args...)
}

func Duration(name string, start time.Time, args ...any) {
	elapsed := time.Since(start)
	if len(args) > 0 {
		Infof("[%s] %s took %v", name, fmt.Sprint(args...), elapsed)
	} else {
		Infof("[%s] took %v", name, elapsed)
	}
}
