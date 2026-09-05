package res

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kivutar/goro/db"
	glua "github.com/yuin/gopher-lua"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

type ItemMetadata struct {
	UnidentifiedDisplayName string
	IdentifiedDisplayName   string
	UnidentifiedResource    string
	IdentifiedResource      string
	CardIllustration        string
	UnidentifiedDescription []string
	IdentifiedDescription   []string
	SlotCount               int
	ClassNum                int
	ClassNumSet             bool
	CardPrefixName          string
	CardPostfix             bool
}

var itemInfoLuaCandidates = []string{
	"System\\itemInfo.lub",
	"System/itemInfo.lub",
	"data\\luafiles514\\lua files\\datainfo\\itemInfo.lub",
	"data\\luafiles514\\lua files\\datainfo\\iteminfo.lub",
	"data\\lua files\\datainfo\\itemInfo.lub",
	"data\\lua files\\datainfo\\iteminfo.lub",
	"lua files\\datainfo\\itemInfo.lub",
	"lua files\\datainfo\\iteminfo.lub",
}

var itemDisplayTableFiles = []struct {
	name       string
	identified bool
}{
	{name: "num2itemdisplaynametable.txt"},
	{name: "idnum2itemdisplaynametable.txt", identified: true},
}

var itemResourceTableFiles = []struct {
	name       string
	identified bool
}{
	{name: "num2itemresnametable.txt"},
	{name: "idnum2itemresnametable.txt", identified: true},
}

var itemDescriptionTableFiles = []struct {
	name       string
	identified bool
}{
	{name: "num2itemdesctable.txt"},
	{name: "idnum2itemdesctable.txt", identified: true},
}

var itemSlotCountTableFiles = []string{
	"itemslotcounttable.txt",
}

var itemCardPrefixTableFiles = []string{
	"cardprefixnametable.txt",
}

var itemCardIllustrationTableFiles = []string{
	"num2cardillustnametable.txt",
}

var itemCardPostfixTableFiles = []string{
	"cardpostfixnametable.txt",
}

