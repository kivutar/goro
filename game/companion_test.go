package game

import (
	"testing"

	"github.com/kivutar/goro/game/pet"

	"github.com/gogpu/gpucontext"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestHomunculusParamChangeUpdatesSessionAndLife(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:          300,
			Active:      true,
			Level:       12,
			Hunger:      37,
			HP:          10,
			MaxHP:       20,
			SP:          5,
			MaxSP:       10,
			AttackRange: 2,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusHP, Value: 18})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusMaxSP, Value: 44})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseExp, Value: 5_000_000_000})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusNextBaseExp, Value: 6_000_000_000})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusSkillPoint, Value: 7})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusSpeed, Value: 150})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 13})

	if sess.Homunculus.HP != 18 || sess.Homunculus.MaxSP != 44 || sess.Homunculus.Level != 13 || sess.Homunculus.Exp != 5_000_000_000 || sess.Homunculus.MaxExp != 6_000_000_000 || sess.Homunculus.Skills.Points != 7 {
		t.Fatalf("homunculus session = %+v", sess.Homunculus)
	}
	if life := mode.battleSystem.ActorLife[300]; life.HP != 18 || life.MaxHP != 20 || life.SP != 5 || life.MaxSP != 44 || !life.HasSP {
		t.Fatalf("actor life = %+v", life)
	}
	if actor := ctx.World.Actors[300]; actor.Speed != 150 || actor.AttackRange != 2 {
		t.Fatalf("actor = %+v", actor)
	}
}

func TestHomunculusPropertyStoresDisplayLifeAndHunger(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, Name: "Vanilmirth2", HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusProperty(ctx, network.HomunculusProperty{
		Name:        "Pipou",
		Level:       45,
		Hunger:      62,
		HP:          123,
		MaxHP:       456,
		SP:          78,
		MaxSP:       90,
		AttackRange: 4,
	})

	life, ok := mode.battle().ActorLifeForDisplay(ctx, ctx.World.Actors[300])
	if !ok {
		t.Fatal("homunculus display life missing")
	}
	if life.HP != 123 || life.MaxHP != 456 || life.SP != 78 || life.MaxSP != 90 || !life.HasSP || life.Hunger != 62 || life.MaxHunger != 100 || !life.HasHunger || !life.Friendly {
		t.Fatalf("homunculus display life = %+v", life)
	}
	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" || actor.AttackRange != 4 {
		t.Fatalf("homunculus actor = %+v", actor)
	}
}

func TestHomunculusActorEntryUsesKnownSessionName(t *testing.T) {
	sess := &session.Session{
		Homunculus: session.Companion{
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World:   worldstate.New(),
	}

	actors.UpsertNetworkActor(ctx, network.ActorEntry{
		ID:            300,
		Job:           6002,
		X:             10,
		Y:             20,
		HasObjectType: true,
		ObjectType:    scene.ActorObjectTypeHomunculus,
	})

	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" {
		t.Fatalf("homunculus actor name = %q, want session name", actor.Name)
	}
}

func TestHomunculusActorEntryDoesNotUseSessionNameForDifferentID(t *testing.T) {
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:   300,
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World:   worldstate.New(),
	}

	actors.UpsertNetworkActor(ctx, network.ActorEntry{
		ID:            301,
		Job:           6002,
		X:             10,
		Y:             20,
		HasObjectType: true,
		ObjectType:    scene.ActorObjectTypeHomunculus,
	})

	if actor := ctx.World.Actors[301]; actor.Name != "" {
		t.Fatalf("other homunculus actor name = %q, want empty before server ack", actor.Name)
	}
}

func TestHomunculusStateChangeAppliesKnownSessionName(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			Name: "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, Name: "Vanilmirth2", HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusStateChange(ctx, network.HomunculusStateChange{GID: 300, State: 0})

	if actor := ctx.World.Actors[300]; actor.Name != "Pipou" {
		t.Fatalf("homunculus actor name = %q, want known session name", actor.Name)
	}
}

func TestHomunculusStateChangeUpdatesDisplayHunger(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			HP:     50,
			MaxHP:  100,
			SP:     10,
			MaxSP:  20,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusStateChange(ctx, network.HomunculusStateChange{GID: 300, State: 2, Data: 24})

	life, ok := mode.battle().ActorLifeForDisplay(ctx, ctx.World.Actors[300])
	if !ok {
		t.Fatal("homunculus display life missing")
	}
	if sess.Homunculus.Hunger != 24 || life.Hunger != 24 || life.MaxHunger != 100 || !life.HasHunger {
		t.Fatalf("homunculus hunger session=%+v life=%+v", sess.Homunculus, life)
	}
}

