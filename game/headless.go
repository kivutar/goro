package game

import (
	"fmt"
	"time"

	"github.com/kivutar/goro/client"
)

func (m *WorldMode) updateHeadless(ctx client.Context, now time.Time) error {
	if m.ui.npcDialog.IsOpen() {
		return fmt.Errorf("headless mode cannot respond to an NPC dialog")
	}
	if m.ui.teleportModal.IsOpen() {
		return fmt.Errorf("headless mode cannot choose a teleport destination")
	}
	if m.ui.nonConsoleKeyboardInputBlocked(ctx) || m.petSlotMachine.active {
		return fmt.Errorf("headless mode cannot respond to an interactive window")
	}
	if !playerIsDead(ctx) {
		m.updateCompanionAI(ctx, now)
	}
	m.updateBot(ctx, now)
	m.updateBotInput(ctx, false)
	if m.bot != nil {
		return m.bot.err
	}
	return nil
}