func (m *Manager) ItemDisplayName(itemID int, identified bool) (string, bool) {
	if itemID <= 0 {
		return "", false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok {
		return "", false
	}
	if identified {
		if metadata.IdentifiedDisplayName != "" {
			return metadata.IdentifiedDisplayName, true
		}
		if metadata.UnidentifiedDisplayName != "" {
			return metadata.UnidentifiedDisplayName, true
		}
	} else {
		if metadata.UnidentifiedDisplayName != "" {
			return metadata.UnidentifiedDisplayName, true
		}
		if metadata.IdentifiedDisplayName != "" {
			return metadata.IdentifiedDisplayName, true
		}
	}
	return "", false
}

func (m *Manager) ItemResourceName(itemID int, identified bool) (string, bool) {
	if itemID <= 0 {
		return "", false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok {
		return "", false
	}
	if identified {
		if metadata.IdentifiedResource != "" {
			return metadata.IdentifiedResource, true
		}
		if metadata.UnidentifiedResource != "" {
			return metadata.UnidentifiedResource, true
		}
	} else {
		if metadata.UnidentifiedResource != "" {
			return metadata.UnidentifiedResource, true
		}
		if metadata.IdentifiedResource != "" {
			return metadata.IdentifiedResource, true
		}
	}
	return "", false
}

func (m *Manager) ItemDescription(itemID int, identified bool) ([]string, bool) {
	if itemID <= 0 {
		return nil, false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok {
		return nil, false
	}
	if identified {
		if len(metadata.IdentifiedDescription) > 0 {
			return append([]string(nil), metadata.IdentifiedDescription...), true
		}
		if len(metadata.UnidentifiedDescription) > 0 {
			return append([]string(nil), metadata.UnidentifiedDescription...), true
		}
	} else {
		if len(metadata.UnidentifiedDescription) > 0 {
			return append([]string(nil), metadata.UnidentifiedDescription...), true
		}
		if len(metadata.IdentifiedDescription) > 0 {
			return append([]string(nil), metadata.IdentifiedDescription...), true
		}
	}
	return nil, false
}

func (m *Manager) ItemSlotCount(itemID int) (int, bool) {
	if itemID <= 0 {
		return 0, false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok || metadata.SlotCount <= 0 {
		return 0, false
	}
	return metadata.SlotCount, true
}

func (m *Manager) ItemClassNum(itemID int) (int, bool) {
	if itemID <= 0 {
		return 0, false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok || !metadata.ClassNumSet {
		return 0, false
	}
	return metadata.ClassNum, true
}

func (m *Manager) ItemCardPrefixName(itemID int) (string, bool) {
	if itemID <= 0 {
		return "", false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok || metadata.CardPrefixName == "" {
		return "", false
	}
	return metadata.CardPrefixName, true
}

func (m *Manager) ItemCardIllustrationName(itemID int) (string, bool) {
	if itemID <= 0 {
		return "", false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	if !ok || metadata.CardIllustration == "" {
		return "", false
	}
	return metadata.CardIllustration, true
}

func (m *Manager) ItemCardPostfix(itemID int) bool {
	if itemID <= 0 {
		return false
	}
	m.loadItemMetadata()
	metadata, ok := m.itemMetadata[itemID]
	return ok && metadata.CardPostfix
}

func (m *Manager) loadItemMetadata() {
	if m.itemMetadataLoaded {
		return
	}
	m.itemMetadataLoaded = true
	m.itemMetadata = make(map[int]ItemMetadata)
	for id, item := range db.ItemTable {
		metadata := m.itemMetadata[id]
		metadata.ClassNum = item.ClassNum
		metadata.ClassNumSet = true
		m.itemMetadata[id] = metadata
	}
	for _, table := range itemDisplayTableFiles {
		for id, value := range m.readItemPairTable(table.name) {
			metadata := m.itemMetadata[id]
			value = normalizeItemDisplayToken(value)
			if table.identified {
				metadata.IdentifiedDisplayName = value
			} else {
				metadata.UnidentifiedDisplayName = value
			}
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemResourceTableFiles {
		for id, value := range m.readItemPairTable(table.name) {
			metadata := m.itemMetadata[id]
			if table.identified {
				metadata.IdentifiedResource = value
			} else {
				metadata.UnidentifiedResource = value
			}
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemDescriptionTableFiles {
		for id, value := range m.readItemDescriptionTable(table.name) {
			metadata := m.itemMetadata[id]
			if table.identified {
				metadata.IdentifiedDescription = value
			} else {
				metadata.UnidentifiedDescription = value
			}
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemSlotCountTableFiles {
		for id, value := range m.readItemPairTable(table) {
			slotCount, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || slotCount <= 0 {
				continue
			}
			metadata := m.itemMetadata[id]
			metadata.SlotCount = slotCount
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemCardIllustrationTableFiles {
		for id, value := range m.readItemPairTable(table) {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			metadata := m.itemMetadata[id]
			metadata.CardIllustration = value
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemCardPrefixTableFiles {
		for id, value := range m.readItemPairTable(table) {
			metadata := m.itemMetadata[id]
			metadata.CardPrefixName = normalizeItemDisplayToken(value)
			m.itemMetadata[id] = metadata
		}
	}
	for _, table := range itemCardPostfixTableFiles {
		for id := range m.readItemIDSetTable(table) {
			metadata := m.itemMetadata[id]
			metadata.CardPostfix = true
			m.itemMetadata[id] = metadata
		}
	}
	m.loadItemInfoLuaMetadata()
}

func (m *Manager) loadItemInfoLuaMetadata() {
	_, data, ok := m.ReadFirst(itemInfoLuaCandidates)
	if !ok {
		return
	}
	globals := make(map[string]luaValue)
	_ = executeLua51BytecodeWithSetup(data, globals, func(state *glua.LState) {
		state.SetGlobal("AddItem", state.NewFunction(func(state *glua.LState) int {
			id := state.OptInt(1, 0)
			if id <= 0 {
				return 1
			}
			metadata := m.itemMetadata[id]
			metadata.UnidentifiedDisplayName = normalizeItemDisplayToken(luaOptionalString(state, 2))
			metadata.UnidentifiedResource = luaOptionalString(state, 3)
			metadata.IdentifiedDisplayName = normalizeItemDisplayToken(luaOptionalString(state, 4))
			metadata.IdentifiedResource = luaOptionalString(state, 5)
			metadata.SlotCount = state.OptInt(6, 0)
			metadata.ClassNum = state.OptInt(7, 0)
			metadata.ClassNumSet = true
			m.itemMetadata[id] = metadata
			return 1
		}))
		state.SetGlobal("AddItemUnidentifiedDesc", state.NewFunction(func(state *glua.LState) int {
			id := state.OptInt(1, 0)
			text := luaOptionalString(state, 2)
			if id > 0 && text != "" {
				metadata := m.itemMetadata[id]
				metadata.UnidentifiedDescription = append(metadata.UnidentifiedDescription, text)
				m.itemMetadata[id] = metadata
			}
			return 1
		}))
		state.SetGlobal("AddItemIdentifiedDesc", state.NewFunction(func(state *glua.LState) int {
			id := state.OptInt(1, 0)
			text := luaOptionalString(state, 2)
			if id > 0 && text != "" {
				metadata := m.itemMetadata[id]
				metadata.IdentifiedDescription = append(metadata.IdentifiedDescription, text)
				m.itemMetadata[id] = metadata
			}
			return 1
		}))
		for _, name := range []string{"AddItemEffectInfo", "AddItemIsCostume", "AddItemPackageID"} {
			state.SetGlobal(name, state.NewFunction(func(state *glua.LState) int { return 1 }))
		}
	})
}

func luaOptionalString(state *glua.LState, index int) string {
	value := state.Get(index)
	if value == glua.LNil || value == nil {
		return ""
	}
	return value.String()
}

func (m *Manager) readItemPairTable(fileName string) map[int]string {
	for _, candidate := range itemTableCandidates(fileName) {
		data, err := m.ReadFile(candidate)
		if err != nil {
			continue
		}
		return parseItemPairTable(data)
	}
	return nil
}

func (m *Manager) readItemDescriptionTable(fileName string) map[int][]string {
	for _, candidate := range itemTableCandidates(fileName) {
		data, err := m.ReadFile(candidate)
		if err != nil {
			continue
		}
		return parseItemDescriptionTable(data)
	}
	return nil
}

func (m *Manager) readItemIDSetTable(fileName string) map[int]struct{} {
	for _, candidate := range itemTableCandidates(fileName) {
		data, err := m.ReadFile(candidate)
		if err != nil {
			continue
		}
		return parseItemIDSetTable(data)
	}
	return nil
}

func itemTableCandidates(fileName string) []string {
	return []string{
		fileName,
		"data\\" + fileName,
		"data/" + fileName,
	}
}

func parseItemDescriptionTable(data []byte) map[int][]string {
	out := make(map[int][]string)
	currentID := 0
	var lines []string
	flush := func() {
		if currentID > 0 {
			out[currentID] = append([]string(nil), lines...)
		}
		currentID = 0
		lines = nil
	}
	for _, rawLine := range strings.Split(decodeItemTableText(data), "\n") {
		line := strings.TrimRight(rawLine, "\r\n")
		if currentID == 0 {
			id, ok := parseItemDescriptionHeader(line)
			if ok {
				currentID = id
				lines = nil
			}
			continue
		}
		if strings.TrimSpace(line) == "#" {
			flush()
			continue
		}
		lines = append(lines, line)
	}
	flush()
	return out
}

func parseItemDescriptionHeader(line string) (int, bool) {
	if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "#") {
		return 0, false
	}
	firstHash := strings.IndexByte(line, '#')
	if firstHash <= 0 {
		return 0, false
	}
	if strings.TrimSpace(line[firstHash+1:]) != "" {
		return 0, false
	}
	id, err := strconv.ParseUint(line[:firstHash], 10, 32)
	if err != nil || id == 0 {
		return 0, false
	}
	return int(id), true
}

func parseItemPairTable(data []byte) map[int]string {
	out := make(map[int]string)
	for _, rawLine := range strings.Split(decodeItemTableText(data), "\n") {
		line := strings.TrimRight(rawLine, "\r\n")
		if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "#") {
			continue
		}
		firstHash := strings.IndexByte(line, '#')
		if firstHash <= 0 {
			continue
		}
		secondHash := strings.IndexByte(line[firstHash+1:], '#')
		if secondHash < 0 {
			continue
		}
		secondHash += firstHash + 1
		id, err := strconv.ParseUint(line[:firstHash], 10, 32)
		if err != nil || id == 0 {
			continue
		}
		out[int(id)] = line[firstHash+1 : secondHash]
	}
	return out
}

func parseItemIDSetTable(data []byte) map[int]struct{} {
	out := make(map[int]struct{})
	for _, rawLine := range strings.Split(decodeItemTableText(data), "\n") {
		line := strings.TrimRight(rawLine, "\r\n")
		if line == "" || strings.HasPrefix(line, "/") || strings.HasPrefix(line, "#") {
			continue
		}
		firstHash := strings.IndexByte(line, '#')
		if firstHash <= 0 {
			continue
		}
		id, err := strconv.ParseUint(line[:firstHash], 10, 32)
		if err != nil || id == 0 {
			continue
		}
		out[int(id)] = struct{}{}
	}
	return out
}

func decodeItemTableText(data []byte) string {
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), data)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}

func normalizeItemDisplayToken(value string) string {
	return strings.ReplaceAll(value, "_", " ")
}

func ItemSpriteResourceCandidates(resource string, extension string) []string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		return nil
	}
	stem := normalizeItemResourceSlash(resource)
	filenameOnly := stem
	if pos := strings.LastIndexAny(filenameOnly, `\/`); pos >= 0 && pos+1 < len(filenameOnly) {
		filenameOnly = filenameOnly[pos+1:]
	}

	seen := make(map[string]struct{})
	out := make([]string, 0, 24)
	add := func(candidate string) {
		if candidate == "" {
			return
		}
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
		if strings.Contains(candidate, "\\") {
			slashed := strings.ReplaceAll(candidate, "\\", "/")
			if _, ok := seen[slashed]; !ok {
				seen[slashed] = struct{}{}
				out = append(out, slashed)
			}
		}
	}
	addStem := func(prefix string) {
		add(prefix + stem + "." + extension)
		if filenameOnly != stem {
			add(prefix + filenameOnly + "." + extension)
		}
	}

	for _, prefix := range itemSpritePrefixes() {
		addStem(prefix)
	}
	addStem("")
	return out
}

func normalizeItemResourceSlash(resource string) string {
	return strings.ReplaceAll(resource, "/", "\\")
}

func itemSpritePrefixes() []string {
	const itemKorPrefix = "data\\sprite\\아이템\\"
	return []string{
		itemKorPrefix,
		"sprite\\아이템\\",
		"data\\sprite\\item\\",
		"sprite\\item\\",
		"data\\sprite\\items\\",
		"sprite\\items\\",
	}
}

func FormatGroundItemLabel(name string, amount int) string {
	if name == "" {
		name = "Item"
	}
	if amount <= 0 {
		amount = 1
	}
	return fmt.Sprintf("%s: %d ea", name, amount)
}