func TestHomunculusParamLevelUpAddsHoUpEffect(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Level:  9,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})
	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.HoUp || effect.ActorID != 300 {
		t.Fatalf("world effect = %+v", effect)
	}
}

func TestHomunculusNotifyEffect2AddsHoUpEffectWithoutSound(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 300, EffectID: effects.HoUp})

	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.EffectID != effects.HoUp || effect.ActorID != 300 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for robr EF_HO_UP", mode.scheduledSounds)
	}
}

func TestHomunculusParamInitialLevelDoesNotAddHoUpEffect(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, X: 12, Y: 34, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.companions().ApplyHomunculusParamChange(ctx, network.HomunculusParamChange{Param: network.StatusBaseLevel, Value: 10})

	if len(mode.effects().Active) != 0 {
		t.Fatalf("world effects = %+v, want none for initial level load", mode.effects().Active)
	}
}

func TestPendingHomunculusDeleteVanishClearsSession(t *testing.T) {
	mode := NewWorldMode()
	mode.companionSystem.HomunculusDeleteID = 300
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Name:   "Pipou",
			HP:     1,
			MaxHP:  10,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}
	mode.ui.companions.HomunculusInfo.OpenInfo(ctx, sess.Homunculus)
	mode.ui.companions.HomunculusSkill.Open(ctx, nil)

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actors.ActorVanishOutOfSight})

	if sess.Homunculus.Active || sess.Homunculus.ID != 0 || sess.Homunculus.Name != "" {
		t.Fatalf("homunculus session after delete = %+v", sess.Homunculus)
	}
	if mode.companionSystem.HomunculusDeleteID != 0 {
		t.Fatalf("pending delete id = %d, want 0", mode.companionSystem.HomunculusDeleteID)
	}
	if _, ok := ctx.World.Actors[300]; ok {
		t.Fatal("deleted homunculus actor remained in world")
	}
	if mode.ui.companions.HomunculusInfo.IsOpen() {
		t.Fatal("homunculus info window stayed open after delete")
	}
	if mode.ui.companions.HomunculusSkill.IsOpen() {
		t.Fatal("homunculus skill window stayed open after delete")
	}
}

func TestHomunculusVanishWithoutDeleteKeepsSession(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Homunculus: session.Companion{
			ID:     300,
			Active: true,
			Name:   "Pipou",
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				300: {ID: 300, HasObjectType: true, ObjectType: scene.ActorObjectTypeHomunculus},
			},
		},
	}

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 300, Reason: actors.ActorVanishOutOfSight})

	if !sess.Homunculus.Active || sess.Homunculus.ID != 300 {
		t.Fatalf("homunculus session should survive non-delete vanish: %+v", sess.Homunculus)
	}
}

func TestApplyMercenaryParamChangeUpdatesRobrowserStatusFields(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:          400,
			Active:      true,
			HP:          100,
			MaxHP:       200,
			SP:          30,
			MaxSP:       60,
			AttackRange: 2,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary},
			},
		},
	}

	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusHP, Value: 77})
	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusSP, Value: 12})
	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercFlee, Value: 141})
	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercKills, Value: 9})
	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusMercFaith, Value: 321})

	if got := sess.Mercenary.HP; got != 77 {
		t.Fatalf("mercenary hp = %d, want 77", got)
	}
	if got := sess.Mercenary.SP; got != 12 {
		t.Fatalf("mercenary sp = %d, want 12", got)
	}
	if got := sess.Mercenary.Flee; got != 141 {
		t.Fatalf("mercenary flee = %d, want 141", got)
	}
	if got := sess.Mercenary.Kills; got != 9 {
		t.Fatalf("mercenary kills = %d, want 9", got)
	}
	if got := sess.Mercenary.Faith; got != 321 {
		t.Fatalf("mercenary faith = %d, want 321", got)
	}
	if life, ok := mode.battleSystem.ActorLife[400]; !ok || life.HP != 77 || life.SP != 12 || !life.HasSP {
		t.Fatalf("actor life = %+v ok=%t, want updated mercenary life", life, ok)
	}
	if actor := ctx.World.Actors[400]; actor.AttackRange != 2 {
		t.Fatalf("actor attack range = %d, want 2", actor.AttackRange)
	}
}

