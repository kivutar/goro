package res

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/db"
)

const (
	playerHumanSpriteRoot = "data\\sprite\\인간족\\"
	playerAccessoryRoot   = "data\\sprite\\악세사리\\"
	playerShieldRoot      = "data\\sprite\\방패\\"
	playerCartRoot        = "data\\sprite\\이팩트\\"
	playerPaletteRoot     = "data\\palette\\"
	playerIMFRoot         = "data\\imf\\"
	playerBodyDir         = "몸통"
	playerHeadDir         = "머리통"
	mercenaryWeaponRoot   = "data\\sprite\\인간족\\용병\\"
	playerPaletteBodyDir  = "몸"
	playerPaletteHeadDir  = "머리"
	playerPaletteHeadFile = "머리"
	playerFemaleSex       = "여"
	playerMaleSex         = "남"
	weaponLightSuffix     = "검광"
)

var playerCartTokens = []string{
	"슈노손수레",
	"손수레",
	"손수레1",
	"손수레2",
	"손수레3",
	"손수레4",
	"손수레5",
	"손수레6",
	"손수레7",
	"손수레8",
	"선물상자카트",
	"포링싣은카트",
	"포링카트",
	"마도카트",
}

var accessoryLuaCandidates = []string{
	"data\\luafiles514\\lua files\\datainfo\\accessoryid.lub",
	"data\\lua files\\datainfo\\accessoryid.lub",
	"lua files\\datainfo\\accessoryid.lub",
}

var accessoryNameLuaCandidates = []string{
	"data\\luafiles514\\lua files\\datainfo\\accname.lub",
	"data\\lua files\\datainfo\\accname.lub",
	"lua files\\datainfo\\accname.lub",
}

func PlayerBodyResourceCandidates(job int, sex byte, extension string) []string {
	sexToken := PlayerSexToken(sex)
	tokens := []string{PlayerJobToken(job)}
	if tokens[0] != PlayerJobToken(0) {
		tokens = append(tokens, PlayerJobToken(0))
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, fmt.Sprintf("%s%s\\%s\\%s_%s.%s", playerHumanSpriteRoot, playerBodyDir, sexToken, token, sexToken, extension))
	}
	return out
}

func PlayerCartResourceCandidates(cartNum int, extension string) []string {
	if cartNum < 0 {
		cartNum = 0
	}
	if cartNum >= len(playerCartTokens) {
		cartNum = len(playerCartTokens) - 1
	}
	return []string{
		fmt.Sprintf("%s%s.%s", playerCartRoot, playerCartTokens[cartNum], extension),
	}
}

func PlayerBodyResourcePath(job int, sex byte, extension string) string {
	sexToken := PlayerSexToken(sex)
	return fmt.Sprintf("%s%s\\%s\\%s_%s.%s", playerHumanSpriteRoot, playerBodyDir, sexToken, PlayerJobToken(job), sexToken, extension)
}

func PlayerHeadResourceCandidates(job int, head int, sex byte, extension string) []string {
	sexToken := PlayerSexToken(sex)
	head = NormalizePlayerHead(head, job)
	return []string{
		fmt.Sprintf("%s%s\\%s\\%d_%s.%s", playerHumanSpriteRoot, playerHeadDir, sexToken, head, sexToken, extension),
	}
}

func PlayerAccessoryResourceCandidates(job int, head int, sex byte, viewID int, resourceName string, extension string) []string {
	if viewID == 185 {
		return PlayerHeadResourceCandidates(job, head, sex, extension)
	}
	if viewID <= 0 || resourceName == "" {
		return nil
	}
	sexToken := PlayerSexToken(sex)
	separator := "_"
	if resourceName[0] == '_' {
		separator = ""
	}
	return []string{
		fmt.Sprintf("%s%s\\%s%s%s.%s", playerAccessoryRoot, sexToken, sexToken, separator, resourceName, extension),
	}
}

func (m *Manager) AccessoryResourceName(viewID int) (string, bool) {
	if viewID <= 0 {
		return "", false
	}
	if !m.accessoryNamesLoaded {
		m.loadAccessoryResourceNames()
	}
	name, ok := m.accessoryNames[viewID]
	if ok && name != "" {
		return name, true
	}
	name, ok = db.HatResourceName[viewID]
	return name, ok && name != ""
}

func (m *Manager) loadAccessoryResourceNames() {
	m.accessoryNamesLoaded = true
	m.accessoryNames = make(map[int]string)
	globals := make(map[string]luaValue)
	for _, candidates := range [][]string{accessoryLuaCandidates, accessoryNameLuaCandidates} {
		_, data, ok := m.ReadFirst(candidates)
		if !ok {
			return
		}
		if err := executeLua51Bytecode(data, globals); err != nil {
			return
		}
	}
	table := globals["AccNameTable"]
	if table.kind != luaTable {
		return
	}
	for key, value := range table.table {
		index, ok := key.(int)
		if !ok || value.kind != luaString || value.str == "" {
			continue
		}
		m.accessoryNames[index] = value.str
	}
}

