package game

import (
	"fmt"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func (m *WorldMode) sendGuildInvite(ctx client.Context, actorID uint32, name string) {
	name = strings.TrimSpace(name)
	if actorID == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("guild invite failed target=%d name=%q: not connected", actorID, name)
		m.ui.console.AddErrorMessage("Guild invitation failed: not connected.")
		return
	}
	accountID, charID := uint32(0), uint32(0)
	if ctx.Session != nil {
		accountID = ctx.Session.AccountID
		charID = ctx.Session.CharID
	}
	if err := ctx.Network.SendGuildInvite(actorID, accountID, charID); err != nil {
		glog.Warnf("guild invite failed target=%d name=%q: %v", actorID, name, err)
		m.ui.console.AddErrorMessage("Guild invitation failed.")
		return
	}
	m.ui.console.AddBlueMessage("%s has received an invitation to join your guild.", guildDisplayName(name))
}

func (m *WorldMode) openGuildInviteRequest(ctx client.Context, request network.GuildInviteRequest) {
	name := guildDisplayName(request.GuildName)
	rawName := strings.TrimSpace(request.GuildName)
	m.ui.guildRequest.Open(ctx, "Guild Invitation", fmt.Sprintf("Would you like to join %s?", name), func() {
		if ctx.Network == nil {
			glog.Warnf("guild invite accept failed: not connected")
			return
		}
		if err := ctx.Network.SendGuildInviteReply(request.GuildID, true); err != nil {
			glog.Warnf("guild invite accept failed guild=%d name=%q: %v", request.GuildID, request.GuildName, err)
			return
		}
		applyLocalGuildName(ctx, rawName)
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("guild invite reject failed: not connected")
			return
		}
		if err := ctx.Network.SendGuildInviteReply(request.GuildID, false); err != nil {
			glog.Warnf("guild invite reject failed guild=%d name=%q: %v", request.GuildID, request.GuildName, err)
		}
	})
}

func (m *WorldMode) handleGuildCreationResult(ctx client.Context, result network.GuildCreationResult) {
	switch result.Result {
	case 0:
		if name := pendingGuildName(ctx); name != "" {
			applyLocalGuildName(ctx, name)
		}
		m.ui.console.AddBlueMessage("Guild created.")
	case 1:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("You are already in a guild.")
	case 2:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("Guild name already exists.")
	case 3:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("You need the required item to create a guild.")
	default:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("Guild creation failed.")
	}
}

func (m *WorldMode) handleGuildInviteAck(ack network.GuildInviteAck) {
	switch ack.Result {
	case 0:
		m.ui.console.AddErrorMessage("That character is already in a guild.")
	case 1:
		m.ui.console.AddErrorMessage("Guild invitation was rejected.")
	case 2:
		m.ui.console.AddBlueMessage("Guild invitation accepted.")
	case 3:
		m.ui.console.AddErrorMessage("The guild is full.")
	default:
		m.ui.console.AddErrorMessage("Guild invitation failed.")
	}
}

func guildDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Guild"
	}
	return name
}

func pendingGuildName(ctx client.Context) string {
	if ctx.Session == nil {
		return ""
	}
	return strings.TrimSpace(ctx.Session.PendingGuildName)
}

func clearPendingGuildName(ctx client.Context) {
	if ctx.Session != nil {
		ctx.Session.PendingGuildName = ""
	}
}

func applyLocalGuildName(ctx client.Context, name string) {
	applyLocalGuildInfo(ctx, 0, 0, name)
}

func applyLocalGuildInfo(ctx client.Context, guildID, emblemVersion uint32, name string) {
	name = strings.TrimSpace(name)
	if name == "" && guildID == 0 && emblemVersion == 0 {
		clearPendingGuildName(ctx)
		return
	}
	if ctx.Session != nil {
		if guildID != 0 {
			ctx.Session.GuildID = guildID
		}
		if emblemVersion != 0 {
			ctx.Session.EmblemVersion = emblemVersion
		}
		if name != "" {
			ctx.Session.GuildName = name
		}
		ctx.Session.PendingGuildName = ""
	}
	if ctx.World != nil {
		if guildID != 0 {
			ctx.World.Player.GuildID = guildID
		}
		if emblemVersion != 0 {
			ctx.World.Player.EmblemVersion = emblemVersion
		}
		if name != "" {
			ctx.World.Player.GuildName = name
		}
	}
}
