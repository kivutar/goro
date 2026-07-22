package res

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/db"
)

const legacyMonsterSpriteRoot = "data\\sprite\\몬스터\\"
const mercenarySpriteRoot = "data\\sprite\\인간족\\몸통\\"

var npcIdentityLuaCandidates = []string{
	"data\\luafiles514\\lua files\\datainfo\\npcidentity.lub",
	"data\\lua files\\datainfo\\npcidentity.lub",
	"lua files\\datainfo\\npcidentity.lub",
}

var jobNameLuaCandidates = []string{
	"data\\luafiles514\\lua files\\datainfo\\jobname.lub",
	"data\\lua files\\datainfo\\jobname.lub",
	"lua files\\datainfo\\jobname.lub",
}

func (m *Manager) NonPCResourceName(job int) (string, bool) {
	if !m.nonPCResourceNamesLoaded {
		m.loadNonPCResourceNames()
	}
	name, ok := m.nonPCResourceNames[job]
	return name, ok && name != ""
}

func (m *Manager) loadNonPCResourceNames() {
	m.nonPCResourceNamesLoaded = true
	m.nonPCResourceNames = make(map[int]string)

	globals := make(map[string]luaValue)
	for _, candidates := range [][]string{npcIdentityLuaCandidates, jobNameLuaCandidates} {
		_, data, ok := m.ReadFirst(candidates)
		if !ok {
			continue
		}
		if err := executeLua51Bytecode(data, globals); err != nil {
			continue
		}
	}
	table := globals["JobNameTable"]
	if table.kind == luaTable {
		for key, value := range table.table {
			index, ok := key.(int)
			if !ok || value.kind != luaString || value.str == "" {
				continue
			}
			m.nonPCResourceNames[index] = value.str
		}
	}
	for id, name := range db.MonsterResourceName {
		if m.nonPCResourceNames[id] == "" {
			m.nonPCResourceNames[id] = name
		}
	}
}

func NonPCSpriteResourceCandidates(job int, resourceName string, extension string) []string {
	if resourceName == "" {
		return nil
	}
	name := normalizeNonPCSpriteResourceName(strings.TrimSuffix(resourceName, "."+extension))
	lowerName := strings.ToLower(name)
	seen := make(map[string]struct{})
	var out []string
	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	addStem := func(root string) {
		add(fmt.Sprintf("%s%s.%s", root, name, extension))
		if lowerName != name {
			add(fmt.Sprintf("%s%s.%s", root, lowerName, extension))
		}
	}

	if job >= 6001 && job <= 6047 {
		if job >= 6017 && job <= 6046 {
			addStem(mercenarySpriteRoot)
			addStem("data\\sprite\\mercenary\\")
		} else {
			addStem("data\\sprite\\homun\\")
		}
		return out
	}
	if job >= 1000 {
		addStem("data\\sprite\\monster\\")
		addStem(legacyMonsterSpriteRoot)
		addStem("data\\sprite\\")
		return out
	}
	addStem("data\\sprite\\NPC\\")
	addStem("data\\sprite\\npc\\")
	return out
}

func normalizeNonPCSpriteResourceName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "\\")
	for strings.Contains(name, "\\\\") {
		name = strings.ReplaceAll(name, "\\\\", "\\")
	}
	return name
}