func PlayerIMFResourceCandidates(job int, sex byte) []string {
	sexToken := PlayerSexToken(sex)
	tokens := []string{PlayerJobToken(job)}
	if tokens[0] != PlayerJobToken(0) {
		tokens = append(tokens, PlayerJobToken(0))
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, fmt.Sprintf("%s%s_%s.imf", playerIMFRoot, token, sexToken))
	}
	return out
}

func PlayerWeaponOverlayResourceCandidates(job int, sex byte, weaponValue int, secondLayer bool, extension string) []string {
	if weaponValue <= 0 {
		return nil
	}
	weaponType := PlayerWeaponOverlayTypeForJob(job, db.PlayerWeaponType(weaponValue), false)
	if weaponType <= 0 {
		return nil
	}
	if secondLayer && !PlayerWeaponOverlaySupportsSecondLayer(weaponType) {
		return nil
	}
	token := PlayerWeaponOverlayToken(weaponType)
	if token == "" {
		return nil
	}
	sexToken := PlayerSexToken(sex)
	jobToken := PlayerJobToken(job)
	out := make([]string, 0, 2)
	if secondLayer {
		out = append(out,
			fmt.Sprintf("%s%s\\%s_%s_%d_%s.%s", playerHumanSpriteRoot, jobToken, jobToken, sexToken, weaponValue&0xFFFF, weaponLightSuffix, extension),
			fmt.Sprintf("%s%s\\%s_%s_%s_%s.%s", playerHumanSpriteRoot, jobToken, jobToken, sexToken, token, weaponLightSuffix, extension),
		)
		return out
	}
	out = append(out,
		fmt.Sprintf("%s%s\\%s_%s_%d.%s", playerHumanSpriteRoot, jobToken, jobToken, sexToken, weaponValue&0xFFFF, extension),
		fmt.Sprintf("%s%s\\%s_%s_%s.%s", playerHumanSpriteRoot, jobToken, jobToken, sexToken, token, extension),
	)
	return out
}

func MercenaryWeaponOverlayResourceCandidates(resourceName string, weaponValue int, secondLayer bool, extension string) []string {
	if resourceName == "" || weaponValue <= 0 {
		return nil
	}
	weaponType := mercenaryWeaponOverlayType(db.PlayerWeaponType(weaponValue))
	if weaponType <= 0 {
		return nil
	}
	if secondLayer && !PlayerWeaponOverlaySupportsSecondLayer(weaponType) {
		return nil
	}
	token := PlayerWeaponOverlayToken(weaponType)
	if token == "" {
		return nil
	}
	name := normalizeNonPCSpriteResourceName(resourceName)
	lowerName := strings.ToLower(name)
	suffix := "." + strings.ToLower(extension)
	if strings.HasSuffix(lowerName, suffix) {
		name = name[:len(name)-len(suffix)]
	}
	if i := strings.LastIndex(name, "\\"); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		return nil
	}
	stem := fmt.Sprintf("%s_%s", name, token)
	if secondLayer {
		stem = fmt.Sprintf("%s_%s", stem, weaponLightSuffix)
	}
	return []string{
		fmt.Sprintf("%s%s.%s", mercenaryWeaponRoot, stem, extension),
	}
}

func mercenaryWeaponOverlayType(weaponType int) int {
	switch weaponType {
	case db.WeaponSword, db.WeaponTwoHandSword:
		return db.WeaponSword
	case db.WeaponSpear, db.WeaponTwoHandSpear:
		return db.WeaponSpear
	case db.WeaponBow:
		return db.WeaponBow
	default:
		return weaponType
	}
}

func (m *Manager) PlayerWeaponViewID(weaponValue int) int {
	return PlayerWeaponViewID(m, weaponValue)
}

func PlayerWeaponViewID(manager *Manager, weaponValue int) int {
	if weaponValue <= 0 {
		return 0
	}
	if weaponValue < db.MaxWeaponType {
		return weaponValue
	}
	if manager != nil {
		if classNum, ok := manager.ItemClassNum(weaponValue); ok {
			return classNum
		}
	}
	return db.PlayerWeaponType(weaponValue)
}

func NormalizePlayerWeaponShield(weapon, shield int) (int, int) {
	if weapon <= 0 && shield > 0 && db.PlayerWeaponType(shield) > 0 && PlayerShieldToken(shield) == "" {
		return shield, 0
	}
	return weapon, shield
}

