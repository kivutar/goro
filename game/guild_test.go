package game

import (
	"image"
	"image/color"
	"testing"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestBuildGuildFlagEmblemTextureAddsTransparentMarginAndColorBleed(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	source.SetNRGBA(0, 0, color.NRGBA{R: 240, G: 80, B: 40, A: 255})

	texture := buildGuildFlagEmblemTexture(source)
	if texture == nil {
		t.Fatal("flag emblem texture is nil")
	}
	if got := texture.Bounds(); got != image.Rect(0, 0, 4, 4) {
		t.Fatalf("flag emblem bounds = %v, want 4x4", got)
	}
	if got := texture.RGBA().RGBAAt(1, 1); got != (color.RGBA{R: 240, G: 80, B: 40, A: 255}) {
		t.Fatalf("centered emblem pixel = %+v", got)
	}
	if got := texture.RGBA().RGBAAt(0, 1); got != (color.RGBA{R: 240, G: 80, B: 40, A: 0}) {
		t.Fatalf("transparent edge bleed = %+v", got)
	}
}

func TestApplyLocalGuildDetailsInfersMasterFromSelectedCharacter(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Name: "Arcer"},
		Guild:    session.Guild{IsMaster: false},
	}

	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{
		GuildID:    1,
		GuildName:  "Mandala",
		MasterName: "Arcer",
	})

	if !s.Guild.IsMaster {
		t.Fatal("selected guild master should get master access from guild info")
	}
}

func TestApplyLocalGuildDetailsClearsMasterWhenSelectedCharacterIsNotMaster(t *testing.T) {
	s := &session.Session{
		Selected: session.Character{Name: "Kivutar"},
		Guild:    session.Guild{IsMaster: true},
	}

	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{
		GuildID:    1,
		GuildName:  "Mandala",
		MasterName: "Arcer",
	})

	if s.Guild.IsMaster {
		t.Fatal("non-master selected character should lose master access from guild info")
	}
}

func TestApplyLocalGuildBelongingStoresInviteRight(t *testing.T) {
	s := &session.Session{}
	applyLocalGuildBelonging(client.Context{Session: s}, network.GuildBelonging{
		GuildID: 1,
		Mode:    guildPermissionInvite,
	})

	if s.Guild.Right != guildPermissionInvite {
		t.Fatalf("guild right = 0x%X, want invite right", s.Guild.Right)
	}
	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{GuildID: 1, GuildName: "Mandala"})
	if s.Guild.Right != guildPermissionInvite {
		t.Fatalf("guild details cleared invite right: 0x%X", s.Guild.Right)
	}
}

