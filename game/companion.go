package game

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

const (
	aiMotionStand  = 0
	aiMotionMove   = 1
	aiMotionAttack = 2
	aiMotionDead   = 3
)

func applyCompanionActorEntry(ctx client.Context, entry network.ActorEntry) {
	if ctx.Session == nil || entry.ID == 0 || !entry.HasObjectType {
		return
	}
	switch entry.ObjectType {
	case actorObjectTypeHomunculus:
		ctx.Session.Homunculus.ID = entry.ID
		ctx.Session.Homunculus.Active = true
		ctx.Session.Homunculus.Job = entry.Job
	case actorObjectTypeMercenary:
		ctx.Session.Mercenary.ID = entry.ID
		ctx.Session.Mercenary.Active = true
		ctx.Session.Mercenary.Job = entry.Job
	}
}

func (m *WorldMode) applyHomunculusProperty(ctx client.Context, property network.HomunculusProperty) {
	if ctx.Session == nil {
		return
	}
	hom := &ctx.Session.Homunculus
	hom.Active = true
	hom.Name = property.Name
	hom.Flags = property.Flags
	hom.Level = property.Level
	hom.Hunger = property.Hunger
	hom.Intimacy = property.Intimacy
	hom.ItemID = property.ItemID
	hom.Attack = property.Attack
	hom.MagicAttack = property.MagicAttack
	hom.Hit = property.Hit
	hom.Critical = property.Critical
	hom.Defense = property.Defense
	hom.MagicDefense = property.MagicDefense
	hom.Flee = property.Flee
	hom.ASPD = property.ASPD
	hom.HP = property.HP
	hom.MaxHP = property.MaxHP
	hom.SP = property.SP
	hom.MaxSP = property.MaxSP
	hom.Exp = property.Exp
	hom.MaxExp = property.MaxExp
	hom.Skills.Points = property.SkillPoints
	hom.AttackRange = property.AttackRange
	if hom.ID == 0 {
		hom.ID = findCompanionActorID(ctx, actorObjectTypeHomunculus)
	}
	m.applyCompanionLife(ctx, hom.ID, property.HP, property.MaxHP, property.SP, property.MaxSP, property.AttackRange)
	glog.Debugf("homunculus property id=%d name=%q level=%d hp=%d/%d sp=%d/%d hunger=%d intimacy=%d range=%d skills=%d", hom.ID, hom.Name, hom.Level, hom.HP, hom.MaxHP, hom.SP, hom.MaxSP, hom.Hunger, hom.Intimacy, hom.AttackRange, hom.Skills.Points)
}

func (m *WorldMode) applyHomunculusStateChange(ctx client.Context, change network.HomunculusStateChange) {
	if ctx.Session == nil || change.GID == 0 {
		return
	}
	hom := &ctx.Session.Homunculus
	hom.ID = change.GID
	hom.Active = true
	switch change.State {
	case 0:
		// Pre-init: robrowser uses this packet to learn Session.homunId.
	case 1:
		hom.Intimacy = int(change.Data)
	case 2:
		hom.Hunger = int(change.Data)
	}
	glog.Debugf("homunculus state id=%d type=%d state=%d data=%d", change.GID, change.Type, change.State, change.Data)
}