func TestMercenaryPropertyStoresDisplayLife(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary},
			},
		},
	}

	mode.companions().ApplyMercenaryProperty(ctx, network.MercenaryProperty{
		ID:          400,
		Name:        "David",
		Level:       1,
		HP:          123,
		MaxHP:       456,
		SP:          78,
		MaxSP:       90,
		AttackRange: 2,
	})

	life, ok := mode.battle().ActorLifeForDisplay(ctx, ctx.World.Actors[400])
	if !ok {
		t.Fatal("mercenary display life missing")
	}
	if life.HP != 123 || life.MaxHP != 456 || life.SP != 78 || life.MaxSP != 90 || !life.HasSP || !life.Friendly {
		t.Fatalf("mercenary display life = %+v", life)
	}
	if actor := ctx.World.Actors[400]; actor.Name != "David" || actor.AttackRange != 2 {
		t.Fatalf("mercenary actor = %+v", actor)
	}
}

func TestMercenaryLifeForDisplayUsesCachedParamChanges(t *testing.T) {
	mode := NewWorldMode()
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
			HP:     100,
			MaxHP:  200,
			SP:     30,
			MaxSP:  60,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary},
			},
		},
	}

	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusHP, Value: 77})
	mode.companions().ApplyMercenaryParamChange(ctx, network.MercenaryParamChange{Param: network.StatusSP, Value: 12})

	life, ok := mode.battle().ActorLifeForDisplay(ctx, ctx.World.Actors[400])
	if !ok {
		t.Fatal("mercenary display life missing")
	}
	if life.HP != 77 || life.MaxHP != 200 || life.SP != 12 || life.MaxSP != 60 || !life.HasSP || !life.Friendly {
		t.Fatalf("mercenary display life = %+v", life)
	}
}

func TestPendingMercenaryDeleteVanishClearsSession(t *testing.T) {
	mode := NewWorldMode()
	mode.companionSystem.MercenaryDeleteID = 400
	sess := &session.Session{
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
			Name:   "Sword Mercenary",
			HP:     1,
			MaxHP:  10,
		},
	}
	ctx := client.Context{
		Session: sess,
		World: &worldstate.World{
			Actors: map[uint32]worldstate.Actor{
				400: {ID: 400, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary},
			},
		},
	}
	mode.ui.companions.MercenaryInfo.OpenInfo(ctx, sess.Mercenary)
	mode.ui.companions.MercenarySkill.Open(ctx, nil)

	mode.applyActorVanish(ctx, network.ActorVanish{ID: 400, Reason: actors.ActorVanishOutOfSight})

	if sess.Mercenary.Active || sess.Mercenary.ID != 0 || sess.Mercenary.Name != "" {
		t.Fatalf("mercenary session after delete = %+v", sess.Mercenary)
	}
	if mode.companionSystem.MercenaryDeleteID != 0 {
		t.Fatalf("pending delete id = %d, want 0", mode.companionSystem.MercenaryDeleteID)
	}
	if _, ok := ctx.World.Actors[400]; ok {
		t.Fatal("deleted mercenary actor remained in world")
	}
	if mode.ui.companions.MercenaryInfo.IsOpen() {
		t.Fatal("mercenary info window stayed open after delete")
	}
	if mode.ui.companions.MercenarySkill.IsOpen() {
		t.Fatal("mercenary skill window stayed open after delete")
	}
}

