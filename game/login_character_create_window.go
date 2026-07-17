package game

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const (
	charCreateNameMinBytes = 4
	charCreateNameMaxBytes = 23
	charCreateMinHairStyle = 2
	charCreateMaxHairStyle = 23
)

type charCreateState struct {
	slot            int
	name            string
	stats           [6]uint8
	hairStyle       int
	hairColor       int
	direction       int
	preview         *humanoidSpriteView
	previewKey      charCreatePreviewKey
	previewFailed   bool
	previewImage    image.Image
	previewImageKey charCreatePreviewKey
}

type charCreatePreviewKey struct {
	sex       byte
	hairStyle int
	hairColor int
}

const (
	createStatStr = iota
	createStatAgi
	createStatVit
	createStatInt
	createStatDex
	createStatLuk
	createStatCount
)

func (m *LoginMode) updateCharacterCreateInput(ctx client.Context) {
	m.showCharacterCreateWindow(ctx)
	if m.charCreateWindow != nil {
		m.charCreateWindow.Update(ctx)
		m.charCreateWindow.Publish(ctx)
	}
}

func (m *LoginMode) openCharacterCreate(ctx client.Context, slot int, now time.Time) {
	slot = clampCharacterSlot(slot, m.maxSlots)
	if _, occupied := characterBySlot(ctx.Session.Characters, slot); occupied {
		empty, ok := firstEmptyCharacterSlot(ctx.Session.Characters, m.maxSlots)
		if !ok {
			m.status = "no empty character slots"
			return
		}
		slot = empty
	}
	m.create = defaultCharCreateState(slot)
	m.charCreateWindow = nil
	m.status = "create a character"
	m.playConfirmSFX(ctx)
	m.startPhaseFade(loginPhaseCreate, now)
}

func defaultCharCreateState(slot int) charCreateState {
	return charCreateState{
		slot:      slot,
		stats:     [6]uint8{5, 5, 5, 5, 5, 5},
		hairStyle: 2,
		hairColor: 0,
		direction: charSelectPreviewDirection,
	}
}

func (m *LoginMode) cancelCharacterCreate(now time.Time) {
	m.create = charCreateState{}
	m.charCreateWindow = nil
	m.status = "select a character"
	m.startPhaseFade(loginPhaseCharacter, now)
}

func (m *LoginMode) submitCharacterCreate(ctx client.Context) {
	name := strings.TrimSpace(m.create.name)
	if name == "" {
		m.status = "enter a character name"
		return
	}
	if len([]byte(name)) < charCreateNameMinBytes {
		m.status = "name must be at least 4 characters"
		return
	}
	packet := network.MakeCharacter{
		Name:      name,
		Str:       m.create.stats[createStatStr],
		Agi:       m.create.stats[createStatAgi],
		Vit:       m.create.stats[createStatVit],
		Int:       m.create.stats[createStatInt],
		Dex:       m.create.stats[createStatDex],
		Luk:       m.create.stats[createStatLuk],
		Slot:      uint8(m.create.slot),
		HairColor: uint16(m.create.hairColor),
		HairStyle: uint16(m.create.hairStyle),
	}
	if err := ctx.Network.SendMakeCharacter(packet); err != nil {
		m.status = "create character failed: " + err.Error()
		return
	}
	m.playConfirmSFX(ctx)
	m.status = "creating character..."
}

func (m *LoginMode) changeCreateHairStyle(delta int) {
	m.create.hairStyle += delta
	if m.create.hairStyle < charCreateMinHairStyle {
		m.create.hairStyle = charCreateMaxHairStyle
	}
	if m.create.hairStyle > charCreateMaxHairStyle {
		m.create.hairStyle = charCreateMinHairStyle
	}
	m.create.preview = nil
	m.create.previewFailed = false
	m.create.previewImage = nil
}

func (m *LoginMode) changeCreateHairColor() {
	m.create.hairColor = (m.create.hairColor + 1) % 10
	m.create.preview = nil
	m.create.previewFailed = false
	m.create.previewImage = nil
}

func (m *LoginMode) drawCharacterCreate(ctx client.Context) {
	m.showCharacterCreateWindow(ctx)
}

func (m *LoginMode) showCharacterCreateWindow(ctx client.Context) {
	m.updateCharacterCreateWindow(ctx)
	if m.charCreateWindow != nil {
		m.charCreateWindow.Publish(ctx)
	}
}

