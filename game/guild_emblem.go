package game

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"io"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type guildEmblem struct {
	version          uint32
	requestedVersion uint32
	image            *render.Image
}

func (m *WorldMode) requestActorGuildEmblem(ctx client.Context, guildID, version uint32) {
	if guildID == 0 || version == 0 || ctx.Network == nil {
		return
	}
	if m.guildEmblems == nil {
		m.guildEmblems = make(map[uint32]guildEmblem)
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image != nil && emblem.version >= version {
		return
	}
	if emblem.requestedVersion >= version {
		return
	}
	if err := ctx.Network.SendGuildEmblemRequest(guildID); err != nil {
		glog.Warnf("request guild emblem failed guild=%d version=%d: %v", guildID, version, err)
		return
	}
	emblem.requestedVersion = version
	m.guildEmblems[guildID] = emblem
}

func (m *WorldMode) applyGuildEmblemImage(ctx client.Context, packet network.GuildEmblemImage) {
	image, err := decodeGuildEmblemImage(packet.Data)
	if err != nil {
		glog.Warnf("decode guild emblem failed guild=%d version=%d: %v", packet.GuildID, packet.EmblemVersion, err)
		return
	}
	if m.guildEmblems == nil {
		m.guildEmblems = make(map[uint32]guildEmblem)
	}
	m.guildEmblems[packet.GuildID] = guildEmblem{
		version:          packet.EmblemVersion,
		requestedVersion: packet.EmblemVersion,
		image:            render.NewImageFromImage(image),
	}
	glog.Debugf("guild emblem loaded guild=%d version=%d size=%dx%d", packet.GuildID, packet.EmblemVersion, image.Bounds().Dx(), image.Bounds().Dy())
}

func decodeGuildEmblemImage(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty emblem")
	}
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("inflate emblem: %w", err)
	}
	defer zr.Close()
	decoded, err := io.ReadAll(io.LimitReader(zr, 4096))
	if err != nil {
		return nil, fmt.Errorf("read inflated emblem: %w", err)
	}
	img, err := res.DecodeImageData(decoded)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (m *WorldMode) applyGuildEmblemChange(ctx client.Context, change network.GuildEmblemChange) {
	if change.ActorID == 0 || ctx.World == nil {
		return
	}
	if isLocalActor(ctx, change.ActorID) {
		ctx.World.Player.GuildID = change.GuildID
		ctx.World.Player.EmblemVersion = change.EmblemVersion
		if ctx.Session != nil {
			ctx.Session.GuildID = change.GuildID
			ctx.Session.EmblemVersion = change.EmblemVersion
		}
	} else if actor, ok := ctx.World.Actors[change.ActorID]; ok {
		actor.GuildID = change.GuildID
		actor.EmblemVersion = change.EmblemVersion
		ctx.World.Actors[change.ActorID] = actor
	}
	m.requestActorGuildEmblem(ctx, change.GuildID, change.EmblemVersion)
}

func (m *WorldMode) actorGuildEmblem(ctx client.Context, actor worldstate.Actor, isPlayer bool) *render.Image {
	guildID := actor.GuildID
	version := actor.EmblemVersion
	if isPlayer && ctx.Session != nil {
		if guildID == 0 {
			guildID = ctx.Session.GuildID
		}
		if version == 0 {
			version = ctx.Session.EmblemVersion
		}
	}
	if guildID == 0 || version == 0 || m.guildEmblems == nil {
		return nil
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image == nil || emblem.version < version {
		m.requestActorGuildEmblem(ctx, guildID, version)
		return nil
	}
	return emblem.image
}
