package effects

import (
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/companion"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type skillActorAction int

const SkillActorActionNone skillActorAction = -1

const (
	SkillActorActionIdle skillActorAction = iota
	SkillActorActionSkill
	SkillActorActionAttack
	SkillActorActionAttack1
	skillActorActionAttack2
	SkillActorActionAttack3
	SkillActorActionPickup
	SkillActorActionReadyFight
	skillActorActionWalk
)

type SkillActionSpec struct {
	Defined       bool
	Action        skillActorAction
	Frame         int
	HasFrame      bool
	Length        int
	Speed         time.Duration
	Duration      time.Duration
	Combo         bool
	Play          bool
	Repeat        bool
	PingPong      bool
	RandomReverse bool
	Delay         time.Duration
	Next          *SkillActionSpec
}

var (
	IdleSkillActionSpec       = newSkillActionSpec(SkillActorActionIdle, true, nil)
	defaultSkillActionSpec    = newSkillActionSpec(SkillActorActionSkill, false, &IdleSkillActionSpec)
	AttackSkillActionSpec     = newSkillActionSpec(SkillActorActionAttack, false, &IdleSkillActionSpec)
	ReadyFightSkillActionSpec = newSkillActionSpec(SkillActorActionReadyFight, false, &IdleSkillActionSpec)
)

func newSkillActionSpec(action skillActorAction, repeat bool, next *SkillActionSpec) SkillActionSpec {
	return SkillActionSpec{
		Defined: true,
		Action:  action,
		Frame:   0,
		Repeat:  repeat,
		Play:    true,
		Next:    cloneSkillActionSpec(next),
	}
}

func cloneSkillActionSpec(spec *SkillActionSpec) *SkillActionSpec {
	if spec == nil {
		return nil
	}
	clone := *spec
	clone.Next = cloneSkillActionSpec(spec.Next)
	return &clone
}

func (s SkillActionSpec) ActionFamilyForActorWithResources(manager *res.Manager, actor worldstate.Actor) int {
	if s.Action == SkillActorActionNone {
		return -1
	}
	mercenary := companion.IsMercenary(actor)
	if !res.HasPlayerJobToken(scene.ActorVisualJob(actor)) && !mercenary {
		return sprites.ActionNonPCAttack
	}
	switch s.Action {
	case SkillActorActionIdle:
		return scene.SpriteActionIdle
	case skillActorActionWalk:
		return scene.SpriteActionWalk
	case SkillActorActionAttack:
		return sprites.AttackActionFamily(manager, actor)
	case SkillActorActionAttack1:
		return sprites.ActionPCAttack1
	case skillActorActionAttack2:
		return sprites.ActionPCAttack2
	case SkillActorActionAttack3:
		return sprites.ActionPCAttack3
	case SkillActorActionPickup:
		return scene.SpriteActionPickup
	case SkillActorActionReadyFight:
		if mercenary {
			return sprites.ActionPCSkill
		}
		return sprites.ActionPCReadyFight
	default:
		return sprites.ActionPCSkill
	}
}

func (s SkillActionSpec) ActorAnimationForActorWithResources(manager *res.Manager, actor worldstate.Actor, started time.Time, duration time.Duration) scene.ActorAnimation {
	if !s.Defined {
		s = defaultSkillActionSpec
	}
	// Clown and Gypsy hold their generic skill pose; explicit dance and
	// attack sequences keep their own frames and timing.
	if s.Action == SkillActorActionSkill && !s.HasFrame && (actor.Job == db.JobBardH || actor.Job == db.JobDancerH) {
		s.HasFrame, s.Play, s.Duration = true, false, 400*time.Millisecond
	}
	if s.RandomReverse && started.UnixNano()&1 != 0 {
		copy := cloneSkillActionSpec(&s)
		last := copy
		for last.Next != nil {
			last = last.Next
		}
		end := s.Frame + last.Frame
		for step := copy; step != nil; step = step.Next {
			step.Frame = end - step.Frame
			step.RandomReverse = false
		}
		s = *copy
	}
	if s.Duration > 0 {
		duration = s.Duration
	}
	if s.Delay > 0 && !started.IsZero() {
		started = started.Add(s.Delay)
	}
	anim := scene.ActorAnimation{
		ActionFamily: s.ActionFamilyForActorWithResources(manager, actor),
		Started:      started,
		StartDelay:   s.Delay,
		Duration:     duration,
		Combo:        s.Combo,
		Loop:         s.Repeat,
		PingPong:     s.PingPong,
		Play:         s.Play,
		HasPlay:      true,
	}
	if s.HasFrame {
		anim.FrameOffset = s.Frame
		anim.HasFrameOffset = true
		if !s.Play {
			anim.FixedMotion = s.Frame
			anim.HasFixedMotion = true
		}
	}
	if s.Length > 0 {
		anim.Length = s.Length
		anim.HasLength = true
	}
	if s.Speed > 0 {
		anim.Speed = s.Speed
		anim.HasSpeed = true
	}
	if s.Next != nil && (!s.Next.isNaturalIdleFallback() || s.Repeat) {
		next := s.Next.ActorAnimationForActorWithResources(manager, actor, time.Time{}, duration)
		anim.Next = &next
	}
	return anim
}

func (s SkillActionSpec) isNaturalIdleFallback() bool {
	return s.Defined && s.Action == SkillActorActionIdle && s.Repeat && s.Play && s.Next == nil
}

func importedSkillActionSpec(skillID uint16) (SkillActionSpec, bool) {
	switch skillID {
	case db.SkillSGFeel, db.SkillSGHate:
		return newSkillActionSpec(SkillActorActionSkill, false, &IdleSkillActionSpec), true
	case db.SkillSGFusion:
		return IdleSkillActionSpec, true
	case db.SkillSGSunWarm, db.SkillSGMoonWarm, db.SkillSGStarWarm:
		return SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true, Frame: 3, Duration: time.Second}, true
	case db.SkillSGSunComfort, db.SkillSGMoonComfort, db.SkillSGStarComfort:
		return SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true, Frame: 2, Duration: time.Second}, true
	}
	if spec, ok := taekwonSkillActionSpec(skillID); ok {
		return spec, true
	}
	if db.IsSoulLinkSkill(skillID) {
		return SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true, Frame: 3, Duration: 2 * time.Second}, true
	}
	if spec, ok := dancerSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := bardSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := monkSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := gunslingerSkillActionSpec(skillID); ok {
		return spec, true
	}
	if spec, ok := ninjaSkillActionSpec(skillID); ok {
		return spec, true
	}
	if skillID == db.SkillASSonicblow {
		return sonicBlowSkillActionSpec(), true
	}
	switch db.SkillActions[skillID] {
	case db.SkillActionNone:
		return SkillActionSpec{Defined: true, Action: SkillActorActionNone}, true
	case db.SkillActionIdle:
		return IdleSkillActionSpec, true
	case db.SkillActionAttack:
		return AttackSkillActionSpec, true
	case db.SkillActionAttack1:
		return newSkillActionSpec(SkillActorActionAttack1, false, &IdleSkillActionSpec), true
	case db.SkillActionAttack2:
		return newSkillActionSpec(skillActorActionAttack2, false, &IdleSkillActionSpec), true
	case db.SkillActionAttack3:
		return newSkillActionSpec(SkillActorActionAttack3, false, &IdleSkillActionSpec), true
	case db.SkillActionSkill, db.SkillActionAction:
		return defaultSkillActionSpec, true
	case db.SkillActionAttackFixedFrame:
		spec := newSkillActionSpec(SkillActorActionAttack, false, nil)
		spec.HasFrame = true
		spec.Frame = 0
		spec.Play = false
		return spec, true
	case db.SkillActionPickup:
		return newSkillActionSpec(SkillActorActionPickup, false, &IdleSkillActionSpec), true
	case db.SkillActionReadyfight:
		return ReadyFightSkillActionSpec, true
	case db.SkillActionDance:
		return DanceSkillActionSpec(), true
	default:
		return SkillActionSpec{}, false
	}
}

