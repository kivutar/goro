package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func actorVisualJob(actor worldstate.Actor) int {
	return visualJobForEffectState(int(actor.Job), actor.EffectState)
}

func actorWithVisualJob(actor worldstate.Actor) worldstate.Actor {
	actor.Job = int16(actorVisualJob(actor))
	return actor
}

func characterWithVisualJob(character session.Character) session.Character {
	return characterWithVisualJobForEffectState(character, character.Option)
}

func characterWithVisualJobForEffectState(character session.Character, effectState uint32) session.Character {
	character.Job = int16(visualJobForEffectState(int(character.Job), effectState))
	return character
}

func visualJobForEffectState(job int, effectState uint32) int {
	if effectState&db.EffectStateWedding != 0 {
		return db.JobMarried
	}
	if effectState&db.EffectStateRiding != 0 {
		if mounted, ok := db.MountJob(job); ok {
			return mounted
		}
	}
	return job
}

func localPlayerVisualCharacter(ctx client.Context) session.Character {
	if ctx.Session == nil {
		return session.Character{}
	}
	character := ctx.Session.SelectedCharacter()
	effectState := character.Option
	if ctx.World != nil && ctx.World.Player.HasState {
		effectState = ctx.World.Player.EffectState
	} else if ctx.World != nil && ctx.World.Player.EffectState != 0 {
		effectState = ctx.World.Player.EffectState
	}
	return characterWithVisualJobForEffectState(character, effectState)
}

func localPlayerVisualJob(ctx client.Context) int {
	if ctx.Session == nil {
		return 0
	}
	return int(localPlayerVisualCharacter(ctx).Job)
}

func (m *WorldMode) reloadPlayerSpriteView(ctx client.Context, reason string) {
	if ctx.Session == nil || ctx.Resources == nil {
		return
	}
	character := localPlayerVisualCharacter(ctx)
	view, status := loadPlayerHumanoidSpriteView(ctx.Resources, character, ctx.Session.Sex, localPlayerIsAdmin(ctx))
	if view != nil {
		m.playerView = view
		glog.Debugf("player sprite reloaded reason=%s visual_job=%d %s", reason, character.Job, status)
		return
	}
	glog.Warnf("player sprite reload failed reason=%s visual_job=%d: %s", reason, character.Job, status)
	m.playerView = nil
}