func TestGuildCanInvitePlayerMatchesRobrowserRequirements(t *testing.T) {
	tests := []struct {
		name          string
		session       *session.Session
		targetGuildID uint32
		want          bool
	}{
		{name: "no session"},
		{name: "not in guild", session: &session.Session{Guild: session.Guild{Right: guildPermissionInvite}}},
		{name: "no invite right", session: &session.Session{GuildID: 1}},
		{name: "target already in guild", session: &session.Session{GuildID: 1, Guild: session.Guild{Right: guildPermissionInvite}}, targetGuildID: 2},
		{name: "invite permitted", session: &session.Session{GuildID: 1, Guild: session.Guild{Right: guildPermissionInvite}}, want: true},
		{name: "nested guild id", session: &session.Session{Guild: session.Guild{ID: 1, Right: guildPermissionInvite}}, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := guildCanInvitePlayer(test.session, test.targetGuildID); got != test.want {
				t.Fatalf("guildCanInvitePlayer() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestGuildCanManageRelationsRequiresMasterAndOtherGuild(t *testing.T) {
	s := &session.Session{Guild: session.Guild{ID: 10, IsMaster: true}}
	if !guildCanManageRelations(s, 20) {
		t.Fatal("guild master could not manage relation with another guild")
	}
	if guildCanManageRelations(s, 0) || guildCanManageRelations(s, 10) {
		t.Fatal("guild relation action allowed an invalid target guild")
	}
	s.Guild.IsMaster = false
	if guildCanManageRelations(s, 20) {
		t.Fatal("non-master could manage guild relations")
	}
}

func TestApplyLocalGuildRelationsAndPreserveThroughDetails(t *testing.T) {
	s := &session.Session{}
	ctx := client.Context{Session: s}
	applyLocalGuildRelations(ctx, []network.GuildRelation{
		{Relation: session.GuildRelationAlliance, GuildID: 10, Name: " Allies "},
		{Relation: session.GuildRelationOpposition, GuildID: 20, Name: "Enemies"},
	})
	applyLocalGuildDetails(ctx, network.GuildInfo{GuildID: 1, GuildName: "Mandala"})
	if len(s.Guild.Relations) != 2 || s.Guild.Relations[0].Name != "Allies" {
		t.Fatalf("guild relations after details = %+v", s.Guild.Relations)
	}
	applyLocalGuildRelationDeleted(ctx, network.GuildRelationDeleted{GuildID: 10, Relation: session.GuildRelationAlliance})
	if len(s.Guild.Relations) != 1 || s.Guild.Relations[0].GuildID != 20 {
		t.Fatalf("guild relations after delete = %+v", s.Guild.Relations)
	}
}

func TestGuildMemberOnlineUpdatesMaintainUserCount(t *testing.T) {
	s := &session.Session{}
	ctx := client.Context{Session: s}
	applyLocalGuildMembers(ctx, []network.GuildMember{
		{AccountID: 1, CharID: 11, CurrentState: 1},
		{AccountID: 2, CharID: 22, CurrentState: 0},
	})
	if s.Guild.UserNum != 1 {
		t.Fatalf("online member count = %d, want 1", s.Guild.UserNum)
	}
	if !applyLocalGuildMemberState(ctx, network.GuildMemberState{AccountID: 2, CharID: 22, State: 1}) || s.Guild.UserNum != 2 {
		t.Fatalf("online update members=%+v count=%d", s.Guild.Members, s.Guild.UserNum)
	}
	if !applyLocalGuildMemberState(ctx, network.GuildMemberState{AccountID: 1, CharID: 11, State: 0}) || s.Guild.UserNum != 1 {
		t.Fatalf("offline update members=%+v count=%d", s.Guild.Members, s.Guild.UserNum)
	}
	if !applyLocalGuildMemberState(ctx, network.GuildMemberState{AccountID: 2, CharID: 22, State: 1, HasAppearance: true, Sex: 1, HeadType: 7, HeadPalette: 8}) {
		t.Fatal("extended member appearance update was ignored")
	}
	if member := s.Guild.Members[1]; member.Sex != 1 || member.HeadType != 7 || member.HeadPalette != 8 {
		t.Fatalf("extended member appearance = %+v", member)
	}
}

func TestActorGuildEmblemRequestsFromUninitializedCache(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		Network: network.NewClient(20080910, false),
		Session: &session.Session{GuildID: 0x01020304, EmblemVersion: 7},
	}

	if emblem := mode.actorGuildEmblem(ctx, worldstate.Actor{}, true); emblem != nil {
		t.Fatalf("emblem = %v, want nil until image packet arrives", emblem)
	}
	if mode.guildEmblems == nil {
		t.Fatal("local guild emblem lookup should initialize request cache")
	}
}

func TestSiegeGuildEmblemEligibilityAndPosition(t *testing.T) {
	entry := sceneActorDrawEntry{
		actor:   worldstate.Actor{GuildID: 10, EmblemVersion: 2},
		screenX: 100,
		screenY: 200,
		scale:   1,
	}
	if !siegeActorShowsGuildEmblem(entry) {
		t.Fatal("visible guild actor did not qualify for a siege emblem")
	}
	entry.hidden = true
	if siegeActorShowsGuildEmblem(entry) {
		t.Fatal("hidden actor qualified for a siege emblem")
	}
	entry.hidden = false
	entry.actor.EffectState = db.EffectStateCloak
	if siegeActorShowsGuildEmblem(entry) {
		t.Fatal("cloaked actor qualified for a siege emblem")
	}
	entry.actor.EffectState = 0
	x, y := siegeGuildEmblemPosition(entry.screenX, actorSpriteTopY(entry.screenY, entry.scale), 24)
	if x != 88 || y != actorSpriteTopY(200, 1)-28 {
		t.Fatalf("siege emblem position = %.0f,%.0f", x, y)
	}
}
