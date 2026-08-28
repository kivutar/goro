package game

import (
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestApplyActorNameAckUpdatesWorldActor(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Job: 1002})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Guide#prontera", PartyName: "Adventurers", GuildName: "Knights"})

	if got := world.Actors[300].Name; got != "Guide" {
		t.Fatalf("actor name = %q, want Guide", got)
	}
	if got := world.Actors[300].PartyName; got != "Adventurers" {
		t.Fatalf("actor party = %q, want Adventurers", got)
	}
	if got := world.Actors[300].GuildName; got != "Knights" {
		t.Fatalf("actor guild = %q, want Knights", got)
	}
}

func TestApplyActorNameAckUpdatesLocalPlayer(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar", PartyName: "Adventurers", GuildName: "Goro"})

	if got := world.Player.Name; got != "Kivutar" {
		t.Fatalf("player name = %q, want Kivutar", got)
	}
	if got := world.Player.PartyName; got != "Adventurers" {
		t.Fatalf("player party = %q, want Adventurers", got)
	}
	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
}

func TestApplyActorNameAckPreservesLocalGuildOnEmptyNameAck(t *testing.T) {
	world := worldstate.New()
	world.Player.GuildName = "Goro"
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200, GuildName: "Goro"},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 200, Name: "Kivutar"})

	if got := world.Player.GuildName; got != "Goro" {
		t.Fatalf("player guild = %q, want Goro", got)
	}
	if got := ctx.Session.GuildName; got != "Goro" {
		t.Fatalf("session guild = %q, want Goro", got)
	}
}

func TestApplyActorNameAckClearsRemotePartyName(t *testing.T) {
	world := worldstate.New()
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Alice", PartyName: "Old Party"})
	ctx := client.Context{
		Session: &session.Session{AccountID: 100, CharID: 200},
		World:   world,
	}

	applyActorNameAck(ctx, network.ActorNameAck{ID: 300, Name: "Alice"})

	if got := world.Actors[300].PartyName; got != "" {
		t.Fatalf("actor party = %q, want cleared", got)
	}
}

func TestHandleMapChangeSameServerUpdatesMapAndResetsActors(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Remote", X: 11, Y: 20})
	sessionState := &session.Session{AccountID: 100, CharID: 200, PlayerDir: 4}
	uiManager := &worldModeTestUIManager{}
	ctx := client.Context{
		Session:   sessionState,
		World:     world,
		Input:     input.NewState(),
		UIManager: uiManager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.npcDialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Warping..."})
	if !mode.ui.npcDialog.Update(ctx) {
		t.Fatal("npc dialog did not publish before map change")
	}
	if len(uiManager.overlays) == 0 {
		t.Fatal("npc dialog overlay was not published before map change")
	}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "geffen", X: 120, Y: 80})

	if next == nil || next.Name() != "world" {
		t.Fatalf("next mode = %#v, want world", next)
	}
	if len(uiManager.overlays) != 0 {
		t.Fatalf("npc dialog overlays after map change = %d, want 0", len(uiManager.overlays))
	}
	if world.MapName != "geffen" || sessionState.Zone.MapName != "geffen" {
		t.Fatalf("map = world %q session %q, want geffen", world.MapName, sessionState.Zone.MapName)
	}
	if world.Player.X != 120 || world.Player.Y != 80 || sessionState.PlayerX != 120 || sessionState.PlayerY != 80 {
		t.Fatalf("position = world %d,%d session %d,%d", world.Player.X, world.Player.Y, sessionState.PlayerX, sessionState.PlayerY)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("actors were not cleared: %+v", world.Actors)
	}
}