func SkillAction(skillID uint16) SkillActionSpec {
	spec := SkillEffectSpecFor(skillID).Action
	if !spec.Defined {
		return defaultSkillActionSpec
	}
	return spec
}

func sonicBlowSkillActionSpec() SkillActionSpec {
	next := newSkillActionSpec(SkillActorActionReadyFight, true, nil)
	for hit := 8; hit >= 1; hit-- {
		spec := newSkillActionSpec(SkillActorActionAttack, false, &next)
		if hit > 1 {
			spec.Speed = 30 * time.Millisecond
		}
		next = spec
	}
	return next
}

func bardSkillActionSpec(skillID uint16) (SkillActionSpec, bool) {
	if skillID == db.SkillBaFrostjoke || skillID == db.SkillBaPangvoice {
		spec := newSkillActionSpec(SkillActorActionSkill, true, &IdleSkillActionSpec)
		spec.Frame, spec.HasFrame, spec.Length = 4, true, 2
		spec.Speed, spec.Duration = 10*db.EffectFrameDuration, 1800*time.Millisecond
		return spec, true
	}
	if skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun {
		spec := DanceSkillActionSpec()
		spec.PingPong = false // ST_SING loops 1,2,3; ST_DANCE plays 1,2,3,2.
		return spec, true
	}
	return SkillActionSpec{}, false
}

