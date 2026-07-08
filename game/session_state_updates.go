package game

import (
	"github.com/kivutar/goro/client"
	"log"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
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
	log.Printf("status snapshot points=%d str=%d agi=%d vit=%d int=%d dex=%d luk=%d", snapshot.Points, snapshot.Str, snapshot.Agi, snapshot.Vit, snapshot.Int, snapshot.Dex, snapshot.Luk)
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
	log.Printf("skill list received count=%d points=%d", len(ctx.Session.Skills.List), ctx.Session.Skills.Points)
}

func applySkillInfoUpdate(ctx client.Context, update network.SkillInfoUpdate) {
	if ctx.Session == nil {
		return
	}
	upsertSessionSkill(ctx.Session, sessionSkillFromNetworkWithResources(ctx.Resources, update.Skill))
	log.Printf("skill update id=%d level=%d sp=%d range=%d upgradable=%t", update.Skill.ID, update.Skill.Level, update.Skill.SPCost, update.Skill.Range, update.Skill.Upgradable)
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

func applyFriendsList(ctx client.Context, friends []network.Friend) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Friends.List = ctx.Session.Friends.List[:0]
	for _, friend := range friends {
		ctx.Session.Friends.List = append(ctx.Session.Friends.List, sessionFriendFromNetwork(friend))
	}
	log.Printf("friend list received count=%d", len(ctx.Session.Friends.List))
}

func applyFriendState(ctx client.Context, state network.FriendState) {
	if ctx.Session == nil {
		return
	}
	for i := range ctx.Session.Friends.List {
		friend := &ctx.Session.Friends.List[i]
		if friend.AccountID != state.AccountID || friend.CharID != state.CharID {
			continue
		}
		friend.State = state.State
		log.Printf("friend state aid=%d gid=%d name=%q state=%d online=%t", friend.AccountID, friend.CharID, friend.Name, friend.State, friend.Online())
		return
	}
	ctx.Session.Friends.List = append(ctx.Session.Friends.List, session.Friend{
		AccountID: state.AccountID,
		CharID:    state.CharID,
		State:     state.State,
	})
	log.Printf("friend state aid=%d gid=%d state=%d online=%t", state.AccountID, state.CharID, state.State, state.State == 0)
}

func applyFriendAddResult(ctx client.Context, result network.FriendAddResult) {
	if ctx.Session == nil || result.Result != 0 {
		return
	}
	upsertSessionFriend(ctx.Session, session.Friend{
		AccountID: result.AccountID,
		CharID:    result.CharID,
		Name:      result.Name,
		State:     0,
	})
	log.Printf("friend added aid=%d gid=%d name=%q", result.AccountID, result.CharID, result.Name)
}

func applyFriendDelete(ctx client.Context, deleted network.FriendDelete) {
	if ctx.Session == nil {
		return
	}
	for i := range ctx.Session.Friends.List {
		friend := ctx.Session.Friends.List[i]
		if friend.AccountID != deleted.AccountID || friend.CharID != deleted.CharID {
			continue
		}
		ctx.Session.Friends.List = append(ctx.Session.Friends.List[:i], ctx.Session.Friends.List[i+1:]...)
		log.Printf("friend deleted aid=%d gid=%d name=%q", friend.AccountID, friend.CharID, friend.Name)
		return
	}
}

func sessionFriendFromNetwork(friend network.Friend) session.Friend {
	return session.Friend{
		AccountID: friend.AccountID,
		CharID:    friend.CharID,
		Name:      friend.Name,
		State:     friend.State,
	}
}

func upsertSessionFriend(s *session.Session, friend session.Friend) {
	for i := range s.Friends.List {
		if s.Friends.List[i].AccountID != friend.AccountID || s.Friends.List[i].CharID != friend.CharID {
			continue
		}
		s.Friends.List[i] = friend
		return
	}
	s.Friends.List = append(s.Friends.List, friend)
}
