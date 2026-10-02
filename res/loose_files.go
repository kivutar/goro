package res

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type looseDirectory struct {
	modified time.Time
	names    map[string]string
}

// LooseFiles exposes files in the client folder, excluding archive contents.
// Names are relative to that folder on every platform.
func (m *Manager) LooseFiles() fs.FS {
	if m.files != nil {
		return m.files
	}
	return os.DirFS(m.Root)
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
		if _, err := m.statLoose(next); err != nil {
			names := m.looseDirectoryNames(current)
			actual, ok := names[strings.ToLower(part)]
			if !ok {
				return "", false
			}
			next = filepath.Join(current, actual)
		}
		current = next
	}
	if info, err := m.statLoose(current); err == nil && !info.IsDir() {
		return current, true
	}
	return "", false
}

func (m *Manager) looseDirectoryNames(directory string) map[string]string {
	info, err := m.statLoose(directory)
	if err != nil || !info.IsDir() {
		return nil
	}
	if cached, ok := m.looseDirectories.Load(directory); ok {
		entry := cached.(looseDirectory)
		if entry.modified.Equal(info.ModTime()) {
			return entry.names
		}
	}
	entries, err := m.readLooseDir(directory)
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

func (m *Manager) statLoose(name string) (fs.FileInfo, error) {
	if m.files != nil {
		return fs.Stat(m.files, filepath.ToSlash(name))
	}
	return os.Stat(name)
}

func (m *Manager) readLoose(name string) ([]byte, error) {
	if m.files != nil {
		return fs.ReadFile(m.files, filepath.ToSlash(name))
	}
	return os.ReadFile(name)
}

func (m *Manager) readLooseDir(name string) ([]fs.DirEntry, error) {
	if m.files != nil {
		return fs.ReadDir(m.files, filepath.ToSlash(name))
	}
	return os.ReadDir(name)
}