func dancerSkillActionSpec(skillID uint16) (SkillActionSpec, bool) {
	if skillID == db.SkillDCScream {
		return SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true, Frame: 0, Duration: 1200 * time.Millisecond}, true
	}
	return SkillActionSpec{}, false
}

func gunslingerSkillActionSpec(id uint16) (SkillActionSpec, bool) {
	action, first, last, duration := SkillActorActionSkill, 0, 3, time.Second
	switch id {
	case db.SkillGSFling, db.SkillGSMadnesscancel, db.SkillGSAdjustment, db.SkillGSIncreasing, db.SkillGSDesperado, db.SkillGSGatlingfever:
	case db.SkillGSGlittering:
		first, last, duration = 4, 5, 500*time.Millisecond
	case db.SkillGSCracker, db.SkillGSGrounddrift:
		action, last, duration = SkillActorActionAttack1, 2, 500*time.Millisecond
	case db.SkillGSTripleaction, db.SkillGSBullseye, db.SkillGSMagicalbullet, db.SkillGSTracking, db.SkillGSDisarm,
		db.SkillGSPiercingshot, db.SkillGSRapidshower, db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack:
		return AttackSkillActionSpec, true
	default:
		return SkillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := SkillActionSpec{Defined: true, Action: action, HasFrame: true, Frame: last, Duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = SkillActionSpec{Defined: true, Action: action, HasFrame: true, Frame: frame, Duration: step, Next: cloneSkillActionSpec(&next)}
	}
	next.RandomReverse = first == 0 && last == 3
	return next, true
}

func GunslingerCastAction(id uint16, now time.Time) (SkillActionSpec, bool) {
	spec := SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true, Frame: 1 + int(now.UnixNano()&1), Duration: 500 * time.Millisecond}
	switch id {
	case db.SkillGSBullseye, db.SkillGSGatlingfever, db.SkillGSTracking, db.SkillGSMadnesscancel,
		db.SkillGSAdjustment, db.SkillGSDisarm, db.SkillGSPiercingshot, db.SkillGSRapidshower:
	case db.SkillGSDust, db.SkillGSFullbuster, db.SkillGSSpreadattack, db.SkillGSGrounddrift:
		spec.Action, spec.Frame = SkillActorActionAttack1, 0
	default:
		return SkillActionSpec{}, false
	}
	return spec, true
}

func ninjaSkillActionSpec(id uint16) (SkillActionSpec, bool) {
	first, last, duration := 0, 1, 500*time.Millisecond
	switch id {
	case db.SkillNJSyuriken, db.SkillNJKunai, db.SkillNJHuuma, db.SkillNJZenynage,
		db.SkillNJUtsusemi, db.SkillNJKouenka, db.SkillNJHyousyouraku,
		db.SkillNJHuujin, db.SkillNJRaigekisai, db.SkillNJKamaitachi, db.SkillNJNen:
		first, last, duration = 2, 5, time.Second
	case db.SkillNJTatamigaeshi, db.SkillNJKasumikiri, db.SkillNJShadowjump,
		db.SkillNJBunsinjyutsu, db.SkillNJKaensin, db.SkillNJBakuenryu, db.SkillNJHyousensou, db.SkillNJSuiton:
	case db.SkillNJIssen:
		return defaultSkillActionSpec, true
	case db.SkillNJKirikage:
		return newSkillActionSpec(skillActorActionAttack2, false, &IdleSkillActionSpec), true
	default:
		return SkillActionSpec{}, false
	}
	step := 6 * db.EffectFrameDuration
	next := SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true,
		Frame: last, Duration: duration - time.Duration(last-first)*step}
	for frame := last - 1; frame >= first; frame-- {
		next = SkillActionSpec{Defined: true, Action: SkillActorActionSkill, HasFrame: true,
			Frame: frame, Duration: step, Next: cloneSkillActionSpec(&next)}
	}
	return next, true
}

