package game

import (
	"time"

	"github.com/kivutar/goro/client"
	gameui "github.com/kivutar/goro/ui"
	lua "github.com/yuin/gopher-lua"
)

func registerLuaUIAPI(L *lua.LState, api *lua.LTable, script *luaScript) {
	ui := L.NewTable()
	control := func(action string) bool {
		return script.ui != nil && script.ui.controlScriptUI(script.ctx, action)
	}
	L.SetFuncs(ui, map[string]lua.LGFunction{
		"active":  func(L *lua.LState) int { L.Push(lua.LBool(control(""))); return 1 },
		"control": func(L *lua.LState) int { L.Push(lua.LBool(control(L.CheckString(1)))); return 1 },
	})
	api.RawSetString("ui", ui)
	api.RawSetString("in_game", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(script.mode != nil))
		return 1
	}))
}

func (m *WorldMode) controlScriptUI(ctx client.Context, action string) bool {
	if m.uiInputSuspended() || !m.ui.npcInputAvailable() || !m.ui.npcDialog.IsOpen() {
		return false
	}
	if action == "" {
		return true
	}
	return m.ui.npcDialog.Control(ctx, action)
}

func (m *LoginMode) controlScriptUI(ctx client.Context, action string) bool {
	// Login owns navigation even while fading or waiting for a server, so the
	// same confirmation cannot also click the next screen through the pointer.
	if m.fade.phase != loginFadeNone {
		return true
	}
	for _, modal := range []*gameui.ConfirmModal{&m.quitConfirm, &m.disconnectDialog, &m.charDeleteConfirm} {
		if modal.IsOpen() {
			modal.Control(ctx, action)
			return true
		}
	}
	if m.charDeletePrompt.IsOpen() {
		m.charDeletePrompt.Control(ctx, action)
		return true
	}
	// Creation uses the existing pointer for its appearance/stat controls.
	// Start/Escape still goes back; its buttons and fields handle submission.
	if m.phase == loginPhaseCreate {
		return false
	}
	if action == "" {
		return true
	}
	if action == "cancel" {
		m.cancelLoginPhase(ctx, time.Now())
		return true
	}
	switch m.phase {
	case loginPhaseCharacter:
		if m.charSelectPending {
			return true
		}
		switch action {
		case "left":
			m.moveToPreviousCharacterSlot()
		case "right":
			m.moveToNextCharacterSlot()
		case "confirm":
			m.activateCharacterSelectSlot(ctx, m.selectedSlot, time.Now())
		}
	case loginPhaseAccount:
		if m.loginPending || m.accountStep == loginAccountCharacterConnecting {
			return true
		}
		if m.serviceWindow != nil {
			return m.serviceWindow.Control(action)
		}
		if m.loginWindow != nil {
			return m.loginWindow.Control(ctx, action)
		}
	}
	return true
}