func PlayerShieldOverlayResourceCandidates(job int, sex byte, shield int, extension string) []string {
	token := PlayerShieldToken(shield)
	if token == "" {
		return nil
	}
	sexToken := PlayerSexToken(sex)
	jobToken := PlayerJobToken(job)
	return []string{
		fmt.Sprintf("%s%s\\%s_%s_%s.%s", playerShieldRoot, jobToken, jobToken, sexToken, token, extension),
	}
}

func PlayerBodyPaletteResourceCandidates(job int, sex byte, palette int, extension string) []string {
	if palette <= 0 {
		return nil
	}
	sexToken := PlayerSexToken(sex)
	tokens := []string{PlayerJobToken(job)}
	if tokens[0] != PlayerJobToken(0) {
		tokens = append(tokens, PlayerJobToken(0))
	}
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		out = append(out, fmt.Sprintf("%s%s\\%s_%s_%d.%s", playerPaletteRoot, playerPaletteBodyDir, token, sexToken, palette, extension))
	}
	return out
}

func PlayerHeadPaletteResourceCandidates(job int, head int, sex byte, palette int, extension string) []string {
	if palette <= 0 {
		return nil
	}
	sexToken := PlayerSexToken(sex)
	head = NormalizePlayerHead(head, job)
	return []string{
		fmt.Sprintf("%s%s\\%s%d_%s_%d.%s", playerPaletteRoot, playerPaletteHeadDir, playerPaletteHeadFile, head, sexToken, palette, extension),
	}
}

func PlayerSexToken(sex byte) string {
	if sex != 0 {
		return playerMaleSex
	}
	return playerFemaleSex
}

func PlayerSexLabel(sex byte) string {
	if sex != 0 {
		return "male"
	}
	return "female"
}

func PlayerJobToken(job int) string {
	if token, ok := db.JobSpriteResourceName(job); ok {
		return token
	}
	token, _ := db.JobSpriteResourceName(0)
	return token
}

func HasPlayerJobToken(job int) bool {
	_, ok := db.JobSpriteResourceName(job)
	return ok
}

func NormalizePlayerHead(head int, job int) int {
	if head == 0 {
		switch job {
		case 1:
			return 2
		case 2:
			return 3
		case 3:
			return 4
		case 4:
			return 5
		case 5:
			return 6
		case 6:
			return 7
		default:
			return 1
		}
	}
	if head < 1 || head > 25 {
		return 13
	}
	return head
}

func PlayerWeaponOverlayTypeForJob(job int, weaponType int, dualWeapon bool) int {
	if weaponType <= 0 {
		if dualWeapon && IsDualWeaponPlayerJob(job) {
			return 2
		}
		return 0
	}
	if dualWeapon {
		if IsDualWeaponPlayerJob(job) && (weaponType == 7 || weaponType == 8) {
			return 2
		}
		return weaponType
	}
	switch {
	case (job == 6 || job == 4007 || job == 4029) && (weaponType == 6 || weaponType == 7 || weaponType == 8):
		return 2
	case (job == 12 || job == 17 || job == 4013 || job == 4018 || job == 4035 || job == 4040) && (weaponType == 7 || weaponType == 8):
		return 2
	case (job == 2 || job == 9 || job == 16 || job == 4003 || job == 4010 || job == 4017 || job == 4025 || job == 4032 || job == 4039) && weaponType == 5:
		return 10
	case (job == 17 || job == 4018 || job == 4040) && weaponType == 6:
		return 1
	case (job == 8 || job == 4009 || job == 4031) && weaponType == 12:
		return 0
	default:
		return weaponType
	}
}

func IsDualWeaponPlayerJob(job int) bool {
	switch job {
	case 12, 4013, 4035, 4060, 4079:
		return true
	default:
		return false
	}
}

func PlayerWeaponOverlaySupportsSecondLayer(weaponType int) bool {
	switch weaponType {
	case 1, 2, 3, 4, 5, 6, 7, 16, 17, 18, 25, 26, 27, 28, 29, 30:
		return true
	default:
		return false
	}
}

func PlayerWeaponOverlayToken(weaponType int) string {
	token, ok := db.WeaponResourceName[weaponType]
	if !ok {
		return ""
	}
	if len(token) > 0 && token[0] == '_' {
		return token[1:]
	}
	return token
}

func PlayerShieldToken(shield int) string {
	if shield <= 0 {
		return ""
	}
	viewID := shield
	if shield > 4 {
		switch shield {
		case 2101, 2102:
			viewID = 1
		case 2103, 2104:
			viewID = 2
		case 2105, 2106:
			viewID = 3
		case 2107, 2108, 2110, 2111:
			viewID = 4
		default:
			return ""
		}
	}
	return db.ShieldResourceName[viewID]
}
