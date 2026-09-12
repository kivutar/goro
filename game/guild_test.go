package game

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"io"
	"net"
	"testing"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	"github.com/kivutar/goro/ui/rotheme"
	worldstate "github.com/kivutar/goro/world"
)

func TestGuildNoticeEditorEnterDoesNotActivateConsole(t *testing.T) {
	mode := NewWorldMode()
	ctx := client.Context{
		Session: &session.Session{GuildID: 99, Guild: session.Guild{
			ID: 99, IsMaster: true, MenuAccess: 0x80, Notice: "Hello",
		}},
		World: worldstate.New(), Input: input.NewState(), Network: network.NewClient(20080910, false), ScreenW: 800, ScreenH: 600,
	}
	t.Cleanup(func() { ctx.Network.Close() })
	mode.ui.guildWindow.OpenWindow(ctx)
	root := mode.ui.guildWindow.Widget()
	wc := widget.NewContext()
	root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
	root.Draw(wc, &uitest.MockCanvas{})
	// Open the final tab through the positioned widget tree, as a click would.
	tabs := root.Children()[0].Children()[1].Children()[0].Children()[0]
	notice := tabs.Children()[5].(interface{ ScreenBounds() geometry.Rect })
	p := notice.ScreenBounds().Center()
	root.Event(wc, event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft, p, p, event.ModNone))
	root.Event(wc, event.NewMouseEvent(event.MouseRelease, event.ButtonLeft, 0, p, p, event.ModNone))
	root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
	root.Draw(wc, &uitest.MockCanvas{})
	var field *rotheme.TextAreaWidget
	var findEditor func(widget.Widget)
	findEditor = func(w widget.Widget) {
		if editor, ok := w.(*rotheme.TextAreaWidget); ok {
			field = editor
		}
		for _, child := range w.Children() {
			findEditor(child)
		}
	}
	findEditor(mode.ui.guildWindow.Widget())
	if field == nil {
		t.Fatalf("Notice tab has no multiline editor: tab=%T bounds=%v root=%v", notice, notice.ScreenBounds(), root.(interface{ ScreenBounds() geometry.Rect }).ScreenBounds())
	}
	field.SetFocused(true)
	field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyEnd, 0, event.ModCtrl))
	field.Event(wc, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	ctx.Input.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.console.Active() || field.Text() != "Hello\n" || !mode.ui.keyboardInputBlocked(ctx) {
		t.Fatal("Enter in the guild notice did not stay in the multiline editor")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetKey(input.KeyEnter, false)
	ctx.Input.SetKey(input.KeyEscape, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.ui.guildWindow.IsOpen() || mode.ui.escapeMenu.IsOpen() || mode.ui.console.Active() || field.IsFocused() {
		t.Fatal("Escape should close only the guild window and release editor focus")
	}
}

func TestAcceptingGuildInvitationWaitsForBelonging(t *testing.T) {
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

	world := worldstate.New()
	sessionState := &session.Session{}
	ctx := client.Context{
		Network: netClient,
		Session: sessionState,
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	mode := &WorldMode{}
	request := network.GuildInviteRequest{GuildID: 9, GuildName: "Knights"}

	mode.openGuildInviteRequest(ctx, request)
	mode.ui.guildRequest.Confirm(ctx)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(network.BuildGuildInviteReplyPacket(request.GuildID, true)))
	if _, err := io.ReadFull(serverConn, got); err != nil {
		t.Fatal(err)
	}
	want := network.BuildGuildInviteReplyPacket(request.GuildID, true)
	if !bytes.Equal(got, want) {
		t.Fatalf("guild invite reply = %x, want %x", got, want)
	}
	if sessionState.GuildID != 0 || sessionState.GuildName != "" {
		t.Fatalf("session guild changed before belonging: id=%d name=%q", sessionState.GuildID, sessionState.GuildName)
	}
	if world.Player.GuildID != 0 || world.Player.GuildName != "" {
		t.Fatalf("player guild changed before belonging: id=%d name=%q", world.Player.GuildID, world.Player.GuildName)
	}
}

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

func TestApplyLocalGuildMenuAccessRequiresMembership(t *testing.T) {
	member := &session.Session{GuildID: 9}
	applyLocalGuildMenuAccess(client.Context{Session: member}, network.GuildMenuAccess{Mask: 0xD7})
	if member.Guild.MenuAccess != 0xD7 {
		t.Fatalf("member menu access = 0x%X, want 0xD7", member.Guild.MenuAccess)
	}

	guildless := &session.Session{}
	applyLocalGuildMenuAccess(client.Context{Session: guildless}, network.GuildMenuAccess{Mask: 0xD7})
	if guildless.Guild.MenuAccess != 0 {
		t.Fatalf("guildless menu access = 0x%X, want 0", guildless.Guild.MenuAccess)
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
	s.Guild.MenuAccess = 0x57
	applyLocalGuildDetails(client.Context{Session: s}, network.GuildInfo{GuildID: 1, GuildName: "Mandala"})
	if s.Guild.Right != guildPermissionInvite {
		t.Fatalf("guild details cleared invite right: 0x%X", s.Guild.Right)
	}
	if s.Guild.MenuAccess != 0x57 {
		t.Fatalf("guild details cleared menu access: 0x%X", s.Guild.MenuAccess)
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

func TestGuildMemberLifecyclePermissionsMatchClassicClient(t *testing.T) {
	s := &session.Session{
		AccountID: 10,
		CharID:    11,
		GuildID:   1,
		Guild:     session.Guild{Right: guildPermissionExpel},
	}
	self := gameui.GuildMemberAction{Kind: gameui.GuildMemberActionLeave, Member: session.GuildMember{AccountID: 10, CharID: 11}}
	other := gameui.GuildMemberAction{Kind: gameui.GuildMemberActionExpel, Member: session.GuildMember{AccountID: 20, CharID: 21}}
	if !guildMemberActionAllowed(s, self) {
		t.Fatal("ordinary guild member should be allowed to leave")
	}
	if !guildMemberActionAllowed(s, other) {
		t.Fatal("member with expulsion right should be allowed to expel another member")
	}
	s.Guild.IsMaster = true
	if guildMemberActionAllowed(s, self) {
		t.Fatal("guild master should disband rather than leave")
	}
	s.Guild.Right = 0
	if !guildMemberActionAllowed(s, other) {
		t.Fatal("guild master should be allowed to expel while rights are not loaded")
	}
	other.Member.AccountID, other.Member.CharID = 10, 11
	if guildMemberActionAllowed(s, other) {
		t.Fatal("member should never be allowed to expel themself")
	}
}

func TestGuildDepartureRemovesRemoteMemberWithoutClearingLocalGuild(t *testing.T) {
	s := &session.Session{
		AccountID: 1,
		CharID:    2,
		GuildID:   9,
		Selected:  session.Character{Name: "Local"},
		Guild: session.Guild{ID: 9, Members: []session.GuildMember{
			{AccountID: 1, CharID: 2, CharName: "Local", CurrentState: 1},
			{AccountID: 3, CharID: 4, CharName: "Alice", CurrentState: 1},
		}},
	}
	w := &worldstate.World{Actors: map[uint32]worldstate.Actor{
		3: {ID: 3, Name: "Alice", GuildID: 9, GuildName: "Mandala", EmblemVersion: 4},
	}}
	mode := &WorldMode{}
	mode.handleGuildMemberDeparture(client.Context{Session: s, World: w}, network.GuildMemberDeparture{CharName: "Alice", Reason: "Moving on"})

	if s.GuildID != 9 || s.Guild.ID != 9 {
		t.Fatal("remote departure cleared the local guild")
	}
	if len(s.Guild.Members) != 1 || s.Guild.Members[0].CharName != "Local" || s.Guild.UserNum != 1 {
		t.Fatalf("guild members after departure = %+v, online=%d", s.Guild.Members, s.Guild.UserNum)
	}
	if actor := w.Actors[3]; actor.GuildID != 0 || actor.GuildName != "" || actor.EmblemVersion != 0 {
		t.Fatalf("departed visible actor retained guild state: %+v", actor)
	}
}

func TestLocalGuildExpulsionClearsSessionAndPlayerState(t *testing.T) {
	s := &session.Session{
		AccountID:        1,
		CharID:           2,
		GuildID:          9,
		GuildName:        "Mandala",
		EmblemVersion:    4,
		Selected:         session.Character{Name: "Local"},
		PendingGuildName: "Pending",
		Guild: session.Guild{ID: 9, Name: "Mandala", Members: []session.GuildMember{
			{AccountID: 1, CharID: 2, CharName: "Local", CurrentState: 1},
		}},
	}
	w := &worldstate.World{Player: worldstate.Actor{GuildID: 9, GuildName: "Mandala", EmblemVersion: 4}}
	mode := &WorldMode{}
	mode.handleGuildMemberExpulsion(client.Context{Session: s, World: w}, network.GuildMemberExpulsion{CharName: "Local", Reason: "Inactive"})

	if s.GuildID != 0 || s.GuildName != "" || s.EmblemVersion != 0 || s.Guild.ID != 0 || s.PendingGuildName != "" {
		t.Fatalf("stale session guild state after expulsion: %+v", s.Guild)
	}
	if w.Player.GuildID != 0 || w.Player.GuildName != "" || w.Player.EmblemVersion != 0 {
		t.Fatalf("stale player guild state after expulsion: %+v", w.Player)
	}
	applyLocalGuildMembers(client.Context{Session: s}, []network.GuildMember{{AccountID: 3, CharID: 4, CharName: "Stale"}})
	if len(s.Guild.Members) != 0 {
		t.Fatalf("post-expulsion member list restored stale guild state: %+v", s.Guild.Members)
	}
}

func TestGuildDisbandClearsVisibleMembersOfDisbandedGuildOnly(t *testing.T) {
	s := &session.Session{GuildID: 9, Guild: session.Guild{ID: 9}}
	w := &worldstate.World{
		Player: worldstate.Actor{GuildID: 9},
		Actors: map[uint32]worldstate.Actor{
			1: {GuildID: 9, GuildName: "Mandala", EmblemVersion: 4},
			2: {GuildID: 10, GuildName: "Other", EmblemVersion: 5},
		},
	}
	mode := &WorldMode{}
	mode.handleGuildDisbandResult(client.Context{Session: s, World: w}, network.GuildDisbandResult{Result: 0})

	if got := w.Actors[1]; got.GuildID != 0 || got.GuildName != "" || got.EmblemVersion != 0 {
		t.Fatalf("disbanded guild actor state = %+v", got)
	}
	if got := w.Actors[2]; got.GuildID != 10 || got.GuildName != "Other" || got.EmblemVersion != 5 {
		t.Fatalf("unrelated guild actor state changed: %+v", got)
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
	s := &session.Session{GuildID: 1}
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
	entry.actor.EffectState = db.EffectStateHide
	if siegeActorShowsGuildEmblem(entry) {
		t.Fatal("hidden actor qualified for a siege emblem")
	}
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
