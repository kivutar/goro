package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
)

func loadPlayerHumanoidSpriteView(manager *res.Manager, character session.Character, sex byte) (*humanoidSpriteView, string) {
	weapon, shield := res.NormalizePlayerWeaponShield(int(character.Weapon), int(character.Shield))
	return loadHumanoidSpriteViewWithAppearance(manager, humanoidAppearance{
		job:         int(character.Job),
		head:        int(character.Hair),
		sex:         sex,
		bodyPalette: int(character.BodyPal),
		headPalette: characterHeadPalette(character),
		weapon:      weapon,
		shield:      shield,
		headTop:     int(character.HeadTop),
		headMid:     int(character.HeadMid),
		headLow:     int(character.HeadLow),
	}, "player")
}

func loadNonPCSpriteView(manager *res.Manager, job int, label string) (*spriteView, string) {
	if actorJobHasNoSprite(job) {
		return nil, fmt.Sprintf("%s job=%d no-sprite", label, job)
	}
	resourceName, ok := manager.NonPCResourceName(job)
	if !ok {
		return nil, fmt.Sprintf("%s job=%d resource-name=missing", label, job)
	}
	if isGR2Resource(resourceName) {
		return nil, fmt.Sprintf("%s job=%d resource=%s gr2-model=unsupported", label, job, resourceName)
	}
	view, status := loadSpriteView(manager, res.NonPCSpriteResourceCandidates(job, resourceName, "act"), res.NonPCSpriteResourceCandidates(job, resourceName, "spr"), nil, label+" "+resourceName)
	if view == nil {
		return nil, status
	}
	if upgrade, ok := loadRicherNonPCSpritePair(manager, job, resourceName, len(view.act.Actions)); ok {
		view.act = upgrade.act
		view.actSource = upgrade.actSource
		view.spr = upgrade.spr
		view.source = upgrade.sprSource
		status += fmt.Sprintf(" sprite-upgraded act=%s spr=%s actions=%d frames=%d", upgrade.actSource, upgrade.sprSource, len(upgrade.act.Actions), len(upgrade.spr.Frames))
	}
	return view, status
}

func actorJobHasNoSprite(job int) bool {
	switch job {
	case actorJobWarpPortal, actorJobHiddenNPC, actorJobClearNPC:
		return true
	default:
		return false
	}
}

func isGR2Resource(resourceName string) bool {
	name := strings.ToLower(strings.ReplaceAll(resourceName, "/", "\\"))
	if i := strings.LastIndex(name, "\\"); i >= 0 {
		name = name[i+1:]
	}
	return strings.HasSuffix(name, ".gr2")
}

func characterHeadPalette(character session.Character) int {
	if character.HeadPal > 0 {
		return int(character.HeadPal)
	}
	return int(character.HairColor)
}

func loadHumanoidSpriteView(manager *res.Manager, job int, head int, sex byte, bodyPalette int, headPalette int, label string) (*humanoidSpriteView, string) {
	return loadHumanoidSpriteViewWithAppearance(manager, humanoidAppearance{
		job:         job,
		head:        head,
		sex:         sex,
		bodyPalette: bodyPalette,
		headPalette: headPalette,
	}, label)
}

func loadHumanoidSpriteViewWithAppearance(manager *res.Manager, appearance humanoidAppearance, label string) (*humanoidSpriteView, string) {
	body, bodyStatus := loadBodySpriteView(manager, appearance.job, appearance.sex, appearance.bodyPalette, label+" body")
	if body == nil {
		return nil, bodyStatus
	}
	headView, headStatus := loadHeadSpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headPalette, label+" head")
	imf, imfSource, imfStatus := loadPlayerIMF(manager, appearance.job, appearance.sex)
	accessoryBottom, accessoryBottomStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headLow, "", label+" accessory-bottom")
	accessoryMid, accessoryMidStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headMid, "", label+" accessory-mid")
	accessoryTop, accessoryTopStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headTop, "", label+" accessory-top")
	weapon, weaponStatus := loadWeaponOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.weapon, false, label+" weapon")
	weaponLight, weaponLightStatus := loadWeaponOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.weapon, true, label+" weapon-light")
	shield, shieldStatus := loadShieldOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.shield, label+" shield")
	view := &humanoidSpriteView{
		body:            body,
		head:            headView,
		accessoryBottom: accessoryBottom,
		accessoryMid:    accessoryMid,
		accessoryTop:    accessoryTop,
		weapon:          weapon,
		weaponLight:     weaponLight,
		shield:          shield,
		imf:             imf,
		imfSource:       imfSource,
		billboards:      make(map[humanoidBillboardKey]*spriteBillboard),
		started:         time.Now(),
	}
	status := bodyStatus + " " + headStatus + imfStatus
	for _, overlayStatus := range []string{accessoryBottomStatus, accessoryMidStatus, accessoryTopStatus, weaponStatus, weaponLightStatus, shieldStatus} {
		if overlayStatus != "" {
			status += " " + overlayStatus
		}
	}
	return view, status
}

