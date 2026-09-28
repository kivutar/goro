package res

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func (m *Manager) scanKnownFiles() error {
	for _, name := range clientInfoCandidates {
		if path, ok := m.Find(name); ok {
			m.FoundFiles = append(m.FoundFiles, path)
		}
	}

	var paths []string
	if iniPath, ok := m.Find("DATA.INI"); ok {
		data, err := os.ReadFile(iniPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", iniPath, err)
		}
		names, err := parseDataINI(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", iniPath, err)
		}
		m.FoundFiles = append(m.FoundFiles, iniPath)
		for _, name := range names {
			// Windows-style relative paths must also work on Unix clients.
			path := filepath.FromSlash(strings.ReplaceAll(name, "\\", "/"))
			if !filepath.IsAbs(path) {
				if found, ok := m.Find(path); ok {
					path = found
				} else {
					path = filepath.Join(m.Root, path)
				}
			}
			paths = append(paths, path)
		}
	} else {
		// Only these compatibility layers are implicit. Event and custom
		// archives must be explicitly selected through DATA.INI.
		for _, name := range []string{"fdata.grf", "rdata.grf", "sdata.grf", "data.grf"} {
			if path, ok := m.Find(name); ok {
				paths = append(paths, path)
			}
		}
	}

	seen := make(map[string]bool)
	for _, path := range paths {
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		archive, err := OpenGRF(path)
		if err != nil {
			return fmt.Errorf("open resource archive %s: %w", path, err)
		}
		seen[path] = true
		m.Archives = append(m.Archives, archive)
		m.FoundFiles = append(m.FoundFiles, path)
	}
	return nil
}

// parseDataINI returns only the archives explicitly listed in [Data], in
// increasing numeric priority. An empty [Data] section selects no archives.
func parseDataINI(data []byte) ([]string, error) {
	reader := transform.NewReader(bytes.NewReader(data), unicode.BOMOverride(transform.Nop))
	scanner := bufio.NewScanner(reader)
	section := ""
	foundData := false
	entries := make(map[int]string)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			name, tail, ok := strings.Cut(line[1:], "]")
			tail = strings.TrimSpace(tail)
			if !ok || tail != "" && !strings.HasPrefix(tail, ";") && !strings.HasPrefix(tail, "#") {
				return nil, fmt.Errorf("line %d: invalid section header", lineNo)
			}
			section = strings.ToLower(strings.TrimSpace(name))
			foundData = foundData || section == "data"
			continue
		}
		if section != "data" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("line %d: expected priority=archive", lineNo)
		}
		priority, err := strconv.Atoi(strings.TrimSpace(key))
		if err != nil || priority < 0 {
			return nil, fmt.Errorf("line %d: invalid archive priority %q", lineNo, key)
		}
		if _, exists := entries[priority]; exists {
			return nil, fmt.Errorf("line %d: duplicate archive priority %d", lineNo, priority)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"' || value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
		entries[priority] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !foundData {
		return nil, fmt.Errorf("missing [Data] section")
	}
	priorities := make([]int, 0, len(entries))
	for priority := range entries {
		priorities = append(priorities, priority)
	}
	sort.Ints(priorities)
	var names []string
	for _, priority := range priorities {
		if entries[priority] != "" {
			names = append(names, entries[priority])
		}
	}
	return names, nil
}
