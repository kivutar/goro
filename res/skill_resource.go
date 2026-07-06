package res

import (
	"fmt"
	"strings"
)

var skillIDLuaCandidates = []string{
	"data\\luafiles514\\lua files\\skillinfoz\\skillid.lub",
	"data\\lua files\\skillinfoz\\skillid.lub",
	"lua files\\skillinfoz\\skillid.lub",
	"data\\luafiles514\\lua files\\skillinfo\\skillid.lub",
	"data\\lua files\\skillinfo\\skillid.lub",
	"lua files\\skillinfo\\skillid.lub",
}

var skillSPAmountCandidates = []string{
	"leveluseskillspamount.txt",
	"data\\leveluseskillspamount.txt",
	"data/leveluseskillspamount.txt",
}

var fallbackSkillResourceNames = map[int]string{
	1:  "NV_BASIC",
	2:  "SM_SWORD",
	3:  "SM_TWOHAND",
	4:  "SM_RECOVERY",
	5:  "SM_BASH",
	6:  "SM_PROVOKE",
	7:  "SM_MAGNUM",
	8:  "SM_ENDURE",
	9:  "MG_SRECOVERY",
	10: "MG_SIGHT",
	11: "MG_NAPALMBEAT",
	12: "MG_SAFETYWALL",
	13: "MG_SOULSTRIKE",
	14: "MG_COLDBOLT",
	15: "MG_FROSTDIVER",
	16: "MG_STONECURSE",
	17: "MG_FIREBALL",
	18: "MG_FIREWALL",
	19: "MG_FIREBOLT",
	20: "MG_LIGHTNINGBOLT",
	21: "MG_THUNDERSTORM",
	22: "AL_DP",
	23: "AL_DEMONBANE",
	24: "AL_RUWACH",
	25: "AL_PNEUMA",
	26: "AL_TELEPORT",
	27: "AL_WARP",
	28: "AL_HEAL",
	29: "AL_INCAGI",
	30: "AL_DECAGI",
	31: "AL_HOLYWATER",
	32: "AL_CRUCIS",
	33: "AL_ANGELUS",
	34: "AL_BLESSING",
	35: "AL_CURE",
}

func (m *Manager) SkillResourceName(skillID int) (string, bool) {
	if skillID <= 0 {
		return "", false
	}
	m.loadSkillResourceNames()
	name, ok := m.skillResourceNames[skillID]
	return name, ok && name != ""
}

func (m *Manager) SkillDisplayName(skillID int) (string, bool) {
	if skillID <= 0 {
		return "", false
	}
	m.loadSkillMetadata()
	name, ok := m.skillDisplayNames[skillID]
	return name, ok && name != ""
}

func (m *Manager) SkillDescription(skillID int) ([]string, bool) {
	if skillID <= 0 {
		return nil, false
	}
	m.loadSkillMetadata()
	lines, ok := m.skillDescriptions[skillID]
	if !ok || len(lines) == 0 {
		return nil, false
	}
	return append([]string(nil), lines...), true
}

func (m *Manager) SkillMaxLevel(skillID int) (int, bool) {
	if skillID <= 0 {
		return 0, false
	}
	m.loadSkillMaxLevels()
	level, ok := m.skillMaxLevels[skillID]
	return level, ok && level > 0
}

func (m *Manager) loadSkillResourceNames() {
	if m.skillResourceNamesLoaded {
		return
	}
	m.skillResourceNamesLoaded = true
	m.skillResourceNames = make(map[int]string, len(fallbackSkillResourceNames))
	for id, name := range fallbackSkillResourceNames {
		m.skillResourceNames[id] = name
	}
	globals := make(map[string]luaValue)
	_, data, ok := m.ReadFirst(skillIDLuaCandidates)
	if !ok {
		return
	}
	if err := executeLua51Bytecode(data, globals); err != nil {
		return
	}
	table := globals["SKID"]
	if table.kind != luaTable {
		return
	}
	for key, value := range table.table {
		name, ok := key.(string)
		if !ok || value.kind != luaNumber || name == "" {
			continue
		}
		id := int(value.num)
		if id > 0 {
			m.skillResourceNames[id] = name
		}
	}
}

func (m *Manager) loadSkillMetadata() {
	if m.skillMetadataLoaded {
		return
	}
	m.skillMetadataLoaded = true
	m.loadSkillResourceNames()
	m.skillDisplayNames = make(map[int]string)
	m.skillDescriptions = make(map[int][]string)
	nameToID := m.skillNameToID()
	if _, data, ok := m.ReadFirst(skillDataTableCandidates("skillnametable.txt")); ok {
		for id, name := range parseSkillNameTable(data, nameToID) {
			m.skillDisplayNames[id] = name
		}
	}
	if _, data, ok := m.ReadFirst(skillDataTableCandidates("skilldesctable.txt")); ok {
		names, descriptions := parseSkillDescriptionTable(data, nameToID)
		for id, name := range names {
			if m.skillDisplayNames[id] == "" {
				m.skillDisplayNames[id] = name
			}
		}
		for id, lines := range descriptions {
			m.skillDescriptions[id] = lines
		}
	}
}