func TestNextWorldModeReusesMinimapOverlay(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	ctx := client.Context{
		World:     world,
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.minimap.Update(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("minimap overlays before map change = %d, want 1", len(manager.overlays))
	}

	next := mode.nextWorldMode()
	world.MapName = "geffen"
	world.SetPlayerPosition(120, 80, 4)
	next.ui.minimap.Update(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("minimap overlays after map change = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesCameraPreferences(t *testing.T) {
	mode := &WorldMode{camera: followCamera{
		yawOffset:  73,
		pitch:      245,
		zoom:       148,
		zoomTarget: 152,
	}}

	next := mode.nextWorldMode()

	if next.camera.yawOffset != 73 || next.camera.pitch != 245 || next.camera.zoom != 148 || next.camera.zoomTarget != 152 {
		t.Fatalf("camera preferences = yaw %.1f pitch %.1f zoom %.1f target %.1f", next.camera.yawOffset, next.camera.pitch, next.camera.zoom, next.camera.zoomTarget)
	}
}

func TestNextWorldModeCarriesOpenInventoryWindow(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{Items: []session.InventoryItem{
			{Index: 2, ItemID: 501, Type: 0, Amount: 3, Identified: true},
		}},
	}
	ctx := client.Context{
		Session:   sessionState,
		World:     worldstate.New(),
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.inventoryBag.Toggle(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) == 0 {
		t.Fatal("inventory overlay was not published before map change")
	}

	next := mode.nextWorldMode()
	if len(manager.overlays) != 1 {
		t.Fatalf("inventory overlays after mode replacement = %d, want carried overlay", len(manager.overlays))
	}
	next.ui.inventoryBag.Update(ctx, &next.ui.shortcutBar, &next.ui.storageWindow, &next.ui.cartWindow, nil, &next.ui.equipmentWindow, &next.ui.itemInfoWindow)
	if len(manager.overlays) != 1 {
		t.Fatalf("inventory overlays after next mode update = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesAndRebindsWhisperWindow(t *testing.T) {
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.whisperWindow.Open(ctx, "Alice")
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.whisperWindow.IsOpen() {
		t.Fatal("next world mode did not carry open whisper window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("whisper overlay was not rebound")
	}

	next.ui.whisperWindow.AddError(ctx, "send failed")
	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays after refresh = %d, want 1", len(manager.overlays))
	}
}

func TestNextWorldModeCarriesAndRebindsFriendSettingsWindow(t *testing.T) {
	ctx := client.Context{
		Session:   &session.Session{},
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.friendSettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("friend settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.friendSettings.IsOpen() {
		t.Fatal("next world mode did not carry open friend settings window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("friend settings overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("friend settings overlay was not rebound")
	}
}

func TestNextWorldModeCarriesAndRebindsPartySettingsWindow(t *testing.T) {
	ctx := client.Context{
		Session: &session.Session{Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 10, Role: 0}},
		}},
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.partySettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("party settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.partySettings.IsOpen() {
		t.Fatal("next world mode did not carry open party settings window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("party settings overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] == previousOverlay {
		t.Fatal("party settings overlay was not rebound")
	}
}

func TestNextWorldModeCarriesAndRebindsPartyHelperWindows(t *testing.T) {
	ctx := client.Context{
		Input:     input.NewState(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := &WorldMode{}
	mode.ui.partyCreate.Open(ctx)
	mode.ui.partyInvite.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 2 {
		t.Fatalf("party helper overlays before map change = %d, want 2", len(manager.overlays))
	}
	previousCreate := manager.overlays[0]
	previousInvite := manager.overlays[1]

	next := mode.nextWorldMode()
	if !next.ui.partyCreate.IsOpen() || !next.ui.partyInvite.IsOpen() {
		t.Fatalf("next world mode did not carry helper windows create=%t invite=%t", next.ui.partyCreate.IsOpen(), next.ui.partyInvite.IsOpen())
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 2 {
		t.Fatalf("party helper overlays after rebind = %d, want 2", len(manager.overlays))
	}
	if manager.overlays[0] == previousCreate || manager.overlays[1] == previousInvite {
		t.Fatal("party helper overlay was not rebound")
	}
}

func TestHandleMapChangeSameLoadedMapReusesModeAndSnapsCamera(t *testing.T) {
	world := worldstate.New()
	world.MapName = "izlude"
	world.GND = &res.GND{}
	world.SetPlayerPosition(10, 20, 4)
	world.UpsertActor(worldstate.Actor{ID: 300, Name: "Remote", X: 11, Y: 20})
	sessionState := &session.Session{AccountID: 100, CharID: 200, PlayerDir: 4}
	ctx := client.Context{
		Session: sessionState,
		World:   world,
	}
	mode := &WorldMode{}

	next := mode.handleMapChange(ctx, network.MapChange{MapName: "izlude", X: 114, Y: 145})

	if next != nil {
		t.Fatalf("next mode = %#v, want nil same-mode reuse", next)
	}
	if world.Player.X != 114 || world.Player.Y != 145 || sessionState.PlayerX != 114 || sessionState.PlayerY != 145 {
		t.Fatalf("position = world %d,%d session %d,%d", world.Player.X, world.Player.Y, sessionState.PlayerX, sessionState.PlayerY)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("actors were not cleared: %+v", world.Actors)
	}
	if !mode.camera.initialized || mode.camera.x != 114.5 || mode.camera.y != 145.5 {
		t.Fatalf("camera = initialized %t %.2f,%.2f, want 114.5,145.5", mode.camera.initialized, mode.camera.x, mode.camera.y)
	}
}

func TestMapCellUpdateChangesGATWalkability(t *testing.T) {
	world := worldstate.New()
	world.MapName = "geffen"
	world.GAT = testPathGAT(3, 3, nil)
	mode := &WorldMode{hoveredWalk: hoveredWalkCellCache{valid: true}}

	mode.applyMapCellUpdate(client.Context{World: world}, network.MapCellUpdate{MapName: "geffen", X: 1, Y: 2, RawType: 5})

	if world.GAT.Walkable(1, 2) {
		t.Fatalf("updated cell should not be walkable: %+v", world.GAT.Cells[2*world.GAT.Width+1])
	}
	if mode.hoveredWalk.valid {
		t.Fatal("hovered walk cache should be invalidated")
	}
}

func TestActorDisplayNameUsesSelectedCharacterForPlayer(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player"}, true); got != "Kivutar" {
		t.Fatalf("display name = %q, want Kivutar", got)
	}
}

func TestActorDisplayNameIncludesPartyName(t *testing.T) {
	ctx := client.Context{Session: &session.Session{
		CharID:   200,
		Selected: session.Character{ID: 200, Name: "Kivutar"},
		Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 300, Name: "Alice"}},
		},
	}}

	if got := actorDisplayName(ctx, worldstate.Actor{Name: "Player", PartyName: "Goro"}, true); got != "Kivutar (Goro)" {
		t.Fatalf("local display name = %q, want Kivutar (Goro)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 300, Name: "Alice", PartyName: "Other Party"}, false); got != "Alice (Other Party)" {
		t.Fatalf("remote display name = %q, want Alice (Other Party)", got)
	}
	if got := actorDisplayName(ctx, worldstate.Actor{ID: 400, Name: "Bob"}, false); got != "Bob" {
		t.Fatalf("remote without packet party name = %q, want Bob", got)
	}
}

func TestActorDisplayLabelsIncludeGuildNameOnSecondLine(t *testing.T) {
	ctx := client.Context{Session: &session.Session{CharID: 200, Selected: session.Character{ID: 200, Name: "Kivutar"}}}

	labels := actorDisplayLabels(ctx, worldstate.Actor{Name: "Player", GuildName: "Knights"}, true)
	if len(labels) != 2 || labels[0] != "Kivutar" || labels[1] != "Knights" {
		t.Fatalf("local labels = %#v, want Kivutar / Knights", labels)
	}

	labels = actorDisplayLabels(ctx, worldstate.Actor{ID: 300, Name: "Alice", GuildName: "Knights", HasObjectType: true, ObjectType: actorObjectTypePC}, false)
	if len(labels) != 2 || labels[0] != "Alice" || labels[1] != "Knights" {
		t.Fatalf("actor labels = %#v, want Alice / Knights", labels)
	}

	labels = actorDisplayLabels(ctx, worldstate.Actor{Name: "Poring", GuildName: "Knights", HasObjectType: true, ObjectType: actorObjectTypeMob}, false)
	if len(labels) != 1 || labels[0] != "Poring" {
		t.Fatalf("mob labels = %#v, want Poring only", labels)
	}
}

func TestActorNameLabelColorUsesYellowForAdmin(t *testing.T) {
	want := color.RGBA{R: 255, G: 255, B: 0, A: 255}
	if got := actorNameLabelColor(worldstate.Actor{IsAdmin: true}, true); got != want {
		t.Fatalf("local admin label color = %+v, want %+v", got, want)
	}
	if got := actorNameLabelColor(worldstate.Actor{IsAdmin: true}, false); got != want {
		t.Fatalf("remote admin label color = %+v, want %+v", got, want)
	}
}

func TestGuildCreationResultAppliesPendingLocalGuildName(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{PendingGuildName: "Knights"},
		World:   world,
	}
	var mode WorldMode

	mode.handleGuildCreationResult(ctx, network.GuildCreationResult{Result: 0})

	if got := ctx.Session.GuildName; got != "Knights" {
		t.Fatalf("session guild = %q, want Knights", got)
	}
	if got := world.Player.GuildName; got != "Knights" {
		t.Fatalf("player guild = %q, want Knights", got)
	}
	if got := ctx.Session.PendingGuildName; got != "" {
		t.Fatalf("pending guild = %q, want empty", got)
	}
}

func TestHandleGuildNoticeUpdatesSessionAndAddsGuildConsoleMessages(t *testing.T) {
	sessionState := &session.Session{}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState}

	mode.handleGuildNotice(ctx, network.GuildNotice{
		Subject: " Maintenance ",
		Notice:  " Gather in Prontera. ",
	})

	if got := sessionState.Guild.NoticeSubject; got != "Maintenance" {
		t.Fatalf("notice subject = %q, want Maintenance", got)
	}
	if got := sessionState.Guild.Notice; got != "Gather in Prontera." {
		t.Fatalf("notice = %q, want Gather in Prontera.", got)
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 2 {
		t.Fatalf("console messages = %+v, want 2 notice lines", messages)
	}
	if messages[0].Text != "[ Maintenance ]" || messages[1].Text != "[ Gather in Prontera. ]" {
		t.Fatalf("console messages = %+v", messages)
	}
	wantColor := color.RGBA{R: 255, G: 255, B: 99, A: 255}
	if messages[0].Color != wantColor || messages[1].Color != wantColor {
		t.Fatalf("console message colors = %+v, want %+v", messages, wantColor)
	}
}

func TestHandleGuildNoticeSkipsEmptyConsoleLines(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{}}

	mode.handleGuildNotice(ctx, network.GuildNotice{
		Subject: " ",
		Notice:  " Guild event tonight.\nMeet in Prontera. ",
	})

	messages := mode.ui.console.Messages()
	if len(messages) != 2 || messages[0].Text != "[ Guild event tonight. ]" || messages[1].Text != "[ Meet in Prontera. ]" {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestActorDisplayNameUsesServerNameBeforeFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Name: "Kafra Employee#izlude", Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Kafra Employee" {
		t.Fatalf("display name = %q, want Kafra Employee", got)
	}
}

func TestActorDisplayNameUsesImportedMonsterFallback(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 1002}

	if got := actorDisplayName(ctx, actor, false); got != "Poring" {
		t.Fatalf("display name = %q, want Poring from imported DB", got)
	}
}

func TestActorDisplayNameDoesNotLabelUnnamedPlayerJob(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	actor := worldstate.Actor{Job: 0}

	if got := actorDisplayName(ctx, actor, false); got != "" {
		t.Fatalf("display name = %q, want empty", got)
	}
}

func TestHoverActorServerNameLookupIncludesCompanions(t *testing.T) {
	for _, actor := range []worldstate.Actor{
		{HasObjectType: true, ObjectType: actorObjectTypeHomunculus},
		{HasObjectType: true, ObjectType: actorObjectTypeMercenary},
	} {
		if !shouldUseServerNameForHoverActor(actor) {
			t.Fatalf("companion actor should request server name: %+v", actor)
		}
	}
}

func TestHoveredActorDisplayNameUsesServerNameForNPC(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	mode := &WorldMode{}
	actor := worldstate.Actor{
		Job:           84,
		ObjectType:    actorObjectTypeNPC,
		HasObjectType: true,
	}

	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "4 M 02" {
		t.Fatalf("hovered NPC name = %q, want imported resource label", got)
	}
	actor.Name = "Kafra Employee#izlude"
	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Kafra Employee" {
		t.Fatalf("hovered NPC server name = %q, want Kafra Employee", got)
	}
}

func TestHoveredActorDisplayNameUsesServerNameForMonster(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	mode := &WorldMode{}
	actor := worldstate.Actor{
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	}

	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Poring" {
		t.Fatalf("hovered monster name = %q, want imported monster label", got)
	}
	actor.Name = "Poring"
	if got := mode.hoveredActorDisplayName(ctx, actor, time.Now()); got != "Poring" {
		t.Fatalf("hovered monster server name = %q, want Poring", got)
	}
}

func TestFormatConsoleMessageUsesMsgStringTable(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte("ignored#\nYou got %d items.#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	got := formatConsoleMessage(manager, network.ChatMessage{MessageID: 1, Value: 3})
	if got != "You got 3 items." {
		t.Fatalf("message = %q", got)
	}
}

func TestExitRefusalMessageUsesOriginalMessageFallback(t *testing.T) {
	mode := &WorldMode{}

	mode.addLeaveWorldRefusalMessage(client.Context{})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 {
		t.Fatalf("console messages = %d, want 1", len(messages))
	}
	if got, want := messages[0].Text, "You cannot exit the game right now."; got != want {
		t.Fatalf("console message = %q, want %q", got, want)
	}
}

func TestExitRefusalMessageUsesMsgString502(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	msgTable := strings.Repeat("ignored#\n", 502) + "Please wait 10 seconds before exiting.#\n"
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte(msgTable), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	mode := &WorldMode{}

	mode.addLeaveWorldRefusalMessage(client.Context{Resources: manager})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Please wait 10 seconds before exiting." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestColoredConsoleMessageUsesPacketColor(t *testing.T) {
	console := &gameui.ChatConsole{}
	addConsoleMessage(console, nil, network.ChatMessage{Text: "Experience Gained Base:1 (0.01%) Job:1 (0.01%)", Color: 0x00B5FFB5, HasColor: true})

	messages := console.Messages()
	wantColor := color.RGBA{R: 0xB5, G: 0xFF, B: 0xB5, A: 255}
	if len(messages) != 1 || messages[0].Text != "Experience Gained Base:1 (0.01%) Job:1 (0.01%)" || messages[0].Color != wantColor {
		t.Fatalf("messages = %+v, want color %+v", messages, wantColor)
	}
}

func TestExpNotifyConsoleMessageUsesMsgStringTable(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	msgTable := strings.Repeat("ignored#\n", 1613) + "Base EXP +%d#\nJob EXP +%d#\n"
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte(msgTable), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	console := &gameui.ChatConsole{}
	addExpNotifyMessage(console, manager, network.ExpNotify{Amount: 42, VarID: network.StatusBaseExp})

	messages := console.Messages()
	wantColor := color.RGBA{R: 255, G: 255, B: 99, A: 255}
	if len(messages) != 1 || messages[0].Text != "Base EXP +42" || messages[0].Color != wantColor {
		t.Fatalf("messages = %+v, want color %+v", messages, wantColor)
	}
}

func TestQuestExpNotifyConsoleMessageUsesRobrowserText(t *testing.T) {
	console := &gameui.ChatConsole{}
	addExpNotifyMessage(console, nil, network.ExpNotify{Amount: 7, VarID: network.StatusJobExp, ExpType: 1})

	messages := console.Messages()
	wantColor := color.RGBA{R: 164, G: 66, B: 220, A: 255}
	if len(messages) != 1 || messages[0].Text != "Experience gained from Quest, Job:7" || messages[0].Color != wantColor {
		t.Fatalf("messages = %+v, want color %+v", messages, wantColor)
	}
}

func TestFormatPickupConsoleMessageUsesMsgStringAndItemName(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	msgTable := strings.Repeat("ignored#\n", 153) + "You got %s %d.#\n"
	if err := os.WriteFile(filepath.Join(dataDir, "msgstringtable.txt"), []byte(msgTable), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "idnum2itemdisplaynametable.txt"), []byte("938#Apple#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	got := formatPickupConsoleMessage(manager, network.ItemPickupAck{ItemID: 938, Amount: 2, Identified: true})
	if got != "You got Apple 2." {
		t.Fatalf("pickup message = %q", got)
	}
}

func TestFormatPickupConsoleMessageFallback(t *testing.T) {
	got := formatPickupConsoleMessage(nil, network.ItemPickupAck{ItemID: 938, Amount: 0})
	if got != "You got item 938 1." {
		t.Fatalf("pickup message = %q", got)
	}
}

func TestActorDisplayNameDoesNotLabelWarpPortal(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	for _, job := range []int16{actorJobWarpPortal, actorJobWarpPortalActive, actorJobWarpPortalWaiting} {
		actor := worldstate.Actor{Job: job}

		if got := actorDisplayName(ctx, actor, false); got != "" {
			t.Fatalf("job %d display name = %q, want empty", job, got)
		}
		if !isWarpActor(actor) {
			t.Fatalf("job %d expected warp actor classification", job)
		}
	}
}

func TestMapFadeAlphaTransitionsThroughBlack(t *testing.T) {
	start := time.Unix(100, 0)
	mode := &WorldMode{}
	mode.startMapFadeOut(network.MapChange{MapName: "geffen"}, start)

	if got := mode.mapFadeAlpha(start); got != 0 {
		t.Fatalf("fade-out start alpha = %d, want 0", got)
	}
	if got := mode.mapFadeAlpha(start.Add(mapFadeOutDuration)); got != 255 {
		t.Fatalf("fade-out end alpha = %d, want 255", got)
	}

	mode.mapFade = mapFadeState{phase: mapFadeHold, started: start}
	if got := mode.mapFadeAlpha(start.Add(time.Second)); got != 255 {
		t.Fatalf("hold alpha = %d, want 255", got)
	}
	mode.startMapPrewarm()
	if got := mode.mapFadeAlpha(start.Add(time.Second)); got != 255 {
		t.Fatalf("prewarm alpha = %d, want 255", got)
	}

	mode.startMapFadeIn(start)
	if got := mode.mapFadeAlpha(start); got != 255 {
		t.Fatalf("fade-in start alpha = %d, want 255", got)
	}
	if got := mode.mapFadeAlpha(start.Add(mapFadeInDuration)); got != 0 {
		t.Fatalf("fade-in end alpha = %d, want 0", got)
	}
}

func TestCharacterSelectTransitionFadesOutAndIn(t *testing.T) {
	ctx := client.Context{
		Resources: &res.Manager{},
		Session:   &session.Session{Playing: true},
		World:     worldstate.New(),
		UIManager: &worldModeTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode := NewWorldMode()
	mode.startCharacterSelectFadeOut(time.Now())
	if mode.mapFade.phase != mapFadeOut || !mode.mapFade.characterSelect {
		t.Fatalf("world exit fade = %+v, want character-select fade-out", mode.mapFade)
	}
	if next, err := mode.Update(ctx); err != nil || next != nil {
		t.Fatalf("mode switched before world fade reached black: next=%T err=%v", next, err)
	}

	mode.mapFade.started = time.Now().Add(-mapFadeOutDuration)
	next, err := mode.Update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if next != nil {
		t.Fatalf("mode switched before an opaque frame was presented: %T", next)
	}
	if mode.mapFade.phase != mapFadeHold {
		t.Fatalf("world fade phase = %d, want opaque hold", mode.mapFade.phase)
	}
	mode.recordCoveredMapFrame()
	next, err = mode.Update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	login, ok := next.(*LoginMode)
	if !ok {
		t.Fatalf("next mode = %T, want *LoginMode", next)
	}
	if ctx.Session.Playing {
		t.Fatal("session remained in playing state after world fade-out")
	}
	if login.fade.phase != loginFadeIn || !login.fade.started.IsZero() {
		t.Fatalf("pending character-select fade = %+v, want unstarted fade-in", login.fade)
	}

	login.Enter(ctx)
	if login.fade.started.IsZero() {
		t.Fatal("character-select fade-in did not start after Enter")
	}
	if got := login.fadeAlpha(login.fade.started); got != 255 {
		t.Fatalf("character-select fade alpha after Enter = %d, want 255", got)
	}
}

func TestMapPrewarmCompletesBeforeFadeInStarts(t *testing.T) {
	start := time.Unix(200, 0)
	mode := NewWorldMode()
	mode.startMapPrewarm()

	mode.advanceMapPrewarm(start)
	if mode.mapFade.phase != mapFadePrewarm {
		t.Fatalf("fade phase without a rendered frame = %d, want prewarm", mode.mapFade.phase)
	}

	mode.recordCoveredMapFrame()
	mode.advanceMapPrewarm(start)
	if mode.mapFade.phase != mapFadePrewarm {
		t.Fatalf("fade phase after one rendered frame = %d, want prewarm", mode.mapFade.phase)
	}

	mode.recordCoveredMapFrame()
	mode.advanceMapPrewarm(start)
	if mode.mapFade.phase != mapFadeIn {
		t.Fatalf("fade phase after prewarming = %d, want fade-in", mode.mapFade.phase)
	}
	if !mode.mapFade.started.Equal(start) {
		t.Fatalf("fade-in started = %s, want %s", mode.mapFade.started, start)
	}
}

func TestMapChangeDuringPrewarmRemainsCovered(t *testing.T) {
	mode := NewWorldMode()
	mode.startMapPrewarm()
	change := network.MapChange{MapName: "geffen"}

	mode.startMapFadeOut(change, time.Unix(300, 0))

	if mode.mapFade.phase != mapFadeHold || !mode.mapFade.hasChange {
		t.Fatalf("fade after chained map change = %+v, want covered handoff", mode.mapFade)
	}
	if mode.mapFade.change.MapName != change.MapName {
		t.Fatalf("pending map = %q, want %q", mode.mapFade.change.MapName, change.MapName)
	}
	if got := mode.mapFadeAlpha(time.Now()); got != 255 {
		t.Fatalf("chained map change alpha = %d, want 255", got)
	}
}

func TestApplyInventoryItemListReplacesExistingAmount(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 938, Amount: 3}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemList(ctx, []network.InventoryItem{{
		Index:      7,
		ItemID:     938,
		Type:       3,
		Identified: true,
		Amount:     5,
	}})

	if len(sessionState.Inventory.Items) != 1 {
		t.Fatalf("inventory item count = %d, want 1", len(sessionState.Inventory.Items))
	}
	if got := sessionState.Inventory.Items[0]; got.Amount != 5 || !got.Identified || got.Type != 3 {
		t.Fatalf("inventory item = %+v, want replaced amount/type", got)
	}
}

func TestInventoryItemDeleteDecrementsAndRemoves(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 938, Amount: 3}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemDelete(ctx, network.InventoryItemDelete{Index: 7, Amount: 2})
	if got := sessionState.Inventory.Items[0].Amount; got != 1 {
		t.Fatalf("amount after partial delete = %d, want 1", got)
	}
	applyInventoryItemDelete(ctx, network.InventoryItemDelete{Index: 7, Amount: 1})
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory item count = %d, want 0", len(sessionState.Inventory.Items))
	}
}

func TestUseItemAckSetsRemainingAmount(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 12, ItemID: 512, Amount: 4}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 3, Result: 1})
	if got := sessionState.Inventory.Items[0].Amount; got != 3 {
		t.Fatalf("item amount = %d, want 3", got)
	}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 0, Result: 1})
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory item count = %d, want 0", len(sessionState.Inventory.Items))
	}
}

