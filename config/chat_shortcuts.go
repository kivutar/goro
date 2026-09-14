package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ChatShortcuts are client-side Alt+1 through Alt+0 commands, independent of
// the server's item and skill hotkey slots.
type ChatShortcuts [10]string

func defaultChatShortcuts() ChatShortcuts {
	return ChatShortcuts{"/!", "/?", "/ho", "/lv", "/swt", "/ic", "/an", "/ag", "/$", "/..."}
}

// LoadChatShortcuts refreshes saved bindings when entering the game again;
// the startup Config may predate edits made before character selection.
func LoadChatShortcuts(defaults ChatShortcuts) (ChatShortcuts, error) {
	cfg := defaultConfig()
	cfg.ChatShortcuts = defaults
	path, err := UserConfigPath()
	if err != nil {
		return defaults, err
	}
	if err := applyINIFile(&cfg, path, false); err != nil {
		return defaults, err
	}
	return cfg.ChatShortcuts, nil
}

func SaveChatShortcuts(commands ChatShortcuts) (string, error) {
	values := make(map[string]string, len(commands))
	for slot, command := range commands {
		if strings.ContainsAny(command, "\r\n\x00") {
			return "", fmt.Errorf("shortcut must be a single line without NUL characters")
		}
		values[strconv.Itoa((slot+1)%10)] = `"` + command + `"`
	}
	return saveUserConfigValues(map[string]map[string]string{"chatshortcuts": values})
}