func (m *LoginMode) updateCharacterCreateWindow(ctx client.Context) {
	opts := gameui.CharacterCreateWindowOptions{
		Name:    m.create.name,
		Stats:   m.create.stats,
		Preview: m.characterCreatePreviewImage(ctx),
	}
	callbacks := gameui.CharacterCreateWindowCallbacks{
		OnNameChange: func(value string) {
			m.create.name = appendCharacterNameInput("", value, charCreateNameMaxBytes)
		},
		OnSubmit: func() {
			m.submitCharacterCreate(ctx)
		},
		OnCancel: func() {
			m.cancelCharacterCreate(time.Now())
		},
		OnHairPrev: func() {
			m.changeCreateHairStyle(-1)
		},
		OnHairNext: func() {
			m.changeCreateHairStyle(1)
		},
		OnHairColor: func() {
			m.changeCreateHairColor()
		},
		OnStat: func(stat int) {
			if !bumpCreateStat(&m.create.stats, stat) {
				m.status = "stat limit reached"
			}
		},
	}
	if m.charCreateWindow == nil {
		m.charCreateWindow = gameui.NewCharacterCreateWindow(ctx, opts, callbacks)
		return
	}
	m.charCreateWindow.SetOptions(ctx, opts)
}

func (m *LoginMode) characterCreatePreviewImage(ctx client.Context) image.Image {
	const previewW, previewH = 142, 166
	key := charCreatePreviewKey{sex: ctx.Session.Sex, hairStyle: m.create.hairStyle, hairColor: m.create.hairColor}
	if m.create.previewImage != nil && m.create.previewImageKey == key {
		return m.create.previewImage
	}
	img := render.NewImage(previewW, previewH)
	view := m.characterCreatePreviewView(ctx)
	if view == nil {
		m.create.previewImage = img.RGBA()
		m.create.previewImageKey = key
		return m.create.previewImage
	}
	billboard, ok := humanoidBillboardForState(view, spriteState{
		actionFamily: spriteActionIdle,
		direction:    m.create.direction,
		started:      time.Now(),
		loopIdle:     true,
	}, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		m.create.previewImage = img.RGBA()
		m.create.previewImageKey = key
		return m.create.previewImage
	}
	const scale = 1.08
	bounds := billboard.image.Bounds()
	var opts render.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(
		float64(previewW)/2-float64(bounds.Dx())*scale/2,
		float64(previewH)/2-float64(bounds.Dy())*scale/2,
	)
	opts.Filter = spriteDrawFilter()
	img.DrawImage(billboard.image, &opts)
	m.create.previewImage = img.RGBA()
	m.create.previewImageKey = key
	return m.create.previewImage
}

func (m *LoginMode) characterCreatePreviewView(ctx client.Context) *humanoidSpriteView {
	key := charCreatePreviewKey{sex: ctx.Session.Sex, hairStyle: m.create.hairStyle, hairColor: m.create.hairColor}
	if m.create.preview != nil && m.create.previewKey == key {
		return m.create.preview
	}
	if m.create.previewFailed && m.create.previewKey == key {
		return nil
	}
	character := session.Character{
		ID:        1,
		Name:      m.create.name,
		Job:       0,
		Hair:      int16(m.create.hairStyle),
		HairColor: uint8(m.create.hairColor),
	}
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex)
	m.create.previewKey = key
	if view == nil {
		m.create.previewFailed = true
		glog.Debugf("char create sprite resources hair=%d color=%d sex=%d %s", m.create.hairStyle, m.create.hairColor, ctx.Session.Sex, status)
		return nil
	}
	m.create.preview = view
	m.create.previewFailed = false
	return view
}

func pairedCreateStat(stat int) int {
	switch stat {
	case createStatStr:
		return createStatInt
	case createStatInt:
		return createStatStr
	case createStatAgi:
		return createStatLuk
	case createStatLuk:
		return createStatAgi
	case createStatVit:
		return createStatDex
	case createStatDex:
		return createStatVit
	default:
		return -1
	}
}

func bumpCreateStat(stats *[createStatCount]uint8, stat int) bool {
	if stats == nil || stat < 0 || stat >= createStatCount {
		return false
	}
	pair := pairedCreateStat(stat)
	if pair < 0 || (*stats)[stat] >= 9 || (*stats)[pair] <= 1 {
		return false
	}
	(*stats)[stat]++
	(*stats)[pair]--
	return true
}

func appendCharacterNameInput(current, input string, maxBytes int) string {
	if maxBytes <= 0 {
		return current
	}
	out := current
	for _, r := range input {
		if r < 32 || r == 127 {
			continue
		}
		next := out + string(r)
		if len([]byte(next)) > maxBytes {
			break
		}
		out = next
	}
	return out
}

func describeMakeCharacterRefuse(code uint8) string {
	switch code {
	case 0:
		return "character name already exists"
	case 1:
		return "account is underaged"
	case 2:
		return "invalid character name"
	default:
		return fmt.Sprintf("character creation refused (%d)", code)
	}
}