func loadMercenaryHumanoidSpriteView(manager *res.Manager, appearance humanoidAppearance, label string) (*humanoidSpriteView, string) {
	appearance.sex = mercenarySexForJob(appearance.job, appearance.sex)
	body, bodyStatus := loadNonPCSpriteView(manager, appearance.job, label+" body")
	if body == nil {
		return nil, bodyStatus
	}
	headView, headStatus := loadHeadSpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headPalette, label+" head")
	accessoryBottom, accessoryBottomStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headLow, "", label+" accessory-bottom")
	accessoryMid, accessoryMidStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headMid, "", label+" accessory-mid")
	accessoryTop, accessoryTopStatus := loadAccessorySpriteView(manager, appearance.job, appearance.head, appearance.sex, appearance.headTop, "", label+" accessory-top")
	weapon, weaponStatus := loadWeaponOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.weapon, false, label+" weapon")
	weaponLight, weaponLightStatus := loadWeaponOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.weapon, true, label+" weapon-light")
	shield, shieldStatus := loadShieldOverlaySpriteView(manager, appearance.job, appearance.sex, appearance.shield, label+" shield")
	view := &humanoidSpriteView{
		body:            body,
		head:            headView,
		accessoryBottom: accessoryBottom,
		accessoryMid:    accessoryMid,
		accessoryTop:    accessoryTop,
		weapon:          weapon,
		weaponLight:     weaponLight,
		shield:          shield,
		billboards:      make(map[humanoidBillboardKey]*spriteBillboard),
		started:         time.Now(),
	}
	status := bodyStatus + " " + headStatus
	for _, overlayStatus := range []string{accessoryBottomStatus, accessoryMidStatus, accessoryTopStatus, weaponStatus, weaponLightStatus, shieldStatus} {
		if overlayStatus != "" {
			status += " " + overlayStatus
		}
	}
	return view, status
}

func loadBodySpriteView(manager *res.Manager, job int, sex byte, palette int, label string) (*spriteView, string) {
	return loadSpriteView(manager, res.PlayerBodyResourceCandidates(job, sex, "act"), res.PlayerBodyResourceCandidates(job, sex, "spr"), res.PlayerBodyPaletteResourceCandidates(job, sex, palette, "pal"), label)
}

func loadHeadSpriteView(manager *res.Manager, job int, head int, sex byte, palette int, label string) (*spriteView, string) {
	return loadSpriteView(manager, res.PlayerHeadResourceCandidates(job, head, sex, "act"), res.PlayerHeadResourceCandidates(job, head, sex, "spr"), res.PlayerHeadPaletteResourceCandidates(job, head, sex, palette, "pal"), label)
}

func loadAccessorySpriteView(manager *res.Manager, job int, head int, sex byte, viewID int, resourceName string, label string) (*spriteView, string) {
	if viewID <= 0 {
		return nil, ""
	}
	if resourceName == "" {
		if name, ok := manager.AccessoryResourceName(viewID); ok {
			resourceName = name
		}
	}
	if viewID != 185 && resourceName == "" {
		return nil, fmt.Sprintf("%s skipped: missing accessory resource table", label)
	}
	return loadSpriteView(manager, res.PlayerAccessoryResourceCandidates(job, head, sex, viewID, resourceName, "act"), res.PlayerAccessoryResourceCandidates(job, head, sex, viewID, resourceName, "spr"), nil, label)
}

