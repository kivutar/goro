package glog

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	charm "github.com/charmbracelet/log"
)

var logger = charm.Default()

type LogConfig struct {
	Level string
	File  string
}

type fileLogWriter struct {
	file   io.Writer
	stderr io.Writer
}

func (w fileLogWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	// A console-free Windows launch may have no valid stderr handle. Its
	// failure must not make the logger retry bytes already written to the file.
	_, _ = w.stderr.Write(p)
	return n, err
}

func Configure(cfg LogConfig) (func() error, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	output := io.Writer(os.Stderr)
	var file *os.File
	if cfg.File != "" {
		if dir := filepath.Dir(cfg.File); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
		}
		file, err = os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, err
		}
		output = fileLogWriter{file: file, stderr: os.Stderr}
	}

	logger = charm.NewWithOptions(output, charm.Options{
		Level:           level,
		ReportTimestamp: true,
		TimeFormat:      charm.DefaultTimeFormat,
	})
	charm.SetDefault(logger)
	slog.SetDefault(slog.New(logger))

	return func() error {
		if file != nil {
			return file.Close()
		}
		return nil
	}, nil
}

func Debugf(format string, args ...any) {
	logger.Debugf(format, args...)
}

func Infof(format string, args ...any) {
	logger.Infof(format, args...)
}

func Warnf(format string, args ...any) {
	logger.Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	logger.Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	logger.Fatalf(format, args...)
}

func parseLevel(level string) (charm.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		return charm.InfoLevel, nil
	case "debug":
		return charm.DebugLevel, nil
	case "warn", "warning":
		return charm.WarnLevel, nil
	case "error":
		return charm.ErrorLevel, nil
	case "fatal":
		return charm.FatalLevel, nil
	default:
		return charm.InfoLevel, fmt.Errorf("invalid log level %q", level)
	}
}
