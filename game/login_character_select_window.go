package game

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"image"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const (
	charSelectPreviewDirection = 4
	charSelectPreviewScale     = 0.92
	charSelectPreviewFeetLift  = 10
	charDeleteKeyMaxBytes      = 40
)

func (m *LoginMode) updateCharacterSelectInput(ctx client.Context) {
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEscape) {
		m.cancelCharacterSelect(ctx)
		return
	}
	if ctx.Input != nil {
		if ctx.Input.JustPressed(input.KeyArrowLeft) {
			m.moveSelectedSlot(-1)
		}
		if ctx.Input.JustPressed(input.KeyArrowRight) {
			m.moveSelectedSlot(1)
		}
		if ctx.Input.JustPressed(input.KeyEnter) {
			m.submitSelectedCharacter(ctx)
		}
	}
	m.showCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.charSelectWindow.Update(ctx)
		m.charSelectWindow.Publish(ctx)
	}
}

func (m *LoginMode) updateCharacterSelectWindow(ctx client.Context) {
	opts := gameui.CharacterSelectWindowOptions{
		SelectedSlot: m.selectedSlot,
		MaxSlots:     m.maxSlots,
	}
	if ctx.Session != nil {
		opts.Characters = ctx.Session.Characters
	}
	opts.PreviewImages = m.characterSelectPreviewImages(ctx, opts)
	callbacks := gameui.CharacterSelectWindowCallbacks{
		OnSelectSlot: func(slot int) {
			m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
		},
		OnActivateSlot: func(slot int) {
			if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
				m.submitSelectedCharacter(ctx)
			} else {
				m.openCharacterCreate(ctx, slot, time.Now())
			}
		},
		OnPreviousPage: func() {
			m.moveSelectedSlot(-3)
		},
		OnNextPage: func() {
			m.moveSelectedSlot(3)
		},
		OnMake: func() {
			slot := m.selectedSlot
			if _, ok := characterBySlot(ctx.Session.Characters, slot); ok {
				if empty, hasEmpty := firstEmptyCharacterSlot(ctx.Session.Characters, m.maxSlots); hasEmpty {
					slot = empty
				}
			}
			m.openCharacterCreate(ctx, slot, time.Now())
		},
		OnOK: func() {
			m.submitSelectedCharacter(ctx)
		},
		OnCancel: func() {
			m.cancelCharacterSelect(ctx)
		},
		OnDelete: func() {
			m.openCharacterDeleteConfirm(ctx)
		},
	}
	if m.charSelectWindow == nil {
		m.charSelectWindow = gameui.NewCharacterSelectWindow(ctx, opts, callbacks)
		return
	}
	m.charSelectWindow.SetOptions(ctx, opts)
}

func (m *LoginMode) characterSelectPreviewImages(ctx client.Context, opts gameui.CharacterSelectWindowOptions) map[int]image.Image {
	images := make(map[int]image.Image, 3)
	pageStart := gameui.CharacterSelectPage(opts.SelectedSlot) * 3
	for localSlot := 0; localSlot < 3; localSlot++ {
		slot := pageStart + localSlot
		character, ok := characterBySlot(opts.Characters, slot)
		if !ok {
			continue
		}
		if image := m.characterPreviewImage(ctx, character); image != nil {
			images[slot] = image
		}
	}
	return images
}

func (m *LoginMode) showCharacterSelectWindow(ctx client.Context) {
	m.updateCharacterSelectWindow(ctx)
	if m.charSelectWindow != nil {
		m.charSelectWindow.Publish(ctx)
	}
}

func (m *LoginMode) updateCharacterDelete(ctx client.Context) bool {
	if m.charDeleteConfirm.Update(ctx) {
		return true
	}
	consumed := m.charDeletePrompt.Update(ctx)
	action := m.charDeletePrompt.PopAction()
	if action.Submitted {
		m.submitCharacterDelete(ctx, action.Text)
		return true
	}
	return consumed
}

func (m *LoginMode) openCharacterDeleteConfirm(ctx client.Context) {
	if ctx.Session == nil {
		m.status = "delete character failed: no session"
		return
	}
	character, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot)
	if !ok {
		m.status = "empty character slot"
		return
	}
	m.deleteCharID = character.ID
	m.charDeleteConfirm.Open(ctx, "Delete Character", fmt.Sprintf("Delete %s?", character.Name), func() {
		m.charDeletePrompt.Open(ctx, "Delete Character", "Email", "Email", charDeleteKeyMaxBytes)
	}, nil)
}