func loadWeaponOverlaySpriteView(manager *res.Manager, job int, sex byte, weapon int, secondLayer bool, label string) (*spriteView, string) {
	if weapon <= 0 {
		return nil, ""
	}
	if manager != nil {
		weapon = manager.PlayerWeaponViewID(weapon)
	}
	return loadSpriteView(manager, res.PlayerWeaponOverlayResourceCandidates(job, sex, weapon, secondLayer, "act"), res.PlayerWeaponOverlayResourceCandidates(job, sex, weapon, secondLayer, "spr"), nil, label)
}

func loadShieldOverlaySpriteView(manager *res.Manager, job int, sex byte, shield int, label string) (*spriteView, string) {
	if shield <= 0 {
		return nil, ""
	}
	return loadSpriteView(manager, res.PlayerShieldOverlayResourceCandidates(job, sex, shield, "act"), res.PlayerShieldOverlayResourceCandidates(job, sex, shield, "spr"), nil, label)
}

func loadActorShadowSpriteView(manager *res.Manager) (*spriteView, string) {
	return loadSpriteView(manager,
		[]string{"data\\sprite\\shadow.act", "data/sprite/shadow.act"},
		[]string{"data\\sprite\\shadow.spr", "data/sprite/shadow.spr"},
		nil,
		"actor shadow",
	)
}

func loadCartSpriteView(manager *res.Manager, cartNum int) (*spriteView, string) {
	return loadSpriteView(manager,
		res.PlayerCartResourceCandidates(cartNum, "act"),
		res.PlayerCartResourceCandidates(cartNum, "spr"),
		nil,
		fmt.Sprintf("cart %d", cartNum),
	)
}

func loadCursorSpriteView(manager *res.Manager) (*spriteView, string) {
	return loadSpriteView(manager,
		[]string{"data\\sprite\\cursors.act", "data/sprite/cursors.act", "data\\sprite\\interface\\cursors.act", "data/sprite/interface/cursors.act"},
		[]string{"data\\sprite\\cursors.spr", "data/sprite/cursors.spr", "data\\sprite\\interface\\cursors.spr", "data/sprite/interface/cursors.spr"},
		nil,
		"cursor",
	)
}

func loadPlayerIMF(manager *res.Manager, job int, sex byte) (*res.IMF, string, string) {
	data, source, err := readFirstResource(manager, res.PlayerIMFResourceCandidates(job, sex))
	if err != nil {
		return nil, "", " imf=missing"
	}
	imf, err := res.ParseIMF(data)
	if err != nil {
		return nil, "", fmt.Sprintf(" imf=%s parse-error=%v", source, err)
	}
	return imf, source, fmt.Sprintf(" imf=%s", source)
}

func loadSpriteView(manager *res.Manager, actCandidates []string, sprCandidates []string, palCandidates []string, label string) (*spriteView, string) {
	actData, actSource, err := readFirstResource(manager, actCandidates)
	if err != nil {
		return nil, fmt.Sprintf("%s act: %v", label, err)
	}
	sprData, sprSource, err := readFirstResource(manager, sprCandidates)
	if err != nil {
		return nil, fmt.Sprintf("%s spr: %v", label, err)
	}
	act, err := res.ParseACT(actData)
	if err != nil {
		return nil, fmt.Sprintf("%s act parse %s: %v", label, actSource, err)
	}
	spr, err := res.ParseSPR(sprData)
	if err != nil {
		return nil, fmt.Sprintf("%s spr parse %s: %v", label, sprSource, err)
	}
	palette, paletteSource, paletteStatus := loadSpritePalette(manager, palCandidates)
	return &spriteView{
		spr:           spr,
		act:           act,
		actSource:     actSource,
		source:        sprSource,
		palette:       palette,
		paletteSource: paletteSource,
		images:        make(map[spriteFrameKey]*render.Image),
		billboards:    make(map[singleSpriteBillboardKey]*spriteBillboard),
		started:       time.Now(),
	}, fmt.Sprintf("%s: %s actions=%d frames=%d%s", label, sprSource, len(act.Actions), len(spr.Frames), paletteStatus)
}

