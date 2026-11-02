package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/battle"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func applyParameterChange(ctx client.Context, change network.ParameterChange) {
	if ctx.Session == nil {
		return
	}
	value := int(change.Value)
	switch change.VarID {
	case network.StatusSpeed:
		ctx.Session.Movement.ServerSpeed = value
		ctx.Session.Movement.HasServerSpeed = value > 0
		if ctx.World != nil {
			refreshLocalPlayerMoveSpeed(ctx)
		}
	case network.StatusBaseExp:
		ctx.Session.Progress.BaseExp = change.Value
	case network.StatusJobExp:
		ctx.Session.Progress.JobExp = change.Value
	case network.StatusHP:
		ctx.Session.Vitals.HP = value
		ctx.Session.Selected.HP = clampInt16(value)
	case network.StatusMaxHP:
		ctx.Session.Vitals.MaxHP = value
		ctx.Session.Selected.MaxHP = clampInt16(value)
	case network.StatusSP:
		ctx.Session.Vitals.SP = value
		ctx.Session.Selected.SP = clampInt16(value)
	case network.StatusMaxSP:
		ctx.Session.Vitals.MaxSP = value
		ctx.Session.Selected.MaxSP = clampInt16(value)
	case network.StatusPoint:
		ctx.Session.Stats.Points = value
	case network.StatusBaseLevel:
		ctx.Session.Progress.BaseLevel = value
		ctx.Session.Selected.Level = clampInt16(value)
	case network.StatusSkillPoint:
		ctx.Session.Skills.Points = value
	case network.StatusStr, network.StatusAgi, network.StatusVit, network.StatusInt, network.StatusDex, network.StatusLuk:
		setSessionStat(ctx.Session, change.VarID, value)
	case network.StatusUStr, network.StatusUAgi, network.StatusUVit, network.StatusUInt, network.StatusUDex, network.StatusULuk:
		setSessionStatCost(ctx.Session, change.VarID, value)
	case network.StatusZeny:
		ctx.Session.Inventory.Zeny = change.Value
	case network.StatusNextBaseExp:
		ctx.Session.Progress.NextBaseExp = change.Value
	case network.StatusNextJobExp:
		ctx.Session.Progress.NextJobExp = change.Value
	case network.StatusWeight:
		ctx.Session.Inventory.Weight = value
	case network.StatusMaxWeight:
		ctx.Session.Inventory.MaxWeight = value
	// Derived stats are signed 32-bit values. Keep the unsigned decoding of
	// experience, zeny and vitals intact while preserving negative modifiers.
	case network.StatusAttack:
		ctx.Session.Stats.Attack = int(int32(change.Value))
	case network.StatusAttackBonus:
		ctx.Session.Stats.AttackBonus = int(int32(change.Value))
	case network.StatusMatkMax:
		ctx.Session.Stats.MatkMax = int(int32(change.Value))
	case network.StatusMatkMin:
		ctx.Session.Stats.MatkMin = int(int32(change.Value))
	case network.StatusDefense:
		ctx.Session.Stats.Defense = int(int32(change.Value))
	case network.StatusDefenseBonus:
		ctx.Session.Stats.DefenseBonus = int(int32(change.Value))
	case network.StatusMDefense:
		ctx.Session.Stats.MDefense = int(int32(change.Value))
	case network.StatusMDefenseBonus:
		ctx.Session.Stats.MDefenseBonus = int(int32(change.Value))
	case network.StatusHit:
		ctx.Session.Stats.Hit = int(int32(change.Value))
	case network.StatusFlee:
		ctx.Session.Stats.Flee = int(int32(change.Value))
	case network.StatusFleeBonus:
		ctx.Session.Stats.FleeBonus = int(int32(change.Value))
	case network.StatusCritical:
		ctx.Session.Stats.Critical = int(int32(change.Value))
	case network.StatusASPD:
		ctx.Session.Stats.ASPD = int(int32(change.Value))
	case network.StatusASPDBonus:
		ctx.Session.Stats.ASPDBonus = int(int32(change.Value))
	case network.StatusJobLevel:
		ctx.Session.Progress.JobLevel = value
		ctx.Session.Selected.JobLevel = clampInt16(value)
	default:
		return
	}
	switch change.VarID {
	case network.StatusHP, network.StatusMaxHP:
		social.SyncLocalPartyVitals(ctx)
	}
	glog.Debugf("parameter change var=%d value=%d hp=%d/%d sp=%d/%d base_lv=%d job_lv=%d base_exp=%d/%d job_exp=%d/%d zeny=%d weight=%d/%d",
		change.VarID,
		change.Value,
		ctx.Session.Vitals.HP,
		ctx.Session.Vitals.MaxHP,
		ctx.Session.Vitals.SP,
		ctx.Session.Vitals.MaxSP,
		ctx.Session.Progress.BaseLevel,
		ctx.Session.Progress.JobLevel,
		ctx.Session.Progress.BaseExp,
		ctx.Session.Progress.NextBaseExp,
		ctx.Session.Progress.JobExp,
		ctx.Session.Progress.NextJobExp,
		ctx.Session.Inventory.Zeny,
		ctx.Session.Inventory.Weight,
		ctx.Session.Inventory.MaxWeight)
}