func TestLegacyUseItemAckClearsConsumedItemShortcut(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 12, ItemID: 512, Amount: 1}},
		},
		Hotkeys: session.Hotkeys{
			Loaded:  true,
			Version: 1,
			Slots:   []session.HotkeySlot{{Type: network.HotkeyTypeItem, ID: 512}},
		},
	}
	ctx := client.Context{Session: sessionState}
	mode := &WorldMode{}
	mode.ui.shortcutBar.SyncFromSession(ctx)
	packet := network.Packet{
		ID:   0x00A8,
		Data: []byte{0xA8, 0x00, 0x0C, 0x00, 0x00, 0x00, 0x01},
	}

	if next, stop := mode.handleNetworkPacket(ctx, packet, time.Now()); next != nil || stop {
		t.Fatalf("use item ack changed mode: next=%T stop=%t", next, stop)
	}
	if len(sessionState.Inventory.Items) != 0 {
		t.Fatalf("inventory item count = %d, want 0", len(sessionState.Inventory.Items))
	}
	if got := sessionState.Hotkeys.Slots[0]; got.ID != 0 {
		t.Fatalf("shortcut hotkey = %+v, want empty", got)
	}
}

func TestItemIdentifyAckMarksInventoryItemIdentified(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 7, ItemID: 1201, Type: 5, Identified: false, Equip: true},
				{Index: 9, ItemID: 1202, Type: 5, Identified: false, Equip: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyItemIdentifyAck(ctx, network.ItemIdentifyAck{Index: 9, Success: true})

	if sessionState.Inventory.Items[0].Identified {
		t.Fatal("wrong item was identified")
	}
	if !sessionState.Inventory.Items[1].Identified {
		t.Fatal("target item was not identified")
	}
}

