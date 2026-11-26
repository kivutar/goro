package game

import (
	"github.com/gogpu/gpucontext"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	gameui "github.com/kivutar/goro/ui"
)

// Like the other shortcuts, these use physical keys, not typed characters.
var windowShortcutKeys = [...]input.KeyCode{
	gpucontext.KeyE, gpucontext.KeyQ, gpucontext.KeyS, gpucontext.KeyA,
	gpucontext.KeyZ, gpucontext.KeyH, gpucontext.KeyW, gpucontext.KeyC,
	gpucontext.KeyV, gpucontext.KeyJ, gpucontext.KeyR, gpucontext.KeyO,
}

// Shared with text filtering so handled shortcuts cannot also type into chat.
func (m *WorldMode) windowShortcutAllowed(ctx client.Context, code input.KeyCode) bool {
	var target *gameui.Window
	if code == gpucontext.KeyR && plainCtrlDown(ctx.Input) {
		target = &m.ui.companions.MercenaryInfo.Window
	} else {
		if !plainAltDown(ctx.Input) {
			return false
		}
		switch code {
		case gpucontext.KeyE, gpucontext.KeyQ, gpucontext.KeyS, gpucontext.KeyA,
			gpucontext.KeyZ, gpucontext.KeyH, gpucontext.KeyW, gpucontext.KeyV:
		case gpucontext.KeyC:
			target = &m.ui.social.ChatRoomCreate.Window
		case gpucontext.KeyJ:
			target = &m.ui.pet.Info.Window
		case gpucontext.KeyR:
			target = &m.ui.companions.HomunculusInfo.Window
		case gpucontext.KeyO:
			target = &m.ui.settingsWindow.Window
		default:
			return false
		}
	}
	return !m.ui.nonConsoleKeyboardInputBlockedExcept(target)
}

func (m *WorldMode) toggleWindowFromInput(ctx client.Context) bool {
	if !plainAltDown(ctx.Input) && !plainCtrlDown(ctx.Input) {
		return false
	}
	for _, code := range windowShortcutKeys {
		if !ctx.Input.KeyCodeJustPressed(code) || !m.windowShortcutAllowed(ctx, code) {
			continue
		}
		ctx.Input.ConsumeKeyCodePress(code)
		switch code {
		case gpucontext.KeyE:
			m.ui.inventoryBag.Toggle(ctx)
		case gpucontext.KeyQ:
			m.ui.equipmentWindow.Toggle(ctx)
		case gpucontext.KeyS:
			m.ui.skillWindow.Toggle(ctx)
		case gpucontext.KeyA:
			m.ui.statsWindow.Toggle(ctx)
		case gpucontext.KeyZ:
			m.ui.social.FriendsWindow.ToggleParty(ctx)
		case gpucontext.KeyH:
			m.ui.social.FriendsWindow.ToggleFriends(ctx)
		case gpucontext.KeyW:
			m.ui.cartWindow.Toggle(ctx)
		case gpucontext.KeyV:
			m.ui.characterWindow.ToggleCompact()
			m.ui.basicMenu.FollowCharacterWindow(ctx, &m.ui.characterWindow)
		case gpucontext.KeyC:
			if m.ui.social.ChatRoomCreate.IsOpen() {
				m.ui.social.ChatRoomCreate.Close()
			} else {
				m.ui.social.ChatRoomCreate.Open(ctx)
			}
		case gpucontext.KeyJ:
			if m.ui.pet.Info.IsOpen() {
				m.ui.pet.Info.Close()
			} else if m.petSystem.ID != 0 {
				m.pets().OpenInfo(ctx)
			}
		case gpucontext.KeyR:
			if plainCtrlDown(ctx.Input) {
				if m.ui.companions.MercenaryInfo.IsOpen() {
					m.ui.companions.MercenaryInfo.Close()
				} else {
					m.companions().OpenMercenaryInfo(ctx)
				}
			} else if m.ui.companions.HomunculusInfo.IsOpen() {
				m.ui.companions.HomunculusInfo.Close()
			} else {
				m.companions().OpenHomunculusInfo(ctx)
			}
		case gpucontext.KeyO:
			if m.ui.settingsWindow.IsOpen() {
				m.ui.settingsWindow.Close()
			} else {
				m.ui.settingsWindow.OpenWindow(ctx)
			}
		}
		return true
	}
	return false
}
