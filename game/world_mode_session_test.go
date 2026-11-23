package game

import (
	"encoding/binary"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func actorDisplayName(ctx client.Context, actor worldstate.Actor, isPlayer bool) string {
	labels := actors.ActorDisplayLabels(ctx, actor, isPlayer)
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

func (m *WorldMode) hoveredActorDisplayName(ctx client.Context, actor worldstate.Actor, now time.Time) string {
	labels := m.actors().HoveredActorDisplayLabels(ctx, actor, now)
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
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
	mode.ui.npc.Dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 10, Message: "Warping..."})
	if !mode.ui.npc.Dialog.Update(ctx) {
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
	next.ui.inventoryBag.Update(ctx, &next.ui.shortcutBar, &next.ui.storageWindow, &next.ui.cartWindow, nil, &next.ui.equipmentWindow, &next.ui.itemWindows)
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
	mode.ui.social.WhisperWindows.Open(ctx, "Alice")
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.social.WhisperWindows.IsOpen() {
		t.Fatal("next world mode did not carry open whisper window")
	}
	next.rebindPersistentUI(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("whisper overlays after rebind = %d, want 1", len(manager.overlays))
	}
	if manager.overlays[0] != previousOverlay {
		t.Fatal("whisper rebind replaced its stable overlay")
	}

	next.ui.social.WhisperWindows.Find("Alice").AddError(ctx, "send failed")
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
	mode.ui.social.FriendSettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("friend settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.social.FriendSettings.IsOpen() {
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
	mode.ui.social.PartySettings.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 1 {
		t.Fatalf("party settings overlays before map change = %d, want 1", len(manager.overlays))
	}
	previousOverlay := manager.overlays[0]

	next := mode.nextWorldMode()
	if !next.ui.social.PartySettings.IsOpen() {
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
	mode.ui.social.PartyCreate.Open(ctx)
	mode.ui.social.PartyInvite.Open(ctx)
	manager := ctx.UIManager.(*worldModeTestUIManager)
	if len(manager.overlays) != 2 {
		t.Fatalf("party helper overlays before map change = %d, want 2", len(manager.overlays))
	}
	previousCreate := manager.overlays[0]
	previousInvite := manager.overlays[1]

	next := mode.nextWorldMode()
	if !next.ui.social.PartyCreate.IsOpen() || !next.ui.social.PartyInvite.IsOpen() {
		t.Fatalf("next world mode did not carry helper windows create=%t invite=%t", next.ui.social.PartyCreate.IsOpen(), next.ui.social.PartyInvite.IsOpen())
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
	world.GAT = testutil.PathGAT(3, 3, nil)
	mode := &WorldMode{hoveredWalk: hoveredWalkCellCache{valid: true}}

	data := make([]byte, 24)
	binary.LittleEndian.PutUint16(data[0:2], 0x0192)
	binary.LittleEndian.PutUint16(data[2:4], 1)
	binary.LittleEndian.PutUint16(data[4:6], 2)
	binary.LittleEndian.PutUint16(data[6:8], 5)
	copy(data[8:24], "geffen.gat")
	mode.handleNetworkPacket(client.Context{World: world}, network.Packet{ID: 0x0192, Data: data}, time.Now())

	if world.GAT.Walkable(1, 2) {
		t.Fatalf("updated cell should not be walkable: %+v", world.GAT.Cells[2*world.GAT.Width+1])
	}
	if mode.hoveredWalk.valid {
		t.Fatal("hovered walk cache should be invalidated")
	}
}

func TestGuildCreationWaitsForBelongingBeforeOpeningGuildWindow(t *testing.T) {
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{PendingGuildName: "Knights"},
		World:   world,
		ScreenW: 800,
		ScreenH: 600,
	}
	var mode WorldMode

	mode.social().HandleGuildCreationResult(ctx, network.GuildCreationResult{Result: 0})

	if got := ctx.Session.GuildName; got != "" {
		t.Fatalf("session guild = %q before belonging, want empty", got)
	}
	if got := world.Player.GuildName; got != "" {
		t.Fatalf("player guild = %q before belonging, want empty", got)
	}
	if got := ctx.Session.PendingGuildName; got != "Knights" {
		t.Fatalf("pending guild = %q before belonging, want Knights", got)
	}
	if !mode.socialSystem.GuildOpenPending {
		t.Fatal("guild window open was not deferred")
	}
	if mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window opened before belonging")
	}

	mode.social().HandleGuildBelonging(ctx, network.GuildBelonging{GuildID: 9, GuildName: "Knights"})

	if got := ctx.Session.GuildID; got != 9 {
		t.Fatalf("session guild ID = %d, want 9", got)
	}
	if got := ctx.Session.GuildName; got != "Knights" {
		t.Fatalf("session guild = %q, want Knights", got)
	}
	if got := world.Player.GuildName; got != "Knights" {
		t.Fatalf("player guild = %q, want Knights", got)
	}
	if got := ctx.Session.PendingGuildName; got != "" {
		t.Fatalf("pending guild = %q after belonging, want empty", got)
	}
	if mode.socialSystem.GuildOpenPending {
		t.Fatal("deferred guild window open was not consumed")
	}
	if !mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window did not open after belonging")
	}
}

func TestGuildCreationOpensWhenBelongingArrivedFirst(t *testing.T) {
	ctx := client.Context{
		Session: &session.Session{GuildID: 9, GuildName: "Knights"},
		ScreenW: 800,
		ScreenH: 600,
	}
	var mode WorldMode

	mode.social().HandleGuildCreationResult(ctx, network.GuildCreationResult{Result: 0})

	if mode.socialSystem.GuildOpenPending {
		t.Fatal("guild window open remained pending after membership was known")
	}
	if !mode.ui.social.GuildWindow.IsOpen() {
		t.Fatal("guild window did not open with known membership")
	}
}

func TestHandleGuildNoticeUpdatesSessionAndAddsGuildConsoleMessages(t *testing.T) {
	sessionState := &session.Session{}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState}

	mode.social().HandleGuildNotice(ctx, network.GuildNotice{
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

	mode.social().HandleGuildNotice(ctx, network.GuildNotice{
		Subject: " ",
		Notice:  " Guild event tonight.\nMeet in Prontera. ",
	})

	messages := mode.ui.console.Messages()
	if len(messages) != 2 || messages[0].Text != "[ Guild event tonight. ]" || messages[1].Text != "[ Meet in Prontera. ]" {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestApplyGuildChatAddsGuildConsoleMessage(t *testing.T) {
	mode := &WorldMode{}

	social.ApplyGuildChat(network.GuildChat{Message: " Kivutar : hello guild "}, &mode.ui.console)

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Kivutar : hello guild" {
		t.Fatalf("console messages = %+v", messages)
	}
	wantColor := color.RGBA{R: 180, G: 255, B: 180, A: 255}
	if messages[0].Color != wantColor {
		t.Fatalf("guild chat color = %+v, want %+v", messages[0].Color, wantColor)
	}
}

func TestHoveredActorDisplayNameUsesServerNameForNPC(t *testing.T) {
	ctx := client.Context{Resources: &res.Manager{}}
	mode := &WorldMode{}
	actor := worldstate.Actor{
		Job:           84,
		ObjectType:    scene.ActorObjectTypeNPC,
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
		ObjectType:    scene.ActorObjectTypeMob,
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
	if mode.mapFade.Phase != maps.FadeOut || !mode.mapFade.CharacterSelect {
		t.Fatalf("world exit fade = %+v, want character-select fade-out", mode.mapFade)
	}
	if next, err := mode.Update(ctx); err != nil || next != nil {
		t.Fatalf("mode switched before world fade reached black: next=%T err=%v", next, err)
	}

	mode.mapFade.Started = time.Now().Add(-maps.FadeOutDuration)
	next, err := mode.Update(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if next != nil {
		t.Fatalf("mode switched before an opaque frame was presented: %T", next)
	}
	if mode.mapFade.Phase != maps.FadeHold {
		t.Fatalf("world fade phase = %d, want opaque hold", mode.mapFade.Phase)
	}
	mode.mapFade.RecordCoveredFrame()
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
	login.Enter(ctx)
}

func TestMapChangeDuringPrewarmRemainsCovered(t *testing.T) {
	mode := NewWorldMode()
	mode.mapFade.StartPrewarm()
	change := network.MapChange{MapName: "geffen"}

	mode.startMapFadeOut(change, time.Unix(300, 0))

	if mode.mapFade.Phase != maps.FadeHold || !mode.mapFade.HasChange {
		t.Fatalf("fade after chained map change = %+v, want covered handoff", mode.mapFade)
	}
	if mode.mapFade.Change.MapName != change.MapName {
		t.Fatalf("pending map = %q, want %q", mode.mapFade.Change.MapName, change.MapName)
	}
	if got := mode.mapFade.Alpha(time.Now()); got != 255 {
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

func TestAutoSpellListPacketOpensModalChooser(t *testing.T) {
	uiManager := &worldModeTestUIManager{}
	ctx := client.Context{
		ScreenW:   800,
		ScreenH:   600,
		UIManager: uiManager,
	}
	mode := &WorldMode{}
	data := make([]byte, 30)
	binary.LittleEndian.PutUint16(data[0:2], network.PacketZCAutoSpellList)
	binary.LittleEndian.PutUint32(data[2:6], 11)
	binary.LittleEndian.PutUint32(data[6:10], 14)

	if next, stop := mode.handleNetworkPacket(ctx, network.Packet{ID: network.PacketZCAutoSpellList, Data: data}, time.Now()); next != nil || stop {
		t.Fatalf("auto spell list changed mode: next=%T stop=%t", next, stop)
	}
	if !mode.ui.battle.AutoSpell.IsOpen() {
		t.Fatal("auto spell list did not open the chooser")
	}
	if !mode.ui.interactionModalOpen() {
		t.Fatal("auto spell chooser did not block world interactions")
	}
	if len(uiManager.overlays) != 1 {
		t.Fatalf("auto spell overlays = %d, want 1", len(uiManager.overlays))
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

func TestUseItemAckYellowPotionSchedulesHealSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.effects().AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 503, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 2000000 || effect.EffectID != effects.PotionYellow || effect.X != 10 || effect.Y != 20 {
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

	mode.effects().AddItemUseEffect(ctx, network.UseItemAck{Index: 12, ItemID: 12041, AID: 2000000, Amount: 2, Result: 1})
	if len(mode.effects().Active) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.effects().Active))
	}
	if effect := mode.effects().Active[0]; effect.ActorID != 2000000 || effect.EffectID != effects.StatFoodSTR || effect.X != 10 || effect.Y != 20 {
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

func TestMapTextureUploadsWaitForSubmissionBeforeFade(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	for _, load := range m.mapScene.TextureLoadTasks(ctx) {
		load()
	}
	m.mapFade.StartPrewarm()
	f := render.NewFrame(100, 100)
	now := time.Now()
	for retry := 0; retry < 2; retry++ {
		f.BeginFrame()
		if !m.mapScene.Images.PrepareUploads(f) || m.mapScene.Images.UploadBatch != maps.TextureUploadCount {
			t.Fatal("did not request the bounded first batch")
		}
		m.mapFade.AdvancePrewarm(now, len(m.mapScene.Images.Uploads) != 0)
		if m.mapFade.Phase != maps.FadePrewarm || len(m.mapScene.Images.Uploads) != 35 {
			t.Fatal("skipped submission advanced loading")
		}
	}
	m.FrameSubmitted()
	if len(m.mapScene.Images.Uploads) != 3 || m.mapFade.CoveredFrames != 0 {
		t.Fatal("upload counted as a rendered scene frame")
	}
	f.BeginFrame()
	m.mapScene.Images.PrepareUploads(f)
	m.FrameSubmitted()
	if len(m.mapScene.Images.Uploads) != 0 || m.mapFade.CoveredFrames != 0 {
		t.Fatal("last upload did not finish before scene warmup")
	}
	for i := 0; i < maps.FadePrewarmFrames; i++ {
		m.mapFade.AdvancePrewarm(now, len(m.mapScene.Images.Uploads) != 0)
		if m.mapFade.Phase != maps.FadePrewarm {
			t.Fatal("fade started before scene warmup finished")
		}
		m.FrameSubmitted()
	}
	m.mapFade.AdvancePrewarm(now, len(m.mapScene.Images.Uploads) != 0)
	if m.mapFade.Phase != maps.FadeIn {
		t.Fatal("fade did not start after all uploads and scene frames")
	}
}

func TestLeavingWorldReleasesMapTextures(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	for _, load := range m.mapScene.TextureLoadTasks(ctx) {
		load()
	}
	old := append([]*render.Image(nil), m.mapScene.Images.Uploads...)
	manager := &Manager{ctx: ctx, mode: m}
	manager.enter(&mapTextureExitMode{})
	for _, img := range old {
		if img.RGBA() != nil {
			t.Fatal("mode change kept old map pixels alive")
		}
	}
	if m.mapScene.Images.Images != nil || len(m.mapScene.Images.Uploads) != 0 {
		t.Fatal("mode change did not cancel pending uploads")
	}
}

type mapTextureExitMode struct{}

func (*mapTextureExitMode) Name() string { return "test" }

func (*mapTextureExitMode) Enter(client.Context) Mode { return nil }

func (*mapTextureExitMode) Update(client.Context) (Mode, error) { return nil, nil }

func (*mapTextureExitMode) Draw(client.Context, *render.Frame) {}

func TestSpriteUpscalingCachesAndReleasesOnMapChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &WorldMode{}
		defer m.Leave()
		source := render.NewImage(13, 9)
		if m.spriteTexture(source) != source || m.spriteUpscaler != nil {
			t.Fatal("disabled upscaling started work")
		}
		m.mapScene.UpscaleTextures = true
		if m.spriteTexture(source) != source {
			t.Fatal("first draw must use the original immediately")
		}
		synctest.Wait()
		upscaled := m.spriteTexture(source)
		if upscaled == source || upscaled.Bounds() != image.Rect(0, 0, 26, 18) {
			t.Fatal("background result was not adopted")
		}
		if m.spriteTexture(source) != upscaled {
			t.Fatal("repeated draws did not reuse the same texture")
		}
		m.Leave()
		synctest.Wait()
		if m.spriteUpscaler != nil || !upscaled.Bounds().Empty() || source.Bounds().Empty() {
			t.Fatal("map exit must release upscaled images without changing the originals")
		}
		// A new map can reuse an original image without reusing a released result.
		m.spriteTexture(source)
		synctest.Wait()
		if next := m.spriteTexture(source); next == upscaled || next.Bounds().Empty() {
			t.Fatal("new map reused a released texture")
		}
	})
}

func TestManagerCloseReleasesSpriteUpscaling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		world := &WorldMode{mapScene: maps.System{UpscaleTextures: true}}
		defer world.Leave()
		source := render.NewImage(16, 16)
		world.spriteTexture(source)
		synctest.Wait()
		upscaled := world.spriteTexture(source)
		if upscaled == source {
			t.Fatal("upscaled frame was not ready")
		}
		manager := &Manager{mode: world}
		manager.Close() // Game.Close uses this path when Android recreates a surface.
		manager.Close() // Closing twice must not close the worker's channel twice.
		synctest.Wait()
		if !upscaled.Bounds().Empty() || source.Bounds().Empty() {
			t.Fatal("closing must release upscaled pixels without modifying the source")
		}
		if manager.mode != nil || world.spriteUpscaler != nil || world.mapScene.Images.Images != nil {
			t.Fatal("closing retained the mode or its texture cache")
		}
	})
}

func TestSpriteUpscalingChangesTextureWithoutMovingBillboard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &WorldMode{mapScene: maps.System{UpscaleTextures: true}}
		defer m.Leave()
		base := &scene.Billboard{Image: render.NewImage(31, 23), AnchorX: 12.5, AnchorY: 19}
		billboard := actors.CartOffsetBillboard(base, 7, -3)
		projection := testutil.Projection(800, 600, 10.5, 20.5, 3)
		draw := func() reflect.Value {
			frame := render.NewFrame(800, 600)
			m.actors().DrawActorSpriteBillboardTintAlpha3D(frame, projection, billboard, 10.5, 20.5, 3, 1.7, 0.6, scene.GrayLight(0.8), color.RGBA{R: 90, G: 160, B: 255, A: 200}, render.BlendSourceOver)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != 1 {
				t.Fatal("missing billboard command")
			}
			return commands.Index(0)
		}
		original := draw()
		synctest.Wait()
		upscaled := draw()
		if original.FieldByName("Texture").Pointer() == upscaled.FieldByName("Texture").Pointer() {
			t.Fatal("render command did not use the upscaled texture")
		}
		for _, name := range []string{"Width", "Height", "AnchorX", "AnchorY", "ColorR", "ColorG", "ColorB", "ColorA", "DepthBias"} {
			if original.FieldByName(name).Float() != upscaled.FieldByName(name).Float() {
				t.Errorf("upscaling changed %s", name)
			}
		}
		for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
			for i := range 3 {
				if original.FieldByName(name).Index(i).Float() != upscaled.FieldByName(name).Index(i).Float() {
					t.Errorf("upscaling changed %s[%d]", name, i)
				}
			}
		}
		if got := m.spriteTexture(base.Image); got != m.spriteTexture(billboard.Image) {
			t.Fatal("anchor offsets duplicated the processed texture")
		}
	})
}

func mapTextureFixture(t *testing.T) (*WorldMode, client.Context) {
	t.Helper()
	return &WorldMode{mapScene: maps.System{Textures: make(map[string]*render.Image), TextureMiss: make(map[string]struct{})}}, testutil.MapTextureContext(t)
}

func TestMapInfoNotifyAddsServerRestrictionToConsole(t *testing.T) {
	mode := &WorldMode{}

	mode.applyMapInfoNotify(network.MapInfoNotify{Result: 1})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Saved point cannot be memorized." {
		t.Fatalf("console messages = %+v", messages)
	}
}