func (m *Manager) loadSkillMaxLevels() {
	if m.skillMaxLevelsLoaded {
		return
	}
	m.skillMaxLevelsLoaded = true
	m.loadSkillResourceNames()
	m.skillMaxLevels = make(map[int]int)
	nameToID := m.skillNameToID()
	if _, data, ok := m.ReadFirst(skillSPAmountCandidates); ok {
		for id, level := range parseSkillSPAmountMaxLevels(data, nameToID) {
			m.skillMaxLevels[id] = level
		}
	}
}

func (m *Manager) skillNameToID() map[string]int {
	out := make(map[string]int, len(m.skillResourceNames))
	for id, name := range m.skillResourceNames {
		if name == "" {
			continue
		}
		out[strings.ToLower(strings.TrimSpace(name))] = id
	}
	return out
}

func skillDataTableCandidates(fileName string) []string {
	return []string{
		fileName,
		"data\\" + fileName,
		"data/" + fileName,
		"data\\luafiles514\\lua files\\skillinfoz\\" + fileName,
		"data/luafiles514/lua files/skillinfoz/" + fileName,
		"data\\lua files\\skillinfoz\\" + fileName,
		"data/lua files/skillinfoz/" + fileName,
		"lua files\\skillinfoz\\" + fileName,
		"lua files/skillinfoz/" + fileName,
		"data\\luafiles514\\lua files\\skillinfo\\" + fileName,
		"data/luafiles514/lua files/skillinfo/" + fileName,
		"data\\lua files\\skillinfo\\" + fileName,
		"data/lua files/skillinfo/" + fileName,
		"lua files\\skillinfo\\" + fileName,
		"lua files/skillinfo/" + fileName,
	}
}

func parseSkillNameTable(data []byte, nameToID map[string]int) map[int]string {
	out := make(map[int]string)
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r\n"))
		if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Split(line, "#")
		if len(tokens) < 2 {
			continue
		}
		id, ok := nameToID[strings.ToLower(strings.TrimSpace(tokens[0]))]
		if !ok || id <= 0 {
			continue
		}
		name := normalizeSkillDisplayToken(tokens[1])
		if name != "" {
			out[id] = name
		}
	}
	return out
}

func parseSkillDescriptionTable(data []byte, nameToID map[string]int) (map[int]string, map[int][]string) {
	names := make(map[int]string)
	descriptions := make(map[int][]string)
	currentID := 0
	expectTitle := false
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(rawLine, "\r\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "#" {
			currentID = 0
			expectTitle = false
			continue
		}
		if strings.HasSuffix(trimmed, "#") && len(trimmed) > 1 {
			key := strings.TrimSpace(strings.TrimSuffix(trimmed, "#"))
			currentID = nameToID[strings.ToLower(key)]
			expectTitle = currentID > 0
			continue
		}
		if currentID <= 0 {
			continue
		}
		if expectTitle {
			name := normalizeSkillDisplayToken(trimmed)
			if name != "" {
				names[currentID] = name
			}
			expectTitle = false
			continue
		}
		descriptions[currentID] = append(descriptions[currentID], line)
	}
	return names, descriptions
}

func parseSkillSPAmountMaxLevels(data []byte, nameToID map[string]int) map[int]int {
	out := make(map[int]int)
	currentID := 0
	levelCount := 0
	flush := func() {
		if currentID > 0 && levelCount > 0 {
			out[currentID] = levelCount
		}
		currentID = 0
		levelCount = 0
	}
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r\n"))
		if line == "" || strings.HasPrefix(line, "/") {
			continue
		}
		if line == "@" {
			flush()
			continue
		}
		if strings.HasSuffix(line, "#") {
			token := strings.TrimSpace(strings.TrimSuffix(line, "#"))
			if id, ok := nameToID[strings.ToLower(token)]; ok && id > 0 {
				flush()
				currentID = id
				continue
			}
			if currentID > 0 {
				levelCount++
			}
		}
	}
	flush()
	return out
}

func normalizeSkillDisplayToken(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "_", " ")
}

func SkillIconTextureCandidates(resource string, skillID int) []string {
	resource = strings.TrimSpace(strings.TrimSuffix(resource, ".bmp"))
	seen := make(map[string]struct{})
	out := make([]string, 0, 16)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	addStem := func(stem string) {
		if stem == "" {
			return
		}
		stem = strings.ReplaceAll(stem, "/", "\\")
		lower := strings.ToLower(stem)
		for _, candidateStem := range []string{lower, stem} {
			const uiKorPrefix = "data\\texture\\\xC0\xAF\xC0\xFA\xC0\xCE\xC5\xCD\xC6\xE4\xC0\xCC\xBD\xBA\\item\\"
			add(uiKorPrefix + candidateStem + ".bmp")
			add(strings.ReplaceAll(uiKorPrefix, "\\", "/") + candidateStem + ".bmp")
			add("texture\\\xC0\xAF\xC0\xFA\xC0\xCE\xC5\xCD\xC6\xE4\xC0\xCC\xBD\xBA\\item\\" + candidateStem + ".bmp")
			add("texture/item/" + candidateStem + ".bmp")
			add("data/texture/item/" + candidateStem + ".bmp")
			add(candidateStem + ".bmp")
		}
	}
	addStem(resource)
	if skillID > 0 {
		addStem(fmt.Sprintf("%d", skillID))
	}
	return out
}
