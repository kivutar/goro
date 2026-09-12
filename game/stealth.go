package game

import (
	"fmt"
	"image/color"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	worldstate "github.com/kivutar/goro/world"
)

const actorStealthMask = db.EffectStateHide | db.EffectStateCloak | db.EffectStateInvisible | db.EffectStateChasewalk

type stealthView uint8

const (
	stealthVisible stealthView = iota
	stealthHidden
	stealthShadow
	stealthSilhouette
)

func actorHasStealth(actor worldstate.Actor) bool {
	return actor.EffectState&actorStealthMask != 0
}

// The 2008 Sakexe hides cloaked bodies, including self and party members.
// CPc::Render (0x005e4920) reveals them only through Clairvoyance, in black.
// CGameActor::SetAttrState (0x00572b80) keeps Hiding shadows for self/GM;
// Invisible takes precedence and is not revealed by Clairvoyance.
func actorStealthView(ctx client.Context, actor worldstate.Actor, local bool) stealthView {
	if !actorHasStealth(actor) {
		return stealthVisible
	}
	if actor.EffectState&db.EffectStateInvisible != 0 {
		return stealthHidden
	}
	if localActorHasStatus(ctx, db.StatusClairvoyance) {
		return stealthSilhouette
	}
	if actor.EffectState&(db.EffectStateCloak|db.EffectStateChasewalk) == 0 && (local || localPlayerIsAdmin(ctx)) {
		return stealthShadow
	}
	return stealthHidden
}

func (v stealthView) tint(tint color.RGBA) color.RGBA {
	if v == stealthSilhouette {
		tint.R, tint.G, tint.B = 0, 0, 0
	}
	return tint
}

func localStealthAllowsSkill(ctx client.Context, skillID uint16) bool {
	// Commands to companions are independent of their owner's stealth.
	if skillCasterKindForID(skillID) != skillCasterPlayer {
		return true
	}
	if ctx.PlayerHasEffectState(db.EffectStateChasewalk) {
		return skillID == db.SkillSTChasewalk
	}
	if ctx.PlayerHasEffectState(db.EffectStateHide) {
		switch skillID {
		case db.SkillTFHiding, db.SkillASGrimtooth, db.SkillRGBackstap, db.SkillRGRaid, db.SkillNJShadowjump, db.SkillNJKirikage:
			return true
		default:
			return false
		}
	}
	return true
}

func checkStealthSkill(ctx client.Context, skillID uint16, targetID uint32) error {
	if !localStealthAllowsSkill(ctx, skillID) {
		return fmt.Errorf("skill cannot be used while hidden")
	}
	if actor, ok, local := actorForCombatID(ctx, targetID); ok && !local && actorHasStealth(actor) {
		return fmt.Errorf("target is hidden")
	}
	return nil
}
