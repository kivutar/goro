package config

import (
	"strings"
	"testing"
)

func TestHeadlessCLI(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		keep bool
		want string
	}{
		{"valid", []string{"--username=tester", "--password=test", "--char-slot=2"}, false, ""},
		{"saved ID", []string{"--password=test", "--char-slot=0"}, true, ""},
		{"missing ID", []string{"--password=test", "--char-slot=0"}, false, "login ID"},
		{"missing password", []string{"--username=tester", "--char-slot=0"}, false, "password"},
		{"missing slot", []string{"--username=tester", "--password=test"}, false, "--char-slot"},
		{"invalid slot", []string{"--username=tester", "--password=test", "--char-slot=9"}, false, "between 0 and 8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultConfig()
			cfg.Login.KeepID, cfg.Login.SavedUsername = tc.keep, "remembered"
			err := applyCLI(&cfg, append([]string{"--headless"}, tc.args...))
			if tc.want != "" {
				if err == nil || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("error = %v, want %q", err, tc.want)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !cfg.Headless || !cfg.Login.AutoLogin || !cfg.Audio.Disabled {
				t.Fatal("headless mode must enable autologin and disable audio")
			}
		})
	}
}
