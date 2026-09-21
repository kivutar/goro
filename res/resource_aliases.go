package res

import (
	"os"
	"path"
	"strings"
)

// resnametable.txt names resources relative to data/ (map geometry, models,
// etc.) or data/texture/ (including minimaps). Resolve the whole relative path,
// never just its basename, so unrelated directories cannot share an alias.
func (m *Manager) resourceAlias(name string) (string, bool) {
	m.resourceAliasesOnce.Do(m.loadResourceAliases)
	name = strings.ToLower(resourceAliasPath(name))
	for _, base := range []string{"data/texture/", "data/", "texture/", ""} {
		if relative, ok := strings.CutPrefix(name, base); ok {
			if target, ok := m.resourceAliases[relative]; ok {
				return base + target, true
			}
		}
	}
	return "", false
}

func (m *Manager) loadResourceAliases() {
	m.resourceAliases = make(map[string]string)
	// Like the original client's CResMgr::ReadResNameTable, use one table
	// selected by normal resource priority. An empty replacement table can
	// deliberately disable aliases from a lower-priority archive.
	// Read directly to avoid consulting the table while loading the table.
	names := []string{"data/resnametable.txt", "resnametable.txt"}
	for _, name := range names {
		if filename, ok := m.Find(name); ok {
			if data, err := os.ReadFile(filename); err == nil {
				m.resourceAliases = parseResourceAliases(data)
				return
			}
		}
	}
	for _, archive := range m.Archives {
		if archive == nil {
			continue
		}
		for _, name := range names {
			if data, err := archive.ReadFile(name); err == nil {
				m.resourceAliases = parseResourceAliases(data)
				return
			}
		}
	}
}

func parseResourceAliases(data []byte) map[string]string {
	aliases := make(map[string]string)
	// Original tables use EUC-KR; patched tables may already be UTF-8.
	text := strings.TrimPrefix(decodeGRFName(data), "\ufeff")
	for _, line := range strings.Split(text, "\n") {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Split(line, "#")
		for i := 0; i+2 < len(fields); i += 2 {
			key := strings.ToLower(resourceAliasPath(fields[i]))
			target := resourceAliasPath(fields[i+1])
			if key != "" && target != "" {
				if _, exists := aliases[key]; !exists {
					// The original table reader keeps the first definition.
					aliases[key] = target
				}
			}
		}
	}
	return aliases
}

func resourceAliasPath(name string) string {
	name = path.Clean(strings.ReplaceAll(strings.TrimSpace(decodeGRFName([]byte(name))), "\\", "/"))
	if name == "." || name == ".." || strings.HasPrefix(name, "../") || path.IsAbs(name) || strings.Contains(name, ":") {
		return ""
	}
	// Preserve target case for loose files on case-sensitive filesystems.
	return name
}
