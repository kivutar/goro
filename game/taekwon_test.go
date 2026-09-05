package game

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestTaekwonFamilySkillEffectsMatchRobrowser(t *testing.T) {
	spiritLinks := []uint16{
		db.SkillSLAlchemist, db.SkillSLMonk, db.SkillSLStar, db.SkillSLSage,
		db.SkillSLCrusader, db.SkillSLSupernovice, db.SkillSLKnight, db.SkillSLWizard,
		db.SkillSLPriest, db.SkillSLBarddancer, db.SkillSLRogue, db.SkillSLAssasin,
		db.SkillSLBlacksmith, db.SkillSLHunter, db.SkillSLSoullinker, db.SkillSLHigh,
	}
	for _, skillID := range spiritLinks {
		expectEffectIDs(t, "spirit link", skillEffectIDs(skillID), 424, 503)
	}
	expectEffectIDs(t, "SG_FUSION", skillEffectIDs(db.SkillSGFusion), 433, effectQuake)
	expectEffectIDs(t, "SL_SKA", skillEffectIDs(db.SkillSLSka), effectSteelBody, effectGumgang2, effectQuake)
}

func TestTaekwonStanceSkillsSuppressDefaultAction(t *testing.T) {
	for _, skillID := range []uint16{
		db.SkillTKReadystorm, db.SkillTKReadydown, db.SkillTKReadyturn,
		db.SkillTKReadycounter, db.SkillTKHighjump, db.SkillTKDodge,
	} {
		action := skillAction(skillID)
		if !action.defined || action.action != skillActorActionNone {
			t.Fatalf("skill %d action = %+v, want none", skillID, action)
		}
	}
}

func TestMildWindUsesLevelSpecificEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, Job: db.JobTaekwon, X: 10, Y: 20}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}
	mode := &WorldMode{}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID: db.SkillTKSevenwind, Amount: 3, SourceID: 2000000, TargetID: 2000000, Result: 1,
	})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectBeginAsura3 || mode.worldEffects[0].actorID != 2000000 {
		t.Fatalf("mild wind effects = %+v, want EF_BEGINASURA3", mode.worldEffects)
	}

	mode = &WorldMode{}
	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{
		SkillID: db.SkillTKSevenwind, Amount: 0, SourceID: 2000000, TargetID: 2000000, Result: 1,
	})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("level zero mild wind effects = %+v, want none", mode.worldEffects)
	}
}

func TestTaekwonStatusEffectsAndStancePosesMatchRobrowser(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobTaekwon, X: 10, Y: 20}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}
	mode := &WorldMode{}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusRun, ActorID: 2000000, Active: false})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectStopEffect {
		t.Fatalf("run stop effects = %+v, want EF_STOPEFFECT", mode.worldEffects)
	}
	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusTing, ActorID: 2000000, Active: true})
	if len(mode.worldEffects) != 2 || mode.worldEffects[1].effectID != effectQuakeBody {
		t.Fatalf("ting effects = %+v, want EF_QUAKEBODY", mode.worldEffects)
	}

	stances := []struct {
		status uint16
		action int
		frame  int
	}{
		{db.StatusStormkickOn, spriteActionPCSkill, 0},
		{db.StatusStormkickReady, spriteActionPCSkill, 0},
		{db.StatusDownkickOn, spriteActionPCSkill, 2},
		{db.StatusDownkickReady, spriteActionPCSkill, 2},
		{db.StatusTurnkickOn, spriteActionPCSkill, 3},
		{db.StatusTurnkickReady, spriteActionPCSkill, 3},
		{db.StatusCounterOn, spriteActionPCSkill, 4},
		{db.StatusCounterReady, spriteActionPCSkill, 4},
		{db.StatusDodgeOn, spriteActionPickup, 1},
		{db.StatusDodgeReady, spriteActionPickup, 1},
	}
	for _, tc := range stances {
		action, frame, ok := taekwonStanceAction(tc.status)
		if !ok || action != tc.action || frame != tc.frame {
			t.Fatalf("status %d stance = action %d frame %d ok=%t, want %d/%d/true", tc.status, action, frame, ok, tc.action, tc.frame)
		}
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusCounterReady, ActorID: 2000000, Active: true})
	anim, ok := mode.actorAnims[150000]
	if !ok || anim.actionFamily != spriteActionPCSkill || !anim.hasFixedMotion || anim.fixedMotion != 4 || anim.play || !anim.holdFinal {
		t.Fatalf("counter stance animation = %+v ok=%t", anim, ok)
	}
}

func TestStarWarmAndSoulLinkStatusTintsMatchRobrowser(t *testing.T) {
	swoo := actorStateTint(worldstate.Actor{Opt3State: db.Opt3Overthrust})
	if swoo.R != 255 || swoo.G != 191 || swoo.B != 191 || swoo.A != 255 {
		t.Fatalf("Eswoo tint = %+v", swoo)
	}
	warm := actorStateTint(worldstate.Actor{Opt3State: db.Opt3Warm})
	if warm.R != 255 || warm.G != 102 || warm.B != 102 || warm.A != 255 {
		t.Fatalf("warm tint = %+v", warm)
	}
	soulLink := actorStateTint(worldstate.Actor{Opt3State: db.Opt3Soullink})
	if soulLink.R != 89 || soulLink.G != 89 || soulLink.B != 229 || soulLink.A != 229 {
		t.Fatalf("soul link tint = %+v", soulLink)
	}
}