func (m *LoginMode) submitCharacterDelete(ctx client.Context, key string) {
	if m.deleteCharID == 0 {
		m.status = "delete character failed: no character selected"
		return
	}
	if ctx.Network == nil {
		m.status = "delete character failed: not connected"
		return
	}
	if err := ctx.Network.SendDeleteCharacter(m.deleteCharID, key); err != nil {
		m.status = "delete character failed: " + err.Error()
		return
	}
	m.status = "delete character request sent"
}

func (m *LoginMode) applyDeleteCharacterAccept(ctx client.Context) {
	if m.deleteCharID == 0 {
		m.status = "deleted character"
		return
	}
	if ctx.Session == nil {
		m.deleteCharID = 0
		m.status = "deleted character"
		return
	}
	var deletedName string
	if character, ok := characterByID(ctx.Session.Characters, m.deleteCharID); ok {
		deletedName = character.Name
	}
	ctx.Session.Characters = removeCharacterByID(ctx.Session.Characters, m.deleteCharID)
	m.charViews = nil
	m.charViewFailed = nil
	m.charPreviewImages = nil
	m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		if len(ctx.Session.Characters) > 0 {
			m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
		} else {
			m.selectedSlot = 0
		}
	}
	m.deleteCharID = 0
	if deletedName != "" {
		m.status = fmt.Sprintf("deleted character %s", deletedName)
	} else {
		m.status = "deleted character"
	}
	m.showCharacterSelectWindow(ctx)
}

func (m *LoginMode) applyDeleteCharacterRefuse(ctx client.Context, code uint8) {
	m.deleteCharID = 0
	m.status = describeDeleteCharacterRefuse(code)
	m.charDeleteConfirm.OpenAlert(ctx, "Delete Character", m.status, nil)
}

func (m *LoginMode) cancelCharacterSelect(ctx client.Context) {
	m.charSelectWindow = nil
	m.startPhaseFade(loginPhaseAccount, time.Now())
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	m.status = "char select cancelled"
}

func (m *LoginMode) prepareCharacterSelectFromSession(ctx client.Context) {
	m.maxSlots = 9
	m.selectedSlot = 0
	if ctx.Session == nil || len(ctx.Session.Characters) == 0 {
		return
	}
	m.maxSlots = charSelectMaxSlots(ctx.Session.Characters)
	if configured := ctx.Config.Login.CharSlot; configured >= 0 {
		m.selectedSlot = clampCharacterSlot(configured, m.maxSlots)
	}
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		m.selectedSlot = firstOccupiedCharacterSlot(ctx.Session.Characters)
	}
}

func (m *LoginMode) autoSelectCharacter(ctx client.Context) bool {
	slot := ctx.Config.Login.CharSlot
	if !ctx.Config.Login.AutoLogin || slot < 0 || m.autoCharAttempted {
		return false
	}
	m.autoCharAttempted = true
	m.selectedSlot = clampCharacterSlot(slot, m.maxSlots)
	if _, ok := characterBySlot(ctx.Session.Characters, m.selectedSlot); !ok {
		m.status = fmt.Sprintf("character slot %d is empty", slot)
		glog.Warnf("autoselect character slot=%d failed: empty slot", slot)
		return false
	}
	glog.Debugf("autoselect character slot=%d", m.selectedSlot)
	m.submitSelectedCharacter(ctx)
	return true
}

func (m *LoginMode) reconnectCharacterServer(ctx client.Context) {
	if ctx.Network == nil || ctx.Session == nil || len(ctx.Session.CharServers) == 0 {
		return
	}
	server := ctx.Session.CharServers[0]
	m.connectCharServer(ctx, network.CharServer{
		Address:   server.Address,
		Port:      server.Port,
		Name:      server.Name,
		UserCount: server.UserCount,
		State:     server.State,
		Property:  server.Property,
	})
}

func (m *LoginMode) drawCharacterSelect(ctx client.Context) {
	m.showCharacterSelectWindow(ctx)
}

type characterPreviewTarget interface {
	DrawImage(*render.Image, *render.DrawImageOptions)
}

func (m *LoginMode) drawCharacterPreview(screen characterPreviewTarget, ctx client.Context, character session.Character, centerX, feetY int) {
	view := m.characterPreviewView(ctx, character)
	if view == nil {
		return
	}
	billboard, ok := humanoidBillboardForState(view, spriteState{
		actionFamily: spriteActionIdle,
		direction:    charSelectPreviewDirection,
		started:      time.Now(),
		loopIdle:     true,
	}, time.Now())
	if !ok || billboard == nil || billboard.image == nil {
		return
	}
	var opts render.DrawImageOptions
	scale := charSelectPreviewScale
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(centerX)-billboard.anchorX*scale, float64(feetY)-billboard.anchorY*scale)
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(billboard.image, &opts)
}