func loadPetAccessorySpriteView(manager *res.Manager, base *spriteView, accessoryID uint32, label string) (*spriteView, string) {
	if manager == nil || base == nil {
		return nil, fmt.Sprintf("%s skipped: missing base sprite", label)
	}
	path, ok := db.PetActionPath(int(accessoryID))
	if !ok {
		return nil, fmt.Sprintf("%s skipped: missing pet action path accessory=%d", label, accessoryID)
	}
	candidates := petActionResourceCandidates(path)
	actData, actSource, err := readFirstResource(manager, candidates)
	if err != nil {
		return nil, fmt.Sprintf("%s act: %v", label, err)
	}
	act, err := res.ParseACT(actData)
	if err != nil {
		return nil, fmt.Sprintf("%s act parse %s: %v", label, actSource, err)
	}
	if !actFitsSPR(act, base.spr) {
		return nil, fmt.Sprintf("%s act %s incompatible with %s", label, actSource, base.source)
	}
	return &spriteView{
		spr:           base.spr,
		act:           act,
		actSource:     actSource,
		source:        base.source,
		palette:       base.palette,
		paletteSource: base.paletteSource,
		images:        base.images,
		billboards:    make(map[singleSpriteBillboardKey]*spriteBillboard),
		started:       time.Now(),
	}, fmt.Sprintf("%s: %s actions=%d frames=%d base=%s", label, actSource, len(act.Actions), len(base.spr.Frames), base.source)
}

func petActionResourceCandidates(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	path = strings.TrimPrefix(path, "data/sprite/")
	path = strings.TrimPrefix(path, "data\\sprite\\")
	backslash := strings.ReplaceAll(path, "/", "\\")
	slash := strings.ReplaceAll(path, "\\", "/")
	return []string{
		"data\\sprite\\" + backslash,
		"data/sprite/" + slash,
	}
}

type nonPCSpritePairUpgrade struct {
	act       *res.ACT
	actSource string
	spr       *res.SPR
	sprSource string
}

func loadRicherNonPCSpritePair(manager *res.Manager, job int, resourceName string, currentActions int) (nonPCSpritePairUpgrade, bool) {
	if manager == nil || job < 1000 || currentActions > 8 {
		return nonPCSpritePairUpgrade{}, false
	}
	var best nonPCSpritePairUpgrade
	for _, archive := range manager.Archives {
		for _, candidate := range res.NonPCSpriteResourceCandidates(job, resourceName, "act") {
			data, err := archive.ReadFile(candidate)
			if err != nil {
				continue
			}
			act, err := res.ParseACT(data)
			if err != nil {
				continue
			}
			if !preferNonPCActUpgrade(job, currentActions, len(act.Actions)) {
				continue
			}
			for _, sprCandidate := range res.NonPCSpriteResourceCandidates(job, resourceName, "spr") {
				sprData, err := archive.ReadFile(sprCandidate)
				if err != nil {
					continue
				}
				spr, err := res.ParseSPR(sprData)
				if err != nil || !actFitsSPR(act, spr) {
					continue
				}
				if best.act == nil || len(act.Actions) > len(best.act.Actions) || len(act.Actions) == len(best.act.Actions) && len(spr.Frames) > len(best.spr.Frames) {
					best = nonPCSpritePairUpgrade{
						act:       act,
						actSource: archive.Path() + ":" + candidate,
						spr:       spr,
						sprSource: archive.Path() + ":" + sprCandidate,
					}
				}
			}
		}
	}
	return best, best.act != nil
}

func preferNonPCActUpgrade(job int, currentActions, candidateActions int) bool {
	return job >= 1000 && currentActions > 0 && currentActions <= 8 && candidateActions >= 40
}

func actFitsSPR(act *res.ACT, spr *res.SPR) bool {
	if act == nil || spr == nil {
		return false
	}
	for _, action := range act.Actions {
		for _, anim := range action.Animations {
			for _, layer := range anim.Layers {
				if layer.Index < 0 {
					continue
				}
				index := int(layer.Index)
				if layer.SPRType == res.SPRFrameRGBA {
					index += spr.RGBAIndex
				}
				if index < 0 || index >= len(spr.Frames) {
					return false
				}
			}
		}
	}
	return true
}

func loadSpritePalette(manager *res.Manager, candidates []string) (*res.Palette, string, string) {
	if len(candidates) == 0 {
		return nil, "", ""
	}
	data, source, err := readFirstResource(manager, candidates)
	if err != nil {
		return nil, "", " palette=default"
	}
	palette, err := res.ParsePAL(data)
	if err != nil {
		return nil, "", fmt.Sprintf(" palette=%s parse-error=%v", source, err)
	}
	return &palette, source, fmt.Sprintf(" palette=%s", source)
}

func readFirstResource(manager *res.Manager, candidates []string) ([]byte, string, error) {
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("not found")
}
