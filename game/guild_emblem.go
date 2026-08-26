package game

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"io"
	"math"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
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

const siegeGuildEmblemSize = 24

func (m *WorldMode) requestActorGuildEmblem(ctx client.Context, guildID, version uint32) {
	m.requestGuildEmblem(ctx, guildID, version, false)
}

func (m *WorldMode) requestGuildEmblem(ctx client.Context, guildID, version uint32, force bool) {
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
	if !force && emblem.requestedVersion >= version {
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
	m.ui.guildWindow.Refresh(ctx)
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
			if guildID == 0 {
				guildID = ctx.Session.Guild.ID
			}
		}
		if version == 0 {
			version = ctx.Session.EmblemVersion
		}
	}
	if guildID == 0 || version == 0 {
		return nil
	}
	if m.guildEmblems == nil {
		m.requestActorGuildEmblem(ctx, guildID, version)
		return nil
	}
	emblem := m.guildEmblems[guildID]
	if emblem.image == nil || emblem.version < version {
		m.requestActorGuildEmblem(ctx, guildID, version)
		return nil
	}
	return emblem.image
}

func (m *WorldMode) drawSiegeGuildEmblems(screen *render.Frame, ctx client.Context, entries []sceneActorDrawEntry) {
	if screen == nil || ctx.World == nil || !ctx.World.MapProperty.IsSiege() {
		return
	}
	for _, entry := range entries {
		if !siegeActorShowsGuildEmblem(entry) {
			continue
		}
		emblem := m.actorGuildEmblem(ctx, entry.actor, entry.isPlayer)
		if emblem == nil {
			continue
		}
		bounds := emblem.Bounds()
		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			continue
		}
		x, y := siegeGuildEmblemPosition(entry, siegeGuildEmblemSize)
		var opts render.DrawImageOptions
		opts.Filter = render.FilterNearest
		opts.GeoM.Scale(float64(siegeGuildEmblemSize)/float64(bounds.Dx()), float64(siegeGuildEmblemSize)/float64(bounds.Dy()))
		opts.GeoM.Translate(x, y)
		screen.DrawImage(emblem, &opts)
	}
}

func siegeActorShowsGuildEmblem(entry sceneActorDrawEntry) bool {
	const hiddenEffectMask = db.EffectStateHide | db.EffectStateCloak | db.EffectStateInvisible | db.EffectStateChasewalk
	return !entry.hidden && entry.actor.EffectState&hiddenEffectMask == 0 && entry.actor.GuildID != 0 && entry.actor.EmblemVersion != 0
}

func siegeGuildEmblemPosition(entry sceneActorDrawEntry, size int) (float64, float64) {
	if size <= 0 {
		size = siegeGuildEmblemSize
	}
	x := math.Round(entry.screenX - float64(size)/2)
	y := math.Round(actorSpriteTopY(entry.screenY, entry.scale) - float64(size) - 4)
	return x, y
}