func applyHomunculusSkillInfoList(ctx client.Context, list network.HomunculusSkillInfoList) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Homunculus.Skills.List = ctx.Session.Homunculus.Skills.List[:0]
	for _, skill := range list.Skills {
		ctx.Session.Homunculus.Skills.List = append(ctx.Session.Homunculus.Skills.List, sessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	glog.Debugf("homunculus skill list received count=%d points=%d", len(ctx.Session.Homunculus.Skills.List), ctx.Session.Homunculus.Skills.Points)
}

func applyHomunculusSkillInfoUpdate(ctx client.Context, update network.HomunculusSkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertCompanionSkill(&ctx.Session.Homunculus, sessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	glog.Debugf("homunculus skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
}

func (m *WorldMode) applyMercenaryProperty(ctx client.Context, property network.MercenaryProperty) {
	if ctx.Session == nil {
		return
	}
	merc := &ctx.Session.Mercenary
	merc.Active = true
	if property.ID != 0 {
		merc.ID = property.ID
	}
	if merc.ID == 0 {
		merc.ID = findCompanionActorID(ctx, actorObjectTypeMercenary)
	}
	if property.Name != "" {
		merc.Name = property.Name
	}
	merc.Level = property.Level
	merc.Faith = property.Faith
	merc.SummonCount = property.SummonCount
	merc.Calls = property.Calls
	merc.Kills = property.Kills
	merc.ExpireTick = property.ExpireTick
	merc.Attack = property.Attack
	merc.MagicAttack = property.MagicAttack
	merc.Hit = property.Hit
	merc.Critical = property.Critical
	merc.Defense = property.Defense
	merc.MagicDefense = property.MagicDefense
	merc.Flee = property.Flee
	merc.ASPD = property.ASPD
	merc.HP = property.HP
	merc.MaxHP = property.MaxHP
	merc.SP = property.SP
	merc.MaxSP = property.MaxSP
	merc.Exp = property.Exp
	if property.AttackRange > 0 {
		merc.AttackRange = property.AttackRange
	}
	m.applyCompanionLife(ctx, merc.ID, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.AttackRange)
	glog.Debugf("mercenary property id=%d name=%q level=%d hp=%d/%d sp=%d/%d faith=%d kills=%d range=%d", merc.ID, merc.Name, merc.Level, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.Faith, merc.Kills, merc.AttackRange)
}

func (m *WorldMode) applyMercenaryParamChange(ctx client.Context, change network.MercenaryParamChange) {
	if ctx.Session == nil || !ctx.Session.Mercenary.Active {
		return
	}
	merc := &ctx.Session.Mercenary
	value := int(change.Value)
	switch change.Param {
	case network.StatusHP:
		merc.HP = value
	case network.StatusMaxHP:
		merc.MaxHP = value
	case network.StatusSP:
		merc.SP = value
	case network.StatusMaxSP:
		merc.MaxSP = value
	default:
		glog.Debugf("mercenary param ignored param=%d value=%d", change.Param, value)
		return
	}
	m.applyCompanionLife(ctx, merc.ID, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP, merc.AttackRange)
	glog.Debugf("mercenary param id=%d param=%d value=%d hp=%d/%d sp=%d/%d", merc.ID, change.Param, value, merc.HP, merc.MaxHP, merc.SP, merc.MaxSP)
}

func applyMercenarySkillInfoList(ctx client.Context, list network.MercenarySkillInfoList) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Mercenary.Skills.List = ctx.Session.Mercenary.Skills.List[:0]
	for _, skill := range list.Skills {
		ctx.Session.Mercenary.Skills.List = append(ctx.Session.Mercenary.Skills.List, sessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	glog.Debugf("mercenary skill list received count=%d", len(ctx.Session.Mercenary.Skills.List))
}

func applyMercenarySkillInfoUpdate(ctx client.Context, update network.MercenarySkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertCompanionSkill(&ctx.Session.Mercenary, sessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	glog.Debugf("mercenary skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
}

func upsertCompanionSkill(companion *session.Companion, skill session.Skill) {
	if companion == nil {
		return
	}
	for i := range companion.Skills.List {
		if companion.Skills.List[i].ID != skill.ID {
			continue
		}
		companion.Skills.List[i] = skill
		return
	}
	companion.Skills.List = append(companion.Skills.List, skill)
}

func findCompanionActorID(ctx client.Context, objectType uint8) uint32 {
	if ctx.World == nil {
		return 0
	}
	for id, actor := range ctx.World.Actors {
		if actor.HasObjectType && actor.ObjectType == objectType {
			return id
		}
	}
	return 0
}

func (m *WorldMode) applyCompanionLife(ctx client.Context, id uint32, hp, maxHP, sp, maxSP, attackRange int) {
	if id == 0 || ctx.World == nil {
		return
	}
	if actor, ok := ctx.World.Actors[id]; ok {
		if attackRange > 0 {
			actor.AttackRange = attackRange
		}
		ctx.World.Actors[id] = actor
	}
	if maxHP <= 0 && maxSP <= 0 {
		return
	}
	if m.actorLife == nil {
		m.actorLife = make(map[uint32]actorLife)
	}
	life := m.actorLife[id]
	if maxHP > 0 {
		life.hp = clampInt(hp, 0, maxHP)
		life.maxHP = maxHP
	}
	if maxSP > 0 {
		life.sp = clampInt(sp, 0, maxSP)
		life.maxSP = maxSP
		life.hasSP = true
	}
	life.updatedAt = time.Now()
	m.actorLife[id] = life
}

func (m *WorldMode) setActorAIMotion(ctx client.Context, id uint32, motion int, targetID uint32) {
	if id == 0 || ctx.World == nil {
		return
	}
	if isLocalActor(ctx, id) {
		ctx.World.Player.AIMotion = motion
		ctx.World.Player.HasAIMotion = true
		ctx.World.Player.AITargetID = targetID
		return
	}
	actor, ok := ctx.World.Actors[id]
	if !ok {
		return
	}
	actor.AIMotion = motion
	actor.HasAIMotion = true
	actor.AITargetID = targetID
	ctx.World.Actors[id] = actor
}
