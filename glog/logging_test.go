package glog

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	charm "github.com/charmbracelet/log"
)

func TestConfigureFileLoggingWithStderr(t *testing.T) {
	for _, available := range []bool{true, false} {
		name := "available"
		if !available {
			name = "unavailable"
		}
		t.Run(name, func(t *testing.T) {
			previousStderr, previousLogger := os.Stderr, logger
			previousCharm, previousSlog := charm.Default(), slog.Default()
			previousOutput, previousFlags := log.Writer(), log.Flags()
			t.Cleanup(func() {
				os.Stderr, logger = previousStderr, previousLogger
				charm.SetDefault(previousCharm)
				slog.SetDefault(previousSlog)
				log.SetOutput(previousOutput)
				log.SetFlags(previousFlags)
			})

			dir := t.TempDir()
			stderrPath := filepath.Join(dir, "stderr.log")
			stderr, err := os.Create(stderrPath)
			if err != nil {
				t.Fatal(err)
			}
			if available {
				t.Cleanup(func() { _ = stderr.Close() })
			} else if err := stderr.Close(); err != nil {
				t.Fatal(err)
			}
			os.Stderr = stderr

			path := filepath.Join(dir, "goro.log")
			const existing = "previous run\n"
			if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
				t.Fatal(err)
			}
			closeLog, err := Configure(LogConfig{Level: "debug", File: path})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := closeLog(); err != nil {
					t.Error(err)
				}
			})

			Debugf("goro debug message")
			slog.Debug("dependency debug message")
			Errorf("goro error message")

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			if !strings.HasPrefix(text, existing) {
				t.Fatalf("previous log contents were lost: %q", text)
			}
			for _, message := range []string{"goro debug message", "dependency debug message", "goro error message"} {
				if strings.Count(text, message) != 1 {
					t.Errorf("expected one %q in log, got %q", message, text)
				}
			}
			if available {
				console, err := os.ReadFile(stderrPath)
				if err != nil {
					t.Fatal(err)
				}
				if string(console) != strings.TrimPrefix(text, existing) {
					t.Errorf("stderr and file logs differ: stderr=%q file=%q", console, text)
				}
			}
		})
	}
}
