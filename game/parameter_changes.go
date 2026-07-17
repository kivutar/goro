package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/client"
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
	case network.StatusJobLevel:
		ctx.Session.Progress.JobLevel = value
		ctx.Session.Selected.JobLevel = clampInt16(value)
	default:
		return
	}
	switch change.VarID {
	case network.StatusHP, network.StatusMaxHP:
		syncLocalPartyVitals(ctx)
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
	m.applyPetTalkParameterChange(ctx, change, previousHP, previousBaseLevel)
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
		visual.applyParameterChange(ctx, m, previousValues[change.VarID])
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
	recoveryKind  damageFloaterKind
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
		recoveryKind:  damageFloaterRecoveryHP,
		recoverySFX:   []string{recoveryHPSFX},
		clearsDeath:   true,
	},
	network.StatusSP: {
		current:       func(s *session.Session) int { return s.Vitals.SP },
		recover:       recoverSessionSP,
		recovery:      true,
		recoveryColor: recoverySPColor,
		recoveryKind:  damageFloaterRecoverySP,
		recoverySFX:   []string{recoverySPSFX},
	},
	network.StatusBaseLevel: {
		current:       func(s *session.Session) int { return s.Progress.BaseLevel },
		levelEffectID: effectBaseLevelUp,
	},
	network.StatusJobLevel: {
		current:       func(s *session.Session) int { return s.Progress.JobLevel },
		levelEffectID: effectJobLevelUp,
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
			mode.addLocalRecoveryFloater(ctx, delta, v.recoveryColor, v.recoveryKind)
			mode.scheduleSound(time.Now(), v.sfxCandidates()...)
		}
		return
	}
	if v.levelEffectID > 0 && current > previous {
		mode.addWorldEffectIfMissing(ctx, v.levelEffectID, localSkillTarget(ctx))
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
