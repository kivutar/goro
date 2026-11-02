package game

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

func TestSuperNovicePrayerChatRouting(t *testing.T) {
	const first, last = "Cher ange", "Aide-moi"
	resources, err := res.NewManagerFS(fstest.MapFS{
		"data/msgstringtable.txt": &fstest.MapFile{Data: []byte(strings.Repeat("unused#\n", 790) + first + "#\nunused#\nunused#\n" + last + "#\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name      string
		job       int16
		dead      bool
		resources *res.Manager
		want      bool
		chatRoom  bool
	}{
		{"super novice", db.JobSuperNovice, false, resources, true, false},
		{"super baby", db.JobSuperNoviceB, false, resources, true, false},
		{"chat room", db.JobSuperNovice, false, resources, true, true},
		{"ordinary novice", db.JobNovice, false, resources, false, false},
		{"dead", db.JobSuperNovice, true, resources, false, false},
		{"no translations", db.JobSuperNovice, false, nil, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			netClient, server := testutil.NewConnection(t, 20080910)
			s := session.New()
			s.Selected = session.Character{Name: "Novice", Job: tt.job}
			s.Dead = tt.dead
			ctx := client.Context{Session: s, Network: netClient, Resources: tt.resources}
			// No EXP or learned skills are supplied: eligibility is server-owned.
			now := time.Now().Add(-time.Minute)
			for range 10 {
				s.SuperNovicePrayer.Record("hello", now)
			}
			for _, line := range []string{first, "any middle line", last} {
				if err := client.SendChat(ctx, line); err != nil {
					t.Fatal(err)
				}
				readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", line, 20080910))
			}
			before := s.SuperNovicePrayer
			for _, text := range []string{"%party", "$guild", "/w Friend whisper"} {
				if err := client.SendChat(ctx, text); err != nil {
					t.Fatal(err)
				}
			}
			readBotTestPackets(t, server, network.BuildPartyMessagePacket("Novice : party"))
			readBotTestPackets(t, server, network.BuildGuildMessagePacket("Novice : guild"))
			readBotTestPackets(t, server, network.BuildWhisperPacket("Friend", "whisper"))
			if s.SuperNovicePrayer != before {
				t.Fatal("non-public chat changed prayer history")
			}
			if tt.chatRoom {
				NewWorldMode().social().HandleChatRoomAction(ctx, gameui.ChatRoomWindowAction{Message: "thanks"})
			} else if err := client.SendChat(ctx, "thanks"); err != nil {
				t.Fatal(err)
			}
			readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", "thanks", 20080910))
			if tt.want {
				readBotTestPackets(t, server, []byte{0xED, 0x01})
			}
			// A subsequent packet also detects an unwanted or duplicate prayer.
			if err := client.SendChat(ctx, "done"); err != nil {
				t.Fatal(err)
			}
			readBotTestPackets(t, server, network.BuildGlobalChatPacketForClientDate("Novice", "done", 20080910))
			before = s.SuperNovicePrayer
			netClient.Close()
			if err := client.SendChat(ctx, first); err == nil {
				t.Fatal("disconnected chat succeeded")
			}
			if s.SuperNovicePrayer != before {
				t.Fatal("failed send changed prayer history")
			}
		})
	}
}

func TestSuperNovicePrayerResetsOnMapEntry(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	ctx.Session.SuperNovicePrayer.Record("previous map", time.Now())
	NewWorldMode().Enter(ctx)
	if ctx.Session.SuperNovicePrayer != (session.SuperNovicePrayer{}) {
		t.Fatal("prayer history leaked into the next map")
	}
}

func TestSuperNovicePrayerUsesServerFuryStatus(t *testing.T) {
	for _, job := range []int16{db.JobSuperNovice, db.JobSuperNoviceB} {
		ctx := testutil.MonkEffectContext()
		ctx.Session.Selected.Job = job
		ctx.World.Player.Job = job
		m := &WorldMode{}
		change := network.StatusEffectChange{ActorID: ctx.Session.AccountID, StatusID: db.StatusExplosionspirits, Active: true}
		m.applyStatusEffectChange(ctx, change)
		if !m.effects().HasActive(effects.Gumgang, ctx.Session.AccountID, time.Now().Add(time.Hour)) {
			t.Fatalf("job %d did not get persistent Fury lightning", job)
		}
		change.Active = false
		m.applyStatusEffectChange(ctx, change)
		if m.effects().HasActive(effects.Gumgang, ctx.Session.AccountID, time.Now()) {
			t.Fatalf("job %d kept Fury lightning after the server cleared it", job)
		}
	}
}

func TestSuperNoviceLevelUpEffects(t *testing.T) {
	for _, job := range []int16{db.JobSuperNovice, db.JobSuperNoviceB} {
		for _, explicit := range []bool{false, true} {
			for _, notifyFirst := range []bool{false, true} {
				ctx := testutil.MonkEffectContext()
				ctx.World.Player.Job, ctx.Session.Selected.Job = job, job
				ctx.Session.Progress.BaseLevel, ctx.Session.Progress.JobLevel = 60, 50
				m := &WorldMode{}
				parameter := func() {
					m.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 61})
					m.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 51})
				}
				notify := func() {
					base, job := uint32(network.SpecialEffectBaseLevelUp), uint32(network.SpecialEffectJobLevelUp)
					if explicit {
						base, job = network.SpecialEffectSuperNoviceBaseLevelUp, network.SpecialEffectSuperNoviceJobLevelUp
					}
					m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: ctx.Session.AccountID, EffectID: base})
					m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: ctx.Session.AccountID, EffectID: job})
				}
				if notifyFirst {
					notify()
					parameter()
				} else {
					parameter()
					notify()
				}
				if len(m.effects().Active) != 2 || !m.effects().HasActive(effects.Angel2, ctx.Session.AccountID, time.Now()) || !m.effects().HasActive(effects.JobLvUp50, ctx.Session.AccountID, time.Now()) {
					t.Fatalf("job %d explicit=%t notifyFirst=%t: duplicated or ordinary effects: %+v", job, explicit, notifyFirst, m.effects().Active)
				}
				if len(m.scheduledSounds) != 1 || m.scheduledSounds[0].paths[0] != "levelup.wav" {
					t.Fatalf("level-up sound must play once: %+v", m.scheduledSounds)
				}
				if !m.ui.levelUpNotifications.BaseVisible() || !m.ui.levelUpNotifications.JobVisible() {
					t.Fatal("missing local level-up notifications")
				}
			}
		}
	}
}