func TestItemIdentifyAckFailureDoesNotChangeInventory(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 1201, Type: 5, Identified: false, Equip: true}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyItemIdentifyAck(ctx, network.ItemIdentifyAck{Index: 7, Success: false})

	if sessionState.Inventory.Items[0].Identified {
		t.Fatal("failed identify ack changed item state")
	}
}

func TestUseItemAckFailureDoesNotChangeInventory(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 12, ItemID: 512, Amount: 4}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyUseItemAck(ctx, network.UseItemAck{Index: 12, ItemID: 512, Amount: 0, Result: 0})
	if got := sessionState.Inventory.Items[0].Amount; got != 4 {
		t.Fatalf("item amount = %d, want 4", got)
	}
}

func TestUseItemAckAddsItemUseEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 501, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectPotionRed || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestUseItemAckYellowPotionSchedulesHealSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 503, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectPotionYellow || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %d, want 1", len(mode.scheduledSounds))
	}
	sound := mode.scheduledSounds[0]
	if len(sound.paths) != 1 || sound.paths[0] != "_heal_effect.wav" {
		t.Fatalf("sound paths = %v, want _heal_effect.wav", sound.paths)
	}
	if !sound.positioned || sound.actorID != 2000000 || sound.x != 10 || sound.y != 20 {
		t.Fatalf("sound position = %+v", sound)
	}
}