func DanceSkillActionSpec() SkillActionSpec {
	spec := newSkillActionSpec(SkillActorActionSkill, true, nil)
	spec.HasFrame = true
	spec.Frame = 1
	spec.Length = 3
	spec.Speed = 10 * db.EffectFrameDuration
	spec.PingPong = true
	return spec
}

func taekwonSkillActionSpec(skillID uint16) (SkillActionSpec, bool) {
	pose := func(action skillActorAction, frame int, duration time.Duration) SkillActionSpec {
		return SkillActionSpec{Defined: true, Action: action, HasFrame: true, Frame: frame, Duration: duration}
	}
	switch skillID {
	case db.SkillTKRun:
		return IdleSkillActionSpec, true
	case db.SkillTKReadystorm:
		return pose(SkillActorActionSkill, 0, 2*time.Second), true
	case db.SkillTKStormkick:
		return pose(SkillActorActionSkill, 1, 2*time.Second), true
	case db.SkillTKReadydown:
		return pose(SkillActorActionSkill, 2, 2*time.Second), true
	case db.SkillTKReadyturn:
		return pose(SkillActorActionSkill, 3, 2*time.Second), true
	case db.SkillTKReadycounter:
		return pose(SkillActorActionSkill, 4, 2*time.Second), true
	case db.SkillTKDodge:
		return pose(SkillActorActionPickup, 1, 2*time.Second), true
	case db.SkillTKJumpkick:
		// EF_JUMPKICK replaces the three-second pose at tick five with a 25 ms hold.
		return pose(SkillActorActionSkill, 5, 5*db.EffectFrameDuration+25*time.Millisecond), true
	case db.SkillTKDownkick:
		return TaekwonKickSequence(skillActorActionAttack2, [5]int{3, 5, 7, 8, 2}, 5), true
	case db.SkillTKTurnkick:
		return TaekwonKickSequence(SkillActorActionAttack3, [5]int{4, 5, 6, 7, 2}, 5), true
	case db.SkillTKCounter:
		return TaekwonKickSequence(SkillActorActionAttack1, [5]int{2, 3, 4, 4, 4}, 4), true
	case db.SkillTKHighjump:
		return TaekwonJumpSequence(false), true
	default:
		return SkillActionSpec{}, false
	}
}

// Native SetForceAnimation2 plays five selected ACT motions, then holds the last
// one until its timeout. The existing animation chain also permits the landing
// pose to finish in a different ACT action.
func TaekwonKickSequence(action skillActorAction, frames [5]int, ticks int) SkillActionSpec {
	step := time.Duration(ticks) * db.EffectFrameDuration
	next := SkillActionSpec{
		Defined: true, Action: action, HasFrame: true, Frame: frames[4],
		Duration: 500*time.Millisecond - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = SkillActionSpec{
			Defined: true, Action: action, HasFrame: true, Frame: frames[i],
			Duration: step, Next: cloneSkillActionSpec(&next),
		}
	}
	return next
}

func TaekwonJumpSequence(casting bool) SkillActionSpec {
	step := 10 * db.EffectFrameDuration
	action, frames := SkillActorActionPickup, [5]int{1, 1, 1, 1, 5}
	finalAction := skillActorActionWalk
	if casting {
		action, frames = SkillActorActionAttack1, [5]int{0, 0, 0, 1, 1}
		finalAction = action
	}
	next := SkillActionSpec{
		Defined: true, Action: finalAction, HasFrame: true, Frame: frames[4],
		Duration: 3*time.Second - 4*step,
	}
	for i := 3; i >= 0; i-- {
		next = SkillActionSpec{
			Defined: true, Action: action, HasFrame: true, Frame: frames[i],
			Duration: step, Next: cloneSkillActionSpec(&next),
		}
	}
	return next
}
