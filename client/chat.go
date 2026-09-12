package client

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
)

// SendChat sends console-style chat input through the appropriate channel.
func SendChat(ctx Context, input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("empty chat message")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}

	name := chatSenderName(ctx)
	switch {
	case strings.HasPrefix(input, "%"):
		return sendGroupChat(name, strings.TrimPrefix(input, "%"), ctx.Network.SendPartyMessage)
	case strings.HasPrefix(input, "$"):
		return sendGroupChat(name, strings.TrimPrefix(input, "$"), ctx.Network.SendGuildMessage)
	}

	command, arguments := splitChatCommand(input)
	switch command {
	case "/sit":
		return SetSitting(ctx, true)
	case "/stand":
		return SetSitting(ctx, false)
	case "/pvpinfo":
		if arguments != "" {
			return fmt.Errorf("usage: /pvpinfo")
		}
		if ctx.Session == nil {
			return fmt.Errorf("missing session")
		}
		return ctx.Network.SendPvPInfoRequest(ctx.Session.CharID, ctx.Session.AccountID)
	case "/w", "/whisper":
		target, message, ok := strings.Cut(arguments, " ")
		target = strings.TrimSpace(target)
		message = strings.TrimSpace(message)
		if !ok || target == "" || message == "" {
			return fmt.Errorf("usage: /w name message")
		}
		return ctx.Network.SendWhisper(target, message)
	}

	return ctx.Network.SendGlobalChat(name, input)
}

// SetSitting requests a sit or stand action and updates the local player state.
func SetSitting(ctx Context, sit bool) error {
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		return fmt.Errorf("cannot sit or stand while hiding")
	}
	if ctx.Network == nil {
		return fmt.Errorf("not connected")
	}
	targetID := localActorID(ctx)
	if targetID == 0 {
		return fmt.Errorf("missing local actor")
	}
	action := network.ActionStandUp
	if sit {
		action = network.ActionSitDown
	}
	if err := ctx.Network.SendActionRequest(targetID, action); err != nil {
		return err
	}
	if ctx.World != nil {
		ctx.World.Player.Sitting = sit
		if sit {
			ctx.World.Player.Moving = false
		}
	}
	return nil
}

func chatSenderName(ctx Context) string {
	if ctx.Session != nil {
		if name := strings.TrimSpace(ctx.Session.Selected.Name); name != "" {
			return name
		}
	}
	return "Player"
}

func localActorID(ctx Context) uint32 {
	if ctx.Session != nil {
		if ctx.Session.AccountID != 0 {
			return ctx.Session.AccountID
		}
		if ctx.Session.CharID != 0 {
			return ctx.Session.CharID
		}
	}
	if ctx.World != nil {
		return ctx.World.Player.ID
	}
	return 0
}

func sendGroupChat(name, input string, send func(string) error) error {
	message := strings.TrimSpace(input)
	if message == "" {
		return fmt.Errorf("empty chat message")
	}
	return send(fmt.Sprintf("%s : %s", name, message))
}

func splitChatCommand(input string) (string, string) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", ""
	}
	input = strings.TrimSpace(input)
	return strings.ToLower(fields[0]), strings.TrimSpace(strings.TrimPrefix(input, fields[0]))
}