func TestUseItemAckStatFoodSchedulesHealSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12041, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectStatFoodSTR || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %d, want 1", len(mode.scheduledSounds))
	}
	sound := mode.scheduledSounds[0]
	if len(sound.paths) != 1 || sound.paths[0] != "_heal_effect.wav" {
		t.Fatalf("sound paths = %v, want _heal_effect.wav", sound.paths)
	}
	if !sound.positioned || sound.actorID != 2000000 || sound.x != 10 || sound.y != 20 {
		t.Fatalf("sound position = %+v", sound)
	}
}

func TestMercenaryPotionItemEffectsTargetVisibleMercenary(t *testing.T) {
	testCases := []struct {
		name     string
		itemID   uint16
		effectID int
	}{
		{"red", 12184, effectFood},
		{"blue", 12185, effectFood},
		{"concentration", 12241, effectItemFast},
		{"awakening", 12242, effectItemFast2},
		{"berserk", 12243, effectItemFast3},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.UpsertActor(worldstate.Actor{ID: 400, X: 13, Y: 24, HasObjectType: true, ObjectType: actorObjectTypeMercenary})
			sessionState := &session.Session{
				AccountID: 2000000,
				Mercenary: session.Companion{
					ID:     400,
					Active: true,
				},
			}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world}

			mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: tc.itemID, AID: 2000000, Amount: 2, Result: 1})

			if len(mode.worldEffects) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
			}
			if effect := mode.worldEffects[0]; effect.actorID != 400 || effect.effectID != tc.effectID || effect.x != 13 || effect.y != 24 {
				t.Fatalf("effect = %+v", effect)
			}
		})
	}
}

