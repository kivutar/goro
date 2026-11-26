package sprites

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func LoadPlayer(manager *res.Manager, character session.Character, sex byte, admin bool) (*Humanoid, string) {
	weapon, shield := res.NormalizePlayerWeaponShield(int(character.Weapon), int(character.Shield))
	return LoadHumanoid(manager, Appearance{
		Job:         int(character.Job),
		Head:        int(character.Hair),
		Sex:         sex,
		Admin:       admin,
		BodyPalette: int(character.BodyPal),
		HeadPalette: characterHeadPalette(character),
		Weapon:      weapon,
		Shield:      shield,
		HeadTop:     int(character.HeadTop),
		HeadMid:     int(character.HeadMid),
		HeadLow:     int(character.HeadLow),
	}, "player")
}

func LoadNonPC(manager *res.Manager, job int, label string) (*scene.SpriteView, string) {
	if JobHasNoSprite(job) {
		return nil, fmt.Sprintf("%s job=%d no-sprite", label, job)
	}
	resourceName, ok := manager.NonPCResourceName(job)
	if !ok {
		return nil, fmt.Sprintf("%s job=%d resource-name=missing", label, job)
	}
	if isGR2Resource(resourceName) {
		return nil, fmt.Sprintf("%s job=%d resource=%s gr2-model=not-sprite", label, job, resourceName)
	}
	view, status := scene.LoadSpriteView(manager, res.NonPCSpriteResourceCandidates(job, resourceName, "act"), res.NonPCSpriteResourceCandidates(job, resourceName, "spr"), nil, label+" "+resourceName)
	if view == nil {
		return nil, status
	}
	if upgrade, ok := loadRicherNonPCSpritePair(manager, job, resourceName, len(view.Act.Actions)); ok {
		view.Act = upgrade.act
		view.ActSource = upgrade.actSource
		view.Spr = upgrade.spr
		view.Source = upgrade.sprSource
		status += fmt.Sprintf(" sprite-upgraded act=%s spr=%s actions=%d frames=%d", upgrade.actSource, upgrade.sprSource, len(upgrade.act.Actions), len(upgrade.spr.Frames))
	}
	return view, status
}

func JobHasNoSprite(job int) bool {
	if job == scene.ActorJobWarpPortal || job == scene.ActorJobWarpPortalActive || job == scene.ActorJobWarpPortalWaiting {
		return true
	}
	return job == scene.ActorJobHiddenNPC || job == scene.ActorJobHiddenWarpNPC || job == scene.ActorJobClearNPC
}

func isGR2Resource(resourceName string) bool {
	return res.IsGR2ResourceName(resourceName)
}

func characterHeadPalette(character session.Character) int {
	if character.HeadPal > 0 {
		return int(character.HeadPal)
	}
	return int(character.HairColor)
}

func LoadHumanoid(manager *res.Manager, appearance Appearance, label string) (*Humanoid, string) {
	body, bodyStatus := loadBodySpriteView(manager, appearance.Job, appearance.Sex, appearance.BodyPalette, appearance.Admin, label+" body")
	if body == nil {
		return nil, bodyStatus
	}
	headView, headStatus := loadHeadSpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadPalette, label+" head")
	imf, imfSource, imfStatus := loadPlayerIMF(manager, appearance.Job, appearance.Sex)
	accessoryBottom, accessoryBottomStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadLow, label+" accessory-bottom")
	accessoryMid, accessoryMidStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadMid, label+" accessory-mid")
	accessoryTop, accessoryTopStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadTop, label+" accessory-top")
	weapon, weaponStatus := loadWeaponOverlaySpriteView(manager, appearance.Job, appearance.Sex, appearance.Weapon, false, label+" weapon")
	weaponLight, weaponLightStatus := loadWeaponOverlaySpriteView(manager, appearance.Job, appearance.Sex, appearance.Weapon, true, label+" weapon-light")
	shield, shieldStatus := loadShieldOverlaySpriteView(manager, appearance.Job, appearance.Sex, appearance.Shield, label+" shield")
	view := &Humanoid{
		Body:            body,
		head:            headView,
		accessoryBottom: accessoryBottom,
		accessoryMid:    accessoryMid,
		accessoryTop:    accessoryTop,
		weapon:          weapon,
		weaponLight:     weaponLight,
		shield:          shield,
		imf:             imf,
		imfSource:       imfSource,
		billboards:      make(map[humanoidBillboardKey]*scene.Billboard),
		Started:         time.Now(),
	}
	status := bodyStatus + " " + headStatus + imfStatus
	for _, overlayStatus := range []string{accessoryBottomStatus, accessoryMidStatus, accessoryTopStatus, weaponStatus, weaponLightStatus, shieldStatus} {
		if overlayStatus != "" {
			status += " " + overlayStatus
		}
	}
	return view, status
}