func TestSoulLinkAndEskeToggleLocalNightLighting(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobLinker}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}
	mode := &WorldMode{
		gndMeshCache: &gndRetainedMeshCache{},
		rsmMeshCache: map[int][]retainedWorldMesh{1: nil},
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSoullink, ActorID: 2000000, Active: true})
	if !mode.taekwonNight || mode.gndMeshCache != nil || len(mode.rsmMeshCache) != 0 {
		t.Fatalf("night state=%t gnd=%p rsm=%v", mode.taekwonNight, mode.gndMeshCache, mode.rsmMeshCache)
	}
	lighting := mode.sceneLighting(nil)
	if lighting.diffuse.x != 0.5 || lighting.diffuse.y != 0.5 || lighting.diffuse.z != 1 || lighting.env.x != 0.5 || lighting.env.y != 0.5 || lighting.env.z != 1 {
		t.Fatalf("night lighting = %+v", lighting)
	}

	mode.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSke, ActorID: 2000000, Active: false})
	if mode.taekwonNight {
		t.Fatal("inactive Eske did not restore day lighting")
	}
	lighting = mode.sceneLighting(nil)
	if lighting.diffuse.x != 1 || lighting.diffuse.y != 1 || lighting.diffuse.z != 1 {
		t.Fatalf("restored lighting = %+v", lighting)
	}
}

func TestTaekwonMissionAndRankingMessages(t *testing.T) {
	mode := &WorldMode{}
	mode.applyTaekwonMission(client.Context{}, network.TaekwonMission{MonsterName: "Spore", MonsterID: 1014, Progress: 37, Result: 1})
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Taekwon mission: Spore (37%)" || messages[0].Color != taekwonAnnouncementColor {
		t.Fatalf("mission messages = %+v", messages)
	}

	entries := make([]network.FameRankingEntry, 10)
	entries[0] = network.FameRankingEntry{Name: "Kicker", Points: 123}
	presentation := fameRankingPresentationFor(network.FameRankingTaekwon)
	if presentation.professionMessageID != 2388 || presentation.pointMessageID != 926 {
		t.Fatalf("taekwon message IDs=(%d, %d) want=(2388, 926)", presentation.professionMessageID, presentation.pointMessageID)
	}
	mode.applyFameRanking(client.Context{}, network.FameRanking{Kind: network.FameRankingTaekwon, Entries: entries})
	messages = mode.ui.console.Messages()
	if len(messages) != 12 {
		t.Fatalf("message count = %d, want mission + header + ten ranks", len(messages))
	}
	if messages[1].Text != "=========== Taekwon Rank ===========" || messages[2].Text != "[1] Kicker : 123 Points" || messages[11].Text != "[10] None : 0 Points" {
		t.Fatalf("ranking messages = %+v", messages[1:])
	}

	mode.applyFamePointUpdate(client.Context{}, network.FamePointUpdate{Kind: network.FameRankingTaekwon, GainedPoints: 1, TotalPoints: 42})
	messages = mode.ui.console.Messages()
	if len(messages) != 13 || messages[12].Text != "[POINT] You have been rewarded with 1 Tae-Kwon Mission rank points. Your point total is 42." {
		t.Fatalf("taekwon point message = %+v", messages)
	}
}

func TestStarPlaceRequestOpensConfirmationAndAccepts(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := listener.Accept()
		accepted <- conn
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	address := listener.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), address.IP.String(), address.Port); err != nil {
		t.Fatal(err)
	}
	serverConn := <-accepted
	if serverConn == nil {
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	ctx := client.Context{Network: netClient, ScreenW: 800, ScreenH: 600}
	mode := &WorldMode{}
	packet := network.Packet{ID: network.PacketZCStarPlace, Data: []byte{0x53, 0x02, 0x01}}
	if next, stop := mode.handleNetworkPacket(ctx, packet, time.Now()); next != nil || stop {
		t.Fatalf("star place request changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.starPlaceConfirm.IsOpen() {
		t.Fatal("star place request did not open confirmation")
	}
	if !mode.ui.interactionModalOpen() {
		t.Fatal("star place confirmation did not block world interactions")
	}

	mode.ui.starPlaceConfirm.Confirm(ctx)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	want := network.BuildAgreeStarPlacePacket(1)
	got := make([]byte, len(want))
	if _, err := io.ReadFull(serverConn, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("star place reply = %x, want %x", got, want)
	}
}

func TestStarPlaceCancelDoesNotSendAgreement(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Network: network.NewClient(20080910, false), ScreenW: 800, ScreenH: 600}

	mode.applyStarPlaceRequest(ctx, network.StarPlace{Place: 2})
	mode.ui.starPlaceConfirm.Cancel(ctx)

	if mode.ui.starPlaceConfirm.IsOpen() {
		t.Fatal("star place confirmation stayed open after cancel")
	}
	if messages := mode.ui.console.Messages(); len(messages) != 0 {
		t.Fatalf("cancel attempted a network reply: %+v", messages)
	}
}