func TestRemoteSuperNoviceLevelUpIsSilent(t *testing.T) {
	for _, job := range []int16{db.JobSuperNovice, db.JobSuperNoviceB} {
		ctx := testutil.MonkEffectContext()
		actor := ctx.World.Actors[300]
		actor.Job = job
		ctx.World.Actors[300] = actor
		m := &WorldMode{}
		m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 300, EffectID: network.SpecialEffectSuperNoviceBaseLevelUp})
		if !m.effects().HasActive(effects.Angel2, 300, time.Now()) {
			t.Fatalf("job %d: missing remote guardian angel", job)
		}
		if len(m.scheduledSounds) != 0 {
			t.Fatalf("job %d: remote guardian angel must be silent: %+v", job, m.scheduledSounds)
		}
	}
}

func TestFirstAidEffectBelongsToCaster(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	m := &WorldMode{}
	notify := network.SkillNoDamageNotify{SourceID: ctx.Session.AccountID, TargetID: 300, SkillID: db.SkillNVFirstaid, Result: 0}
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active) != 0 {
		t.Fatal("failed First Aid produced a glow")
	}
	notify.Result = 1
	m.applySkillNoDamageNotify(ctx, notify)
	if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != effects.FirstAid || m.effects().Active[0].ActorID != ctx.Session.AccountID {
		t.Fatalf("First Aid must appear on its caster: %+v", m.effects().Active)
	}
}

func TestSuperNoviceRawLevelUpEffectsRemainExplicit(t *testing.T) {
	ctx := testutil.MonkEffectContext()
	ctx.Session.Selected.Job, ctx.World.Player.Job = db.JobSuperNovice, db.JobSuperNovice
	for _, id := range []int{effects.BaseLevelUp, effects.Angel2} {
		m := &WorldMode{}
		m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: ctx.Session.AccountID, EffectID: uint32(id)})
		if len(m.effects().Active) != 1 || m.effects().Active[0].EffectID != id {
			t.Fatalf("explicit effect %d was replaced: %+v", id, m.effects().Active)
		}
		if id == effects.Angel2 && len(m.scheduledSounds) != 0 {
			t.Fatal("the raw guardian angel effect should be silent")
		}
	}
}