func LoadMercenary(manager *res.Manager, appearance Appearance, label string) (*Humanoid, string) {
	appearance.Sex = companion.MercenarySexForJob(appearance.Job, appearance.Sex)
	appearance.Weapon = companion.MercenaryWeaponForAppearance(appearance.Job, appearance.Weapon)
	resourceName := ""
	if manager != nil {
		resourceName, _ = manager.NonPCResourceName(appearance.Job)
	}
	body, bodyStatus := LoadNonPC(manager, appearance.Job, label+" body")
	if body == nil {
		return nil, bodyStatus
	}
	headView, headStatus := loadHeadSpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadPalette, label+" head")
	accessoryBottom, accessoryBottomStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadLow, label+" accessory-bottom")
	accessoryMid, accessoryMidStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadMid, label+" accessory-mid")
	accessoryTop, accessoryTopStatus := loadAccessorySpriteView(manager, appearance.Job, appearance.Head, appearance.Sex, appearance.HeadTop, label+" accessory-top")
	weapon, weaponStatus := loadMercenaryWeaponOverlaySpriteView(manager, resourceName, appearance.Job, appearance.Sex, appearance.Weapon, false, label+" weapon")
	weaponLight, weaponLightStatus := loadMercenaryWeaponOverlaySpriteView(manager, resourceName, appearance.Job, appearance.Sex, appearance.Weapon, true, label+" weapon-light")
	shield, shieldStatus := loadShieldOverlaySpriteView(manager, appearance.Job, appearance.Sex, appearance.Shield, label+" shield")
	view := &Humanoid{
		Body:            body,
		head:            headView,
		accessoryBottom: accessoryBottom,
		accessoryMid:    accessoryMid,
		accessoryTop:    accessoryTop,
		weapon:          weapon,
		weaponLight:     weaponLight,
		shield:          shield,
		billboards:      make(map[humanoidBillboardKey]*scene.Billboard),
		Started:         time.Now(),
	}
	status := bodyStatus + " " + headStatus
	for _, overlayStatus := range []string{accessoryBottomStatus, accessoryMidStatus, accessoryTopStatus, weaponStatus, weaponLightStatus, shieldStatus} {
		if overlayStatus != "" {
			status += " " + overlayStatus
		}
	}
	return view, status
}

func loadBodySpriteView(manager *res.Manager, job int, sex byte, palette int, admin bool, label string) (*scene.SpriteView, string) {
	actCandidates := res.PlayerBodyResourceCandidates(job, sex, "act")
	sprCandidates := res.PlayerBodyResourceCandidates(job, sex, "spr")
	if admin {
		actCandidates = append(res.PlayerAdminBodyResourceCandidates(sex, "act"), actCandidates...)
		sprCandidates = append(res.PlayerAdminBodyResourceCandidates(sex, "spr"), sprCandidates...)
	}
	return scene.LoadSpriteView(manager, actCandidates, sprCandidates, res.PlayerBodyPaletteResourceCandidates(job, sex, palette, "pal"), label)
}

