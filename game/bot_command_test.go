package game

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gogpu/gpucontext"
	lua "github.com/yuin/gopher-lua"
)

func TestChatScriptSwitchSurvivesMapChangeAndStops(t *testing.T) {
	ctx := chatShortcutTestContext(t)
	ctx.Config.Script.Path = filepath.Join("..", "scripts", "wasd.lua")
	mode := NewWorldMode()
	loadKeyboardTestBot(t, ctx, mode)
	original := mode.bot
	mode.ui.console.SendText(ctx, "/script wasd")
	mode.updateBot(ctx, time.Now())
	if mode.bot == nil || mode.bot.disabled || mode.bot.path != "builtin:wasd" || original.state != nil {
		t.Fatal("chat command did not replace the configured script and close its Lua state")
	}
	t.Cleanup(mode.bot.close)
	if err := mode.bot.state.DoString(`function input() did_input = true end`); err != nil {
		t.Fatal(err)
	}
	mode.updateBotInput(ctx, true)
	if mode.bot.state.GetGlobal("did_input") != lua.LTrue {
		t.Fatal("frame input still used the configured script path")
	}

	next := mode.nextWorldMode()
	next.updateBot(ctx, time.Now())
	if next.bot == nil || next.bot.disabled || next.bot.path != "builtin:wasd" {
		t.Fatal("map change lost the selected script")
	}
	selected := next.bot
	t.Cleanup(selected.close)
	ctx.Input.SetKeyCode(gpucontext.KeyW, true)
	next.botKeyPress(ctx, gpucontext.KeyW)
	if !ctx.Input.KeyCodeConsumed(gpucontext.KeyW) {
		t.Fatal("selected script did not receive keyboard input")
	}
	ctx.Input.ResetKeyboard()
	next.ui.console.SendText(ctx, "/script none")
	ctx.Input.SetKeyCode(gpucontext.KeyW, true)
	next.botKeyPress(ctx, gpucontext.KeyW)
	if ctx.Input.KeyCodeConsumed(gpucontext.KeyW) {
		t.Fatal("stopped script still intercepted the keyboard")
	}
	next.updateBot(ctx, time.Now())
	if next.bot != nil || selected.state != nil {
		t.Fatal("stop command did not close the Lua state")
	}
}
