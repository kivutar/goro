package res

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/db"
)

var skillIDLuaCandidates = []string{
	"data\\luafiles514\\lua files\\skillinfoz\\skillid.lub",
	"data\\lua files\\skillinfoz\\skillid.lub",
	"lua files\\skillinfoz\\skillid.lub",
	"data\\luafiles514\\lua files\\skillinfo\\skillid.lub",
	"data\\lua files\\skillinfo\\skillid.lub",
	"lua files\\skillinfo\\skillid.lub",
}

var skillTreeViewLuaCandidates = []string{
	"data\\luafiles514\\lua files\\skillinfoz\\skilltreeview.lub",
	"data\\lua files\\skillinfoz\\skilltreeview.lub",
	"lua files\\skillinfoz\\skilltreeview.lub",
	"data\\luafiles514\\lua files\\skillinfo\\skilltreeview.lub",
	"data\\lua files\\skillinfo\\skilltreeview.lub",
	"lua files\\skillinfo\\skilltreeview.lub",
}

var jobIdentityLuaCandidates = []string{
	"data\\luafiles514\\lua files\\datainfo\\jobidentity.lub",
	"data\\lua files\\datainfo\\jobidentity.lub",
	"lua files\\datainfo\\jobidentity.lub",
}

var skillSPAmountCandidates = []string{
	"leveluseskillspamount.txt",
	"data\\leveluseskillspamount.txt",
	"data/leveluseskillspamount.txt",
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

// SkillTreePositions returns the read-only client-defined grid position for
// each skill belonging to job. Positions are zero-based in seven columns.
func (m *Manager) SkillTreePositions(job int) (map[int]int, bool) {
	if job < 0 {
		return nil, false
	}
	m.loadSkillTreePositions()
	positions, ok := m.skillTreePositions[job]
	if !ok || len(positions) == 0 {
		return nil, false
	}
	return positions, true
}

func (m *Manager) loadSkillResourceNames() {
	if m.skillResourceNamesLoaded {
		return
	}
	m.skillResourceNamesLoaded = true
	m.skillResourceNames = make(map[int]string, len(db.SkillResourceName))
	for id, name := range db.SkillResourceName {
		m.skillResourceNames[int(id)] = name
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

func (m *Manager) loadSkillTreePositions() {
	if m.skillTreePositionsLoaded {
		return
	}
	m.skillTreePositionsLoaded = true
	m.skillTreePositions = make(map[int]map[int]int)
	m.loadSkillResourceNames()

	globals := make(map[string]luaValue)
	_, jobData, ok := m.ReadFirst(jobIdentityLuaCandidates)
	if !ok || executeLua51Bytecode(jobData, globals) != nil {
		return
	}
	// Older clients expose job constants as JTtbl while SkillTreeView expects
	// the same table under JOBID.
	jobs := globals["JOBID"]
	if jobs.kind != luaTable {
		jobs = globals["JTtbl"]
		if jobs.kind != luaTable {
			return
		}
		globals["JOBID"] = jobs
	}
	skillIDs := make(map[interface{}]luaValue, len(m.skillResourceNames))
	for id, name := range m.skillResourceNames {
		if id > 0 && name != "" {
			skillIDs[name] = luaValue{kind: luaNumber, num: float64(id)}
		}
	}
	globals["SKID"] = luaValue{kind: luaTable, table: skillIDs}
	_, treeData, ok := m.ReadFirst(skillTreeViewLuaCandidates)
	if !ok || executeLua51Bytecode(treeData, globals) != nil {
		return
	}
	m.skillTreePositions = parseSkillTreePositions(globals["SKILL_TREEVIEW_FOR_JOB"])
}

func parseSkillTreePositions(table luaValue) map[int]map[int]int {
	out := make(map[int]map[int]int)
	if table.kind != luaTable {
		return out
	}
	for rawJob, rawPositions := range table.table {
		job, ok := rawJob.(int)
		if !ok || job < 0 || rawPositions.kind != luaTable {
			continue
		}
		positions := make(map[int]int)
		for rawPosition, rawSkill := range rawPositions.table {
			position, ok := rawPosition.(int)
			if !ok || position < 0 || rawSkill.kind != luaNumber {
				continue
			}
			skillID := int(rawSkill.num)
			if skillID > 0 {
				positions[skillID] = position
			}
		}
		if len(positions) > 0 {
			out[job] = positions
		}
	}
	return out
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
			const uiKorPrefix = "data\\texture\\유저인터페이스\\item\\"
			add(uiKorPrefix + candidateStem + ".bmp")
			add(strings.ReplaceAll(uiKorPrefix, "\\", "/") + candidateStem + ".bmp")
			add("texture\\유저인터페이스\\item\\" + candidateStem + ".bmp")
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