func loadHeadSpriteView(manager *res.Manager, job int, head int, sex byte, palette int, label string) (*scene.SpriteView, string) {
	return scene.LoadSpriteView(manager, res.PlayerHeadResourceCandidates(job, head, sex, "act"), res.PlayerHeadResourceCandidates(job, head, sex, "spr"), res.PlayerHeadPaletteResourceCandidates(job, head, sex, palette, "pal"), label)
}

func loadAccessorySpriteView(manager *res.Manager, job int, head int, sex byte, viewID int, label string) (*scene.SpriteView, string) {
	if viewID <= 0 {
		return nil, ""
	}
	resourceName, _ := manager.AccessoryResourceName(viewID)
	if viewID != 185 && resourceName == "" {
		return nil, fmt.Sprintf("%s skipped: missing accessory resource table", label)
	}
	return scene.LoadSpriteView(manager, res.PlayerAccessoryResourceCandidates(job, head, sex, viewID, resourceName, "act"), res.PlayerAccessoryResourceCandidates(job, head, sex, viewID, resourceName, "spr"), nil, label)
}

func loadWeaponOverlaySpriteView(manager *res.Manager, job int, sex byte, weapon int, secondLayer bool, label string) (*scene.SpriteView, string) {
	if weapon <= 0 {
		return nil, ""
	}
	viewID := weapon
	if manager != nil {
		viewID = manager.PlayerWeaponViewID(weapon)
	}
	return scene.LoadSpriteView(manager, res.PlayerWeaponOverlayResourceCandidatesForItem(job, sex, weapon, viewID, secondLayer, "act"), res.PlayerWeaponOverlayResourceCandidatesForItem(job, sex, weapon, viewID, secondLayer, "spr"), nil, label)
}

func loadMercenaryWeaponOverlaySpriteView(manager *res.Manager, resourceName string, job int, sex byte, weapon int, secondLayer bool, label string) (*scene.SpriteView, string) {
	if weapon <= 0 {
		return nil, ""
	}
	viewID := weapon
	if manager != nil {
		viewID = manager.PlayerWeaponViewID(weapon)
	}
	actCandidates := res.MercenaryWeaponOverlayResourceCandidates(resourceName, viewID, secondLayer, "act")
	sprCandidates := res.MercenaryWeaponOverlayResourceCandidates(resourceName, viewID, secondLayer, "spr")
	if len(actCandidates) > 0 && len(sprCandidates) > 0 {
		if view, status := scene.LoadSpriteView(manager, actCandidates, sprCandidates, nil, label); view != nil {
			return view, status
		}
	}
	return loadWeaponOverlaySpriteView(manager, companion.MercenaryWeaponBaseJob(job), sex, viewID, secondLayer, label)
}

func loadShieldOverlaySpriteView(manager *res.Manager, job int, sex byte, shield int, label string) (*scene.SpriteView, string) {
	if shield <= 0 {
		return nil, ""
	}
	return scene.LoadSpriteView(manager, res.PlayerShieldOverlayResourceCandidates(job, sex, shield, "act"), res.PlayerShieldOverlayResourceCandidates(job, sex, shield, "spr"), nil, label)
}

func LoadShadow(manager *res.Manager) (*scene.SpriteView, string) {
	return scene.LoadSpriteView(manager,
		[]string{"data\\sprite\\shadow.act", "data/sprite/shadow.act"},
		[]string{"data\\sprite\\shadow.spr", "data/sprite/shadow.spr"},
		nil,
		"actor shadow",
	)
}

func LoadCart(manager *res.Manager, cartNum int) (*scene.SpriteView, string) {
	return scene.LoadSpriteView(manager,
		res.PlayerCartResourceCandidates(cartNum, "act"),
		res.PlayerCartResourceCandidates(cartNum, "spr"),
		nil,
		fmt.Sprintf("cart %d", cartNum),
	)
}