func TestMercenaryActionFamiliesUseGuildWeaponActions(t *testing.T) {
	archer := worldstate.Actor{Job: 6017, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
	lancer := worldstate.Actor{Job: 6027, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
	sword := worldstate.Actor{Job: 6037, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
	mob := worldstate.Actor{Job: 1002, HasObjectType: true, ObjectType: scene.ActorObjectTypeMob}

	if got := sprites.AttackActionFamily(nil, archer); got != sprites.ActionPCAttack2 {
		t.Fatalf("archer mercenary attack action = %d, want %d", got, sprites.ActionPCAttack2)
	}
	if got := sprites.AttackActionFamily(nil, lancer); got != sprites.ActionPCAttack3 {
		t.Fatalf("lancer mercenary attack action = %d, want %d", got, sprites.ActionPCAttack3)
	}
	if got := sprites.AttackActionFamily(nil, sword); got != sprites.ActionPCAttack2 {
		t.Fatalf("sword mercenary attack action = %d, want %d", got, sprites.ActionPCAttack2)
	}
	if got := hurtActionFamilyForActor(archer); got != sprites.ActionPCHurt {
		t.Fatalf("archer mercenary hurt action = %d, want %d", got, sprites.ActionPCHurt)
	}
	if got := actors.DeathActionFamilyForActor(archer); got != sprites.ActionNonPCDeath {
		t.Fatalf("archer mercenary death action = %d, want %d", got, sprites.ActionNonPCDeath)
	}
	if got := actors.DeathActionFamilyForActor(sword); got != sprites.ActionPCDeath {
		t.Fatalf("sword mercenary death action = %d, want %d", got, sprites.ActionPCDeath)
	}
	if got := skillCastActionFamilyForActor(sword, 0); got != sprites.ActionPCSkill {
		t.Fatalf("sword mercenary cast action = %d, want %d", got, sprites.ActionPCSkill)
	}
	if got := effects.ReadyFightSkillActionSpec.ActionFamilyForActorWithResources(nil, sword); got != sprites.ActionPCSkill {
		t.Fatalf("sword mercenary readyfight skill action = %d, want %d", got, sprites.ActionPCSkill)
	}
	if got := sprites.AttackActionFamily(nil, mob); got != sprites.ActionNonPCAttack {
		t.Fatalf("mob attack action = %d, want %d", got, sprites.ActionNonPCAttack)
	}
}

func TestMercenarySpriteKeyDerivesSexFromMercenaryResource(t *testing.T) {
	archer := worldstate.Actor{Job: 6017, Sex: 1, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
	sword := worldstate.Actor{Job: 6037, Sex: 0, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}

	if got := sprites.MercenaryAppearanceForActor(archer).Sex; got != 0 {
		t.Fatalf("archer mercenary sex = %d, want female resource sex 0", got)
	}
	if got := sprites.MercenaryAppearanceForActor(sword).Sex; got != 1 {
		t.Fatalf("sword mercenary sex = %d, want male resource sex 1", got)
	}
}

func TestMercenarySpriteKeyUsesGuildDefaultWeapon(t *testing.T) {
	cases := []struct {
		name string
		job  int16
		want int
	}{
		{name: "archer", job: 6017, want: db.WeaponBow},
		{name: "lancer", job: 6027, want: db.WeaponSpear},
		{name: "sword", job: 6037, want: db.WeaponSword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actor := worldstate.Actor{Job: tc.job, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
			if got := sprites.MercenaryAppearanceForActor(actor).Weapon; got != tc.want {
				t.Fatalf("default weapon = %d, want %d", got, tc.want)
			}
		})
	}

	actor := worldstate.Actor{Job: 6037, Weapon: 7, HasObjectType: true, ObjectType: scene.ActorObjectTypeMercenary}
	if got := sprites.MercenaryAppearanceForActor(actor).Weapon; got != 7 {
		t.Fatalf("explicit weapon = %d, want 7", got)
	}
}

func openCompanionInputTestWindow(ctx client.Context, m *WorldMode, name string) *gameui.Window {
	switch name {
	case "pet":
		m.ui.pet.Info.OpenInfo(ctx, network.PetProperty{Name: "Poring"})
		return &m.ui.pet.Info.Window
	case "homunculus":
		m.ui.companions.HomunculusInfo.OpenInfo(ctx, ctx.Session.Homunculus)
		return &m.ui.companions.HomunculusInfo.Window
	case "homunculus skills":
		m.ui.companions.HomunculusSkill.Open(ctx, m)
		return &m.ui.companions.HomunculusSkill.Window
	case "mercenary":
		m.ui.companions.MercenaryInfo.OpenInfo(ctx, ctx.Session.Mercenary)
		return &m.ui.companions.MercenaryInfo.Window
	default:
		m.ui.companions.MercenarySkill.Open(ctx, m)
		return &m.ui.companions.MercenarySkill.Window
	}
}

func TestCompanionWindowsAllowGameplayInput(t *testing.T) {
	for _, name := range []string{"pet", "homunculus", "homunculus skills", "mercenary", "mercenary skills"} {
		for _, control := range []string{"camera", "skill", "wasd"} {
			t.Run(name+"/"+control, func(t *testing.T) {
				ctx := wasdControlsTestContext(t)
				conn, server := testutil.NewConnection(t, 20080910)
				ctx.Network = conn
				ctx.UIManager = gameui.NewManager()
				ctx.Session.Homunculus.Active = true
				ctx.Session.Mercenary.Active = true
				mode := NewWorldMode()
				w := openCompanionInputTestWindow(ctx, mode, name)
				w.SetAutoPosition(100, 100)
				w.Publish(ctx)
				ctx.Input.SetMousePosition(700, 400)
				ctx.Input.EndFrame()
				switch control {
				case "camera":
					ctx.Input.SetMouseButton(input.MouseButtonRight, true)
					ctx.Input.SetMousePosition(720, 400)
				case "skill":
					ctx.Input.SetKey(input.KeyF1, true)
				case "wasd":
					loadKeyboardTestBot(t, ctx, mode)
					if !prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "w") {
						t.Fatal("WASD did not claim the movement key")
					}
				}
				if _, err := mode.Update(ctx); err != nil {
					t.Fatal(err)
				}
				if !w.IsOpen() {
					t.Fatal("gameplay input closed the companion window")
				}
				switch control {
				case "camera":
					if mode.camera.yawOffset == 0 {
						t.Fatal("open companion window blocked camera rotation outside the UI")
					}
					yaw := mode.camera.yawOffset
					ctx.Input.EndFrame()
					ctx.Input.SetMousePosition(120, 150)
					if !uiPointerBlocked(ctx) {
						t.Fatal("test pointer is not over the companion window")
					}
					if _, err := mode.Update(ctx); err != nil {
						t.Fatal(err)
					}
					if mode.camera.yawOffset != yaw {
						t.Fatal("camera rotated through the companion window")
					}
				case "skill":
					if mode.skillSystem.Pending.Skill.ID != ctx.Session.Skills.List[0].ID {
						t.Fatal("open companion window blocked the F1 skill shortcut")
					}
				case "wasd":
					walk, _ := network.BuildWalkToXYPacketForClientDate(10, 28, 20080910)
					readBotTestPackets(t, server, walk)
				}
			})
		}
	}
}

func TestCompanionRenameKeepsKeyboardInput(t *testing.T) {
	for _, name := range []string{"pet", "homunculus"} {
		t.Run(name, func(t *testing.T) {
			ctx := wasdControlsTestContext(t)
			ctx.Network = network.NewClient(20080910, false)
			t.Cleanup(ctx.Network.Close)
			ctx.Session.Homunculus.Active = true
			mode := NewWorldMode()
			loadKeyboardTestBot(t, ctx, mode)
			w := openCompanionInputTestWindow(ctx, mode, name)
			var field *textfield.Widget
			var find func(widget.Widget)
			find = func(node widget.Widget) {
				if f, ok := node.(*textfield.Widget); ok {
					field = f
				}
				for _, child := range node.Children() {
					find(child)
				}
			}
			find(w.Widget())
			if field == nil {
				t.Fatal("companion has no rename field")
			}
			field.SetFocused(true)
			if name == "pet" {
				field.SetText("Fluffy")
				mode.pets().ApplyProperty(ctx, network.PetProperty{Name: "Poring", Fullness: 80})
				mode.pets().ApplyStateChange(ctx, network.PetStateChange{Type: pet.StateHunger, Data: 79})
			}
			if prepareKeyboardTestText(mode, ctx, gpucontext.KeyW, "w") || mode.ui.console.Active() {
				t.Fatal("gameplay or chat stole text intended for the rename field")
			}
			ctx.Input.SetKey(input.KeyF1, true)
			if _, err := mode.Update(ctx); err != nil {
				t.Fatal(err)
			}
			if mode.skillSystem.Pending.Skill.ID != 0 || mode.bot.KeyboardAvailable {
				t.Fatal("gameplay keyboard input remained active during rename")
			}
			if mode.ui.nonConsoleKeyboardInputBlockedExcept(w) {
				t.Fatal("rename blocked the window's own toggle shortcut")
			}
			field.SetFocused(false)
			if mode.ui.KeyboardShortcutsBlocked(ctx) {
				t.Fatal("leaving the rename field did not restore shortcuts")
			}
			field.SetFocused(true)
			w.Close()
			if mode.ui.KeyboardShortcutsBlocked(ctx) {
				t.Fatal("closing the window did not restore shortcuts")
			}
		})
	}
}