func (m *WorldMode) applyParameterChange(ctx client.Context, change network.ParameterChange) {
	if ctx.Session == nil {
		return
	}
	previousHP := ctx.Session.Vitals.HP
	previousSP := ctx.Session.Vitals.SP
	previousBaseLevel := ctx.Session.Progress.BaseLevel
	previousJobLevel := ctx.Session.Progress.JobLevel
	applyParameterChange(ctx, change)
	m.pets().ApplyTalkParameterChange(ctx, change, previousHP, previousBaseLevel)
	if change.Value <= 0 {
		return
	}
	previousValues := map[uint16]int{
		network.StatusHP:        previousHP,
		network.StatusSP:        previousSP,
		network.StatusBaseLevel: previousBaseLevel,
		network.StatusJobLevel:  previousJobLevel,
	}
	if visual, ok := statusVisualEffects[change.VarID]; ok {
		if !visual.recovery {
			visual.applyParameterChange(ctx, m, previousValues[change.VarID])
		}
	}
	if change.VarID == network.StatusBaseLevel {
		m.effects().SyncLevel99AuraEffects(ctx, time.Now())
	}
}

var (
	recoveryHPColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}
	recoverySPColor = color.RGBA{R: 0, G: 0, B: 255, A: 255}
)

const (
	recoveryHPSFX = "_heal_effect.wav"
	recoverySPSFX = "effect\\흡기.wav"
)

type statusVisualEffect struct {
	current       func(*session.Session) int
	recover       func(*session.Session, int) bool
	recovery      bool
	recoveryColor color.RGBA
	recoveryKind  battle.FloaterKind
	recoverySFX   []string
	clearsDeath   bool
	levelEffectID int
}

var statusVisualEffects = map[uint16]statusVisualEffect{
	network.StatusHP: {
		current:       func(s *session.Session) int { return s.Vitals.HP },
		recover:       recoverSessionHP,
		recovery:      true,
		recoveryColor: recoveryHPColor,
		recoveryKind:  battle.FloaterRecoveryHP,
		recoverySFX:   []string{recoveryHPSFX},
		clearsDeath:   true,
	},
	network.StatusSP: {
		current:       func(s *session.Session) int { return s.Vitals.SP },
		recover:       recoverSessionSP,
		recovery:      true,
		recoveryColor: recoverySPColor,
		recoveryKind:  battle.FloaterRecoverySP,
		recoverySFX:   []string{recoverySPSFX},
	},
	network.StatusBaseLevel: {
		current:       func(s *session.Session) int { return s.Progress.BaseLevel },
		levelEffectID: effects.BaseLevelUp,
	},
	network.StatusJobLevel: {
		current:       func(s *session.Session) int { return s.Progress.JobLevel },
		levelEffectID: effects.JobLevelUp,
	},
}

func (v statusVisualEffect) applyParameterChange(ctx client.Context, mode *WorldMode, previous int) {
	if v.current == nil || mode == nil || ctx.Session == nil {
		return
	}
	current := v.current(ctx.Session)
	if v.recovery {
		delta := current - previous
		if delta > 0 {
			mode.battle().AddLocalRecoveryFloater(ctx, delta, v.recoveryColor, v.recoveryKind)
			mode.scheduleSound(time.Now(), v.sfxCandidates()...)
		}
		return
	}
	if v.levelEffectID > 0 && current > previous {
		actorID := scene.LocalSkillTarget(ctx)
		mode.addLevelUpEffect(ctx, actorID, v.levelEffectID)
		switch v.levelEffectID {
		case effects.BaseLevelUp:
			mode.ui.levelUpNotifications.NotifyBase()
		case effects.JobLevelUp:
			mode.ui.levelUpNotifications.NotifyJob()
		}
	}
}

func (v statusVisualEffect) sfxCandidates() []string {
	return append([]string(nil), v.recoverySFX...)
}

func recoverSessionHP(s *session.Session, amount int) bool {
	maxHP := s.Vitals.MaxHP
	if maxHP <= 0 {
		maxHP = int(s.Selected.MaxHP)
	}
	next := s.Vitals.HP + amount
	if maxHP > 0 && next > maxHP {
		next = maxHP
	}
	s.Vitals.HP = next
	s.Selected.HP = clampInt16(next)
	return true
}

func recoverSessionSP(s *session.Session, amount int) bool {
	maxSP := s.Vitals.MaxSP
	if maxSP <= 0 {
		maxSP = int(s.Selected.MaxSP)
	}
	next := s.Vitals.SP + amount
	if maxSP > 0 && next > maxSP {
		next = maxSP
	}
	s.Vitals.SP = next
	s.Selected.SP = clampInt16(next)
	return true
}

func clampInt16(value int) int16 {
	if value < -32768 {
		return -32768
	}
	if value > 32767 {
		return 32767
	}
	return int16(value)
}