func LoadFalcon(manager *res.Manager, job int) (*scene.SpriteView, string) {
	return scene.LoadSpriteView(manager,
		res.PlayerFalconResourceCandidates(job, "act"),
		res.PlayerFalconResourceCandidates(job, "spr"),
		nil,
		fmt.Sprintf("falcon job=%d", job),
	)
}

func LoadCursor(manager *res.Manager) (*scene.SpriteView, string) {
	if manager == nil {
		return nil, "no resource manager"
	}
	return scene.LoadSpriteView(manager,
		[]string{"data\\sprite\\cursors.act", "data/sprite/cursors.act", "data\\sprite\\interface\\cursors.act", "data/sprite/interface/cursors.act"},
		[]string{"data\\sprite\\cursors.spr", "data/sprite/cursors.spr", "data\\sprite\\interface\\cursors.spr", "data/sprite/interface/cursors.spr"},
		nil,
		"cursor",
	)
}

func loadPlayerIMF(manager *res.Manager, job int, sex byte) (*res.IMF, string, string) {
	imf, source, err := manager.LoadIMF(res.PlayerIMFResourceCandidates(job, sex))
	if err != nil {
		if source == "" {
			return nil, "", " imf=missing"
		}
		return nil, "", fmt.Sprintf(" imf=%s parse-error=%v", source, err)
	}
	return imf, source, fmt.Sprintf(" imf=%s", source)
}

func LoadPetAccessory(manager *res.Manager, base *scene.SpriteView, accessoryID uint32, label string) (*scene.SpriteView, string) {
	if manager == nil || base == nil {
		return nil, fmt.Sprintf("%s skipped: missing base sprite", label)
	}
	path, ok := db.PetActionPath(int(accessoryID))
	if !ok {
		return nil, fmt.Sprintf("%s skipped: missing pet action path accessory=%d", label, accessoryID)
	}
	candidates := petActionResourceCandidates(path)
	act, actSource, err := manager.LoadACT(candidates)
	if err != nil {
		return nil, fmt.Sprintf("%s act: %v", label, err)
	}
	if !actFitsSPR(act, base.Spr) {
		return nil, fmt.Sprintf("%s act %s incompatible with %s", label, actSource, base.Source)
	}
	return &scene.SpriteView{
		Spr:           base.Spr,
		Act:           act,
		ActSource:     actSource,
		Source:        base.Source,
		Palette:       base.Palette,
		PaletteSource: base.PaletteSource,
		Images:        base.Images,
		Billboards:    make(map[scene.SpriteBillboardKey]*scene.Billboard),
		Started:       time.Now(),
	}, fmt.Sprintf("%s: %s actions=%d frames=%d base=%s", label, actSource, len(act.Actions), len(base.Spr.Frames), base.Source)
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
			act, err := manager.LoadArchiveACT(archive, candidate)
			if err != nil {
				continue
			}
			if !preferNonPCActUpgrade(job, currentActions, len(act.Actions)) {
				continue
			}
			for _, sprCandidate := range res.NonPCSpriteResourceCandidates(job, resourceName, "spr") {
				spr, err := manager.LoadArchiveSPR(archive, sprCandidate)
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

func MercenaryAppearanceForActor(actor worldstate.Actor) Appearance {
	return Appearance{
		Job:         int(actor.Job),
		Head:        int(actor.Head),
		Sex:         companion.MercenarySexForJob(int(actor.Job), actor.Sex),
		BodyPalette: int(actor.BodyPal),
		HeadPalette: int(actor.HeadPal),
		Weapon:      companion.MercenaryWeaponForAppearance(int(actor.Job), int(actor.Weapon)),
		Shield:      int(actor.Shield),
		HeadTop:     int(actor.HeadTop),
		HeadMid:     int(actor.HeadMid),
		HeadLow:     int(actor.HeadLow),
	}
}
