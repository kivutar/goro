package res

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type looseDirectory struct {
	modified time.Time
	names    map[string]string
}

// Find tries the exact spelling first. On case-sensitive filesystems, walk
// only the requested path's components to provide the same lookup as a GRF.
// Cache directory listings instead of recursively indexing the client folder.
func (m *Manager) findCaseInsensitive(name string) (string, bool) {
	name = resourceAliasPath(name)
	if name == "" {
		return "", false
	}
	current := m.Root
	for _, part := range strings.Split(name, "/") {
		next := filepath.Join(current, part)
		if _, err := os.Stat(next); err != nil {
			names := m.looseDirectoryNames(current)
			actual, ok := names[strings.ToLower(part)]
			if !ok {
				return "", false
			}
			next = filepath.Join(current, actual)
		}
		current = next
	}
	if info, err := os.Stat(current); err == nil && !info.IsDir() {
		return current, true
	}
	return "", false
}

func (m *Manager) looseDirectoryNames(directory string) map[string]string {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	if cached, ok := m.looseDirectories.Load(directory); ok {
		entry := cached.(looseDirectory)
		if entry.modified.Equal(info.ModTime()) {
			return entry.names
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(entries))
	for _, entry := range entries {
		key := strings.ToLower(entry.Name())
		if _, exists := names[key]; !exists {
			// ReadDir sorts names, making ambiguous case variants stable.
			names[key] = entry.Name()
		}
	}
	m.looseDirectories.Store(directory, looseDirectory{modified: info.ModTime(), names: names})
	return names
}