func TestMercenaryPotionItemEffectsFallbackToAckActorWithoutMercenary(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12184, AID: 2000000, Amount: 2, Result: 1})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != effectFood || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestRemoteMercenaryPotionItemEffectKeepsAckActor(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{ID: 400, X: 13, Y: 24, HasObjectType: true, ObjectType: actorObjectTypeMercenary})
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 30, Y: 40})
	sessionState := &session.Session{
		AccountID: 2000000,
		Mercenary: session.Companion{
			ID:     400,
			Active: true,
		},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12184, AID: 1100, Amount: 2, Result: 1})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectFood || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestUseItemAckDispatchesAllMappedItemEffectArrays(t *testing.T) {
	const itemID uint16 = 65000
	itemEffectSpecs[itemID] = itemEffectSpec{
		effectIDs:         []int{effectPotionRed, effectBlessing},
		effectIDsOnCaster: []int{effectEndure},
	}
	defer delete(itemEffectSpecs, itemID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: itemID, AID: 1100, Amount: 2, Result: 1})

	want := []struct {
		effectID int
		actorID  uint32
		x        int
		y        int
	}{
		{effectPotionRed, 1100, 12, 22},
		{effectBlessing, 1100, 12, 22},
		{effectEndure, 1100, 12, 22},
	}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, wantEffect := range want {
		got := mode.worldEffects[i]
		if got.effectID != wantEffect.effectID || got.actorID != wantEffect.actorID || got.x != wantEffect.x || got.y != wantEffect.y {
			t.Fatalf("effect %d = %+v, want %+v", i, got, wantEffect)
		}
	}
}

func TestButterflyWingEffectIsPinnedAtUsePosition(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.addItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 602, AID: 2000000, Amount: 1, Result: 1})
	world.Player.X = 30
	world.Player.Y = 40

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectTeleportation || effect.x != 10 || effect.y != 20 {
		t.Fatalf("effect = %+v", effect)
	}
}