func (m *LoginMode) characterPreviewImage(ctx client.Context, character session.Character) image.Image {
	if character.ID == 0 {
		return nil
	}
	if m.charPreviewImages == nil {
		m.charPreviewImages = make(map[uint32]image.Image)
	}
	if img := m.charPreviewImages[character.ID]; img != nil {
		return img
	}
	img := render.NewImage(139, 144)
	m.drawCharacterPreview(img, ctx, character, 139/2, 144-15-charSelectPreviewFeetLift)
	m.charPreviewImages[character.ID] = img.RGBA()
	return m.charPreviewImages[character.ID]
}

func (m *LoginMode) characterPreviewView(ctx client.Context, character session.Character) *humanoidSpriteView {
	if character.ID == 0 {
		return nil
	}
	if _, failed := m.charViewFailed[character.ID]; failed {
		return nil
	}
	if m.charViews == nil {
		m.charViews = make(map[uint32]*humanoidSpriteView)
	}
	if view := m.charViews[character.ID]; view != nil {
		return view
	}
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex)
	if view == nil {
		if m.charViewFailed == nil {
			m.charViewFailed = make(map[uint32]struct{})
		}
		m.charViewFailed[character.ID] = struct{}{}
		glog.Debugf("char select sprite resources char_id=%d name=%s job=%d %s", character.ID, character.Name, character.Job, status)
		return nil
	}
	m.charViews[character.ID] = view
	return view
}

func (m *LoginMode) loadCharacterSelectSkin(ctx client.Context) {
	if m.charWindow == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/win_select.bmp"); ok {
			m.charWindow = img
		}
	}
	if m.charBox == nil {
		if img, _, ok := loadLoginBackgroundImage(ctx.Resources, "login_interface/box_select.bmp"); ok {
			m.charBox = img
		}
	}
}

func charSelectMaxSlots(characters []session.Character) int {
	maxSlots := 9
	for _, character := range characters {
		if int(character.Slot)+1 > maxSlots {
			maxSlots = int(character.Slot) + 1
		}
	}
	if maxSlots%3 != 0 {
		maxSlots += 3 - maxSlots%3
	}
	return maxSlots
}

func firstOccupiedCharacterSlot(characters []session.Character) int {
	if len(characters) == 0 {
		return 0
	}
	slot := int(characters[0].Slot)
	for _, character := range characters[1:] {
		if int(character.Slot) < slot {
			slot = int(character.Slot)
		}
	}
	return slot
}

func firstEmptyCharacterSlot(characters []session.Character, maxSlots int) (int, bool) {
	if maxSlots <= 0 {
		maxSlots = 9
	}
	occupied := make(map[int]struct{}, len(characters))
	for _, character := range characters {
		occupied[int(character.Slot)] = struct{}{}
	}
	for slot := 0; slot < maxSlots; slot++ {
		if _, ok := occupied[slot]; !ok {
			return slot, true
		}
	}
	return 0, false
}

func clampCharacterSlot(slot, maxSlots int) int {
	if maxSlots <= 0 {
		maxSlots = 1
	}
	if slot < 0 {
		return 0
	}
	if slot >= maxSlots {
		return maxSlots - 1
	}
	return slot
}

func characterBySlot(characters []session.Character, slot int) (session.Character, bool) {
	for _, character := range characters {
		if int(character.Slot) == slot {
			return character, true
		}
	}
	return session.Character{}, false
}

func characterByID(characters []session.Character, id uint32) (session.Character, bool) {
	for _, character := range characters {
		if character.ID == id {
			return character, true
		}
	}
	return session.Character{}, false
}

func upsertCharacter(characters []session.Character, character session.Character) []session.Character {
	for i := range characters {
		if characters[i].ID == character.ID || characters[i].Slot == character.Slot {
			characters[i] = character
			return characters
		}
	}
	return append(characters, character)
}

func removeCharacterByID(characters []session.Character, id uint32) []session.Character {
	out := characters[:0]
	for _, character := range characters {
		if character.ID != id {
			out = append(out, character)
		}
	}
	return out
}

func describeDeleteCharacterRefuse(code uint8) string {
	switch code {
	case 0:
		return "Character deletion refused."
	default:
		return fmt.Sprintf("Character deletion refused (%d).", code)
	}
}
