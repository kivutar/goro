package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

const (
	defaultPlayerMoveSpeedMS = 150
	skillPushCart            = 39
)

func applyStatusSnapshot(ctx client.Context, snapshot network.StatusSnapshot) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Stats.Points = snapshot.Points
	setSessionStat(ctx.Session, network.StatusStr, snapshot.Str)
	setSessionStat(ctx.Session, network.StatusAgi, snapshot.Agi)
	setSessionStat(ctx.Session, network.StatusVit, snapshot.Vit)
	setSessionStat(ctx.Session, network.StatusInt, snapshot.Int)
	setSessionStat(ctx.Session, network.StatusDex, snapshot.Dex)
	setSessionStat(ctx.Session, network.StatusLuk, snapshot.Luk)
	ctx.Session.Stats.StrBonus = snapshot.StrBonus
	ctx.Session.Stats.AgiBonus = snapshot.AgiBonus
	ctx.Session.Stats.VitBonus = snapshot.VitBonus
	ctx.Session.Stats.IntBonus = snapshot.IntBonus
	ctx.Session.Stats.DexBonus = snapshot.DexBonus
	ctx.Session.Stats.LukBonus = snapshot.LukBonus
	ctx.Session.Stats.StrCost = snapshot.StrCost
	ctx.Session.Stats.AgiCost = snapshot.AgiCost
	ctx.Session.Stats.VitCost = snapshot.VitCost
	ctx.Session.Stats.IntCost = snapshot.IntCost
	ctx.Session.Stats.DexCost = snapshot.DexCost
	ctx.Session.Stats.LukCost = snapshot.LukCost
	ctx.Session.Stats.Attack = snapshot.Attack
	ctx.Session.Stats.AttackBonus = snapshot.AttackBonus
	ctx.Session.Stats.MatkMin = snapshot.MatkMin
	ctx.Session.Stats.MatkMax = snapshot.MatkMax
	ctx.Session.Stats.Defense = snapshot.Defense
	ctx.Session.Stats.DefenseBonus = snapshot.DefenseBonus
	ctx.Session.Stats.MDefense = snapshot.MDefense
	ctx.Session.Stats.MDefenseBonus = snapshot.MDefenseBonus
	ctx.Session.Stats.Hit = snapshot.Hit
	ctx.Session.Stats.Flee = snapshot.Flee
	ctx.Session.Stats.FleeBonus = snapshot.FleeBonus
	ctx.Session.Stats.Critical = snapshot.Critical
	ctx.Session.Stats.ASPD = snapshot.ASPD
	ctx.Session.Stats.ASPDBonus = snapshot.ASPDBonus
	glog.Debugf("status snapshot points=%d str=%d agi=%d vit=%d int=%d dex=%d luk=%d", snapshot.Points, snapshot.Str, snapshot.Agi, snapshot.Vit, snapshot.Int, snapshot.Dex, snapshot.Luk)
}

func setSessionStat(s *session.Session, statusID uint16, value int) {
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}
	switch statusID {
	case network.StatusStr:
		s.Stats.Str = value
		s.Selected.Str = uint8(value)
	case network.StatusAgi:
		s.Stats.Agi = value
		s.Selected.Agi = uint8(value)
	case network.StatusVit:
		s.Stats.Vit = value
		s.Selected.Vit = uint8(value)
	case network.StatusInt:
		s.Stats.Int = value
		s.Selected.Int = uint8(value)
	case network.StatusDex:
		s.Stats.Dex = value
		s.Selected.Dex = uint8(value)
	case network.StatusLuk:
		s.Stats.Luk = value
		s.Selected.Luk = uint8(value)
	}
}

func setSessionStatCost(s *session.Session, statusID uint16, value int) {
	if s == nil {
		return
	}
	switch statusID {
	case network.StatusUStr:
		s.Stats.StrCost = value
	case network.StatusUAgi:
		s.Stats.AgiCost = value
	case network.StatusUVit:
		s.Stats.VitCost = value
	case network.StatusUInt:
		s.Stats.IntCost = value
	case network.StatusUDex:
		s.Stats.DexCost = value
	case network.StatusULuk:
		s.Stats.LukCost = value
	}
}

func applySkillInfoList(ctx client.Context, list network.SkillInfoList) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Skills.List = ctx.Session.Skills.List[:0]
	for _, skill := range list.Skills {
		ctx.Session.Skills.List = append(ctx.Session.Skills.List, sessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	refreshLocalPlayerMoveSpeed(ctx)
	glog.Debugf("skill list received count=%d points=%d", len(ctx.Session.Skills.List), ctx.Session.Skills.Points)
}

func applySkillInfoUpdate(ctx client.Context, update network.SkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertSessionSkill(ctx.Session, sessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	refreshLocalPlayerMoveSpeed(ctx)
	glog.Debugf("skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
}

func upsertSessionSkill(s *session.Session, skill session.Skill) {
	for i := range s.Skills.List {
		if s.Skills.List[i].ID != skill.ID {
			continue
		}
		s.Skills.List[i] = skill
		return
	}
	s.Skills.List = append(s.Skills.List, skill)
}

func refreshLocalPlayerMoveSpeed(ctx client.Context) {
	if ctx.World == nil || ctx.Session == nil {
		return
	}
	speed := defaultPlayerMoveSpeedMS
	if ctx.Session.Movement.HasServerSpeed && ctx.Session.Movement.ServerSpeed > 0 {
		speed = ctx.Session.Movement.ServerSpeed
	}
	if localPlayerHasCart(ctx) {
		level := sessionSkillLevel(ctx.Session, skillPushCart)
		if level > 0 && level < 10 {
			speed += speed * (50 - 5*level) / 100
		}
	}
	ctx.World.Player.Speed = speed
}

func localPlayerHasCart(ctx client.Context) bool {
	if ctx.World != nil && ctx.World.Player.HasCartState {
		return ctx.World.Player.HasCart
	}
	if ctx.Session == nil {
		return false
	}
	if ctx.Session.SelectedCharacter().Option&actorEffectCartMask != 0 {
		return true
	}
	return ctx.Session.Cart.MaxAmount > 0 || len(ctx.Session.Cart.Items) > 0
}

func sessionSkillLevel(s *session.Session, skillID uint16) int {
	if s == nil {
		return 0
	}
	for _, skill := range s.Skills.List {
		if skill.ID == skillID {
			return skill.Level
		}
	}
	return 0
}
