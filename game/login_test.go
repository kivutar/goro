package game

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestLoginBackgroundSetsPrefer2008SingleImage(t *testing.T) {
	sets := loginBackgroundSets(20080910)
	if len(sets) == 0 || len(sets[0]) != 1 || sets[0][0] != "bgi_temp.bmp" {
		t.Fatalf("first 2008 login background set = %#v", sets)
	}
}

func TestLoginBackgroundSetsIncludeModernTiles(t *testing.T) {
	sets := loginBackgroundSets(20181114)
	if len(sets) == 0 || len(sets[0]) != 12 {
		t.Fatalf("first 2018 login background set = %#v", sets)
	}
}

func TestLoginInterfaceCandidatesUseROInterfacePath(t *testing.T) {
	candidates := loginInterfaceCandidates("bgi_temp.bmp")
	if len(candidates) == 0 {
		t.Fatal("no candidates")
	}
	if !strings.HasPrefix(candidates[0], "data\\texture\\") || !strings.HasSuffix(candidates[0], "\\bgi_temp.bmp") {
		t.Fatalf("first candidate = %q", candidates[0])
	}
}

func TestLoginBackgroundRealDataWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	img, source, ok := loadLoginBackgroundImage(manager, "bgi_temp.bmp")
	if !ok {
		t.Skip("bgi_temp.bmp not present in configured client data")
	}
	if img == nil || img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Fatalf("invalid login background from %s: %#v", source, img)
	}
}

func TestCharacterSelectSlotHelpers(t *testing.T) {
	characters := []session.Character{
		{ID: 10, Slot: 5},
		{ID: 11, Slot: 2},
	}
	if got := firstOccupiedCharacterSlot(characters); got != 2 {
		t.Fatalf("first occupied slot = %d, want 2", got)
	}
	if got := charSelectMaxSlots(characters); got != 9 {
		t.Fatalf("max slots = %d, want 9", got)
	}
	character, ok := characterBySlot(characters, 5)
	if !ok || character.ID != 10 {
		t.Fatalf("characterBySlot = %+v, %t", character, ok)
	}
	if got := clampCharacterSlot(99, 9); got != 8 {
		t.Fatalf("clamp high = %d, want 8", got)
	}
	if got, ok := firstEmptyCharacterSlot(characters, 9); !ok || got != 0 {
		t.Fatalf("first empty slot = %d, %t, want 0, true", got, ok)
	}
}

func TestCharacterSelectUsesConfiguredSlot(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Config: config.Config{Login: config.LoginConfig{CharSlot: 4}},
		Session: &session.Session{Characters: []session.Character{
			{ID: 10, Slot: 2},
			{ID: 11, Slot: 4},
		}},
	}

	mode.prepareCharacterSelectFromSession(ctx)

	if mode.selectedSlot != 4 {
		t.Fatalf("selected slot = %d, want configured slot 4", mode.selectedSlot)
	}
}

func TestConvertCharacterPreservesExperience(t *testing.T) {
	got := convertCharacter(network.Character{ID: 10, Exp: 123456})
	if got.ID != 10 || got.Exp != 123456 {
		t.Fatalf("converted character = %+v, want ID 10 and EXP 123456", got)
	}
}

func TestCharacterSelectWindowArrowsMoveOneSlot(t *testing.T) {
	mode := NewLoginMode()
	mode.selectedSlot = 1
	mode.maxSlots = 9
	callbacks := mode.characterSelectWindowCallbacks(client.Context{})

	callbacks.OnNextSlot()
	if mode.selectedSlot != 2 {
		t.Fatalf("right arrow selected slot = %d, want 2", mode.selectedSlot)
	}

	callbacks.OnPreviousSlot()
	if mode.selectedSlot != 1 {
		t.Fatalf("left arrow selected slot = %d, want 1", mode.selectedSlot)
	}
}

func TestLoginModeAppliesEarlySpeedParameterChange(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	ctx := client.Context{
		Session: &session.Session{},
		World:   world,
	}

	if !mode.applyLoginParameterChange(ctx, testParameterChangePacket(network.StatusSpeed, 112)) {
		t.Fatal("speed parameter change was not handled")
	}
	if !ctx.Session.Movement.HasServerSpeed || ctx.Session.Movement.ServerSpeed != 112 {
		t.Fatalf("session speed = %+v, want authoritative 112", ctx.Session.Movement)
	}
	if world.Player.Speed != 112 {
		t.Fatalf("player speed = %d, want 112", world.Player.Speed)
	}
}

func TestLoginModeAppliesStartupCartDuringWorldFade(t *testing.T) {
	mode := NewLoginMode()
	mode.startWorldFade(time.Now())
	ctx := client.Context{
		Session: &session.Session{},
		World:   worldstate.New(),
	}

	if !mode.applyLoginCartPacket(ctx, testCartAmountPacket(1, 100, 450, 80000)) {
		t.Fatal("cart amount packet was not handled")
	}
	if !ctx.Session.Cart.Open || ctx.Session.Cart.Amount != 1 || ctx.Session.Cart.MaxAmount != 100 || ctx.Session.Cart.Weight != 450 || ctx.Session.Cart.MaxWeight != 80000 {
		t.Fatalf("cart after startup packet = %+v", ctx.Session.Cart)
	}
}

func TestLoginModeAppliesSavedStatusDuringWorldFade(t *testing.T) {
	mode := NewLoginMode()
	mode.startWorldFade(time.Now())
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}}

	if !mode.applyLoginStatusEffectPacket(ctx, testStatusEffectChangePacket(db.StatusAngelus, 2000000, true)) {
		t.Fatal("Angelus status packet was not handled")
	}
	effect, ok := ctx.Session.Statuses.Active[db.StatusAngelus]
	if !ok || effect.Source != 2000000 {
		t.Fatalf("saved Angelus status = %+v, present=%t", effect, ok)
	}
}

func TestLoginModeInitialSameMapChangeDoesNotRestartWorldFade(t *testing.T) {
	mode := NewLoginMode()
	start := time.Now()
	mode.startWorldFade(start)
	change, ok, err := network.ParseMapChange(testMapChangePacket("geffen", 81, 179))
	if err != nil || !ok {
		t.Fatalf("parse map change ok=%t err=%v", ok, err)
	}
	ctx := client.Context{
		Session: &session.Session{Zone: session.ZoneServer{MapName: "geffen"}, PlayerX: 81, PlayerY: 179},
		World:   worldstate.New(),
	}
	ctx.World.MapName = "geffen"

	mode.applyLoginMapChange(ctx, change)

	if !mode.fade.enterWorld || !mode.fade.started.Equal(start) {
		t.Fatalf("fade = %+v, want original world fade", mode.fade)
	}
	if ctx.Session.PlayerX != 81 || ctx.Session.PlayerY != 179 || ctx.World.MapName != "geffen" {
		t.Fatalf("map state = %s %d,%d", ctx.World.MapName, ctx.Session.PlayerX, ctx.Session.PlayerY)
	}
}

func TestLoginMapChangeResetsWorldForChangedSelectedCharacter(t *testing.T) {
	mode := NewLoginMode()
	change, ok, err := network.ParseMapChange(testMapChangePacket("geffen", 81, 179))
	if err != nil || !ok {
		t.Fatalf("parse map change ok=%t err=%v", ok, err)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{
		ID:           150000,
		Job:          db.JobAlchemist,
		HasCartState: true,
		HasCart:      true,
		CartNum:      4,
		EffectState:  db.EffectStateCart4,
		Opt3State:    db.Opt3Quicken,
		HasState:     true,
	}
	world.Actors[300] = worldstate.Actor{ID: 300, Name: "stale actor"}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 2000000,
			CharID:    150001,
			Selected:  session.Character{ID: 150001, Name: "Wizard", Job: db.JobWizard},
		},
		World: world,
	}

	mode.applyLoginMapChange(ctx, change)

	if world.Player.ID != 150001 || world.Player.Job != db.JobWizard {
		t.Fatalf("world player = %+v, want selected wizard", world.Player)
	}
	if world.Player.HasCart || world.Player.Opt3State != 0 || world.Player.EffectState&actorEffectCartMask != 0 {
		t.Fatalf("old local state leaked after login map change: %+v", world.Player)
	}
	if len(world.Actors) != 0 {
		t.Fatalf("stale actors survived login map change: %+v", world.Actors)
	}
}

func testParameterChangePacket(varID uint16, value uint32) network.Packet {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], 0x00B0)
	binary.LittleEndian.PutUint16(data[2:4], varID)
	binary.LittleEndian.PutUint32(data[4:8], value)
	return network.Packet{ID: 0x00B0, Data: data}
}

func testCartAmountPacket(amount, maxAmount uint16, weight, maxWeight uint32) network.Packet {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], 0x0121)
	binary.LittleEndian.PutUint16(data[2:4], amount)
	binary.LittleEndian.PutUint16(data[4:6], maxAmount)
	binary.LittleEndian.PutUint32(data[6:10], weight)
	binary.LittleEndian.PutUint32(data[10:14], maxWeight)
	return network.Packet{ID: 0x0121, Data: data}
}

func testStatusEffectChangePacket(statusID uint16, actorID uint32, active bool) network.Packet {
	data := make([]byte, 9)
	binary.LittleEndian.PutUint16(data[0:2], 0x0196)
	binary.LittleEndian.PutUint16(data[2:4], statusID)
	binary.LittleEndian.PutUint32(data[4:8], actorID)
	if active {
		data[8] = 1
	}
	return network.Packet{ID: 0x0196, Data: data}
}

func testMapChangePacket(mapName string, x, y int) network.Packet {
	data := make([]byte, 22)
	binary.LittleEndian.PutUint16(data[0:2], 0x0091)
	copy(data[2:18], []byte(mapName))
	binary.LittleEndian.PutUint16(data[18:20], uint16(x))
	binary.LittleEndian.PutUint16(data[20:22], uint16(y))
	return network.Packet{ID: 0x0091, Data: data}
}

func TestAutoSelectCharacterRequiresAutologin(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Config: config.Config{Login: config.LoginConfig{CharSlot: 4}},
		Session: &session.Session{Characters: []session.Character{
			{ID: 11, Slot: 4},
		}},
	}
	mode.maxSlots = 9

	if mode.autoSelectCharacter(ctx) {
		t.Fatal("auto select ran without autologin")
	}
	if mode.autoCharAttempted {
		t.Fatal("auto select marked attempted without autologin")
	}
}

func TestCharacterSelectEnterOnEmptySlotOpensCreate(t *testing.T) {
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.selectedSlot = 2
	mode.maxSlots = 9
	inputState := input.NewState()
	inputState.SetKey(input.KeyEnter, true)
	ctx := client.Context{
		Input:     inputState,
		Resources: &res.Manager{},
		Session:   &session.Session{},
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateCharacterSelectInput(ctx)

	if mode.create.slot != 2 {
		t.Fatalf("create slot = %d, want 2", mode.create.slot)
	}
	if mode.fade.phase != loginFadeOut || !mode.fade.hasTarget || mode.fade.target != loginPhaseCreate {
		t.Fatalf("fade = %+v, want fade to character create", mode.fade)
	}
	if mode.status != "create a character" {
		t.Fatalf("status = %q, want create a character", mode.status)
	}
}

func TestCharacterSelectEnterOnOccupiedSlotSubmits(t *testing.T) {
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.selectedSlot = 1
	mode.maxSlots = 9
	inputState := input.NewState()
	inputState.SetKey(input.KeyEnter, true)
	ctx := client.Context{
		Input:     inputState,
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{Slot: 1, Name: "Alice"},
		}},
		ScreenW: 1280,
		ScreenH: 720,
	}

	mode.updateCharacterSelectInput(ctx)

	if mode.fade.phase != loginFadeNone {
		t.Fatalf("fade = %+v, want no create transition", mode.fade)
	}
	if mode.status != "select character failed: not connected" {
		t.Fatalf("status = %q, want submit path", mode.status)
	}
}

func TestLoginModeSendsCharServerPingAfterInterval(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	type acceptResult struct {
		conn net.Conn
		err  error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		conn, err := ln.Accept()
		accepted <- acceptResult{conn: conn, err: err}
	}()

	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	addr := ln.Addr().(*net.TCPAddr)
	if err := netClient.Connect(context.Background(), addr.IP.String(), addr.Port); err != nil {
		t.Fatal(err)
	}

	var serverConn net.Conn
	select {
	case result := <-accepted:
		if result.err != nil {
			t.Fatal(result.err)
		}
		serverConn = result.conn
	case <-time.After(time.Second):
		t.Fatal("server did not accept test client")
	}
	defer serverConn.Close()

	mode := NewLoginMode()
	mode.phase = loginPhaseCreate
	now := time.Unix(100, 0)
	mode.enableCharServerPing(now.Add(-charServerPingInterval))
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{AccountID: 0x11223344},
	}

	mode.maybeSendCharServerPing(ctx, now)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 6)
	if _, err := io.ReadFull(serverConn, packet); err != nil {
		t.Fatalf("reading char ping: %v", err)
	}
	if binary.LittleEndian.Uint16(packet[0:2]) != network.PacketPing {
		t.Fatalf("opcode = % x", packet[:2])
	}
	if binary.LittleEndian.Uint32(packet[2:6]) != 0x11223344 {
		t.Fatalf("account id = 0x%08x", binary.LittleEndian.Uint32(packet[2:6]))
	}
}

func TestLoginModeKeepsLoginServerAliveDuringServiceSelection(t *testing.T) {
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
	mode := NewLoginMode()
	mode.username = "Kivutar"
	mode.accountStep = loginAccountCharacterService
	now := time.Unix(100, 0)
	mode.enableLoginServerPing(now.Add(-loginServerPingInterval))

	mode.maybeSendLoginServerPing(client.Context{Network: netClient}, now)

	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 26)
	if _, err := io.ReadFull(serverConn, packet); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(packet[:2]) != network.PacketCAConnectInfoChanged {
		t.Fatalf("login keepalive opcode = 0x%04X", binary.LittleEndian.Uint16(packet[:2]))
	}
	if got := strings.TrimRight(string(packet[2:]), "\x00"); got != "Kivutar" {
		t.Fatalf("login keepalive username = %q", got)
	}
}

func TestCharacterCreateDefaultStatsAreServerValid(t *testing.T) {
	state := defaultCharCreateState(3)
	if state.slot != 3 {
		t.Fatalf("slot = %d, want 3", state.slot)
	}
	assertCreateStatsValid(t, state.stats)
}

func TestCharacterCreateBumpKeepsClassicPairsValid(t *testing.T) {
	stats := defaultCharCreateState(0).stats
	for i := 0; i < 4; i++ {
		if !bumpCreateStat(&stats, createStatStr) {
			t.Fatalf("bump STR %d failed", i)
		}
	}
	if stats[createStatStr] != 9 || stats[createStatInt] != 1 {
		t.Fatalf("STR/INT = %d/%d, want 9/1", stats[createStatStr], stats[createStatInt])
	}
	if bumpCreateStat(&stats, createStatStr) {
		t.Fatal("bump above STR limit succeeded")
	}
	assertCreateStatsValid(t, stats)
}

func TestCharacterCreateHairStyleStaysInServerRange(t *testing.T) {
	mode := NewLoginMode()
	mode.create = defaultCharCreateState(0)
	mode.create.hairStyle = charCreateMaxHairStyle
	mode.changeCreateHairStyle(1)
	if mode.create.hairStyle != charCreateMinHairStyle {
		t.Fatalf("hair wrap high = %d, want %d", mode.create.hairStyle, charCreateMinHairStyle)
	}
	mode.changeCreateHairStyle(-1)
	if mode.create.hairStyle != charCreateMaxHairStyle {
		t.Fatalf("hair wrap low = %d, want %d", mode.create.hairStyle, charCreateMaxHairStyle)
	}
}

func TestAppendCharacterNameInputLimitsBytesAndSkipsControls(t *testing.T) {
	got := appendCharacterNameInput("Kiv", "\nuta漢字", 8)
	if got != "Kivuta" {
		t.Fatalf("name input = %q, want Kivuta", got)
	}
}

func assertCreateStatsValid(t *testing.T, stats [createStatCount]uint8) {
	t.Helper()
	sum := 0
	for _, value := range stats {
		if value < 1 || value > 9 {
			t.Fatalf("stat value %d outside 1..9 in %#v", value, stats)
		}
		sum += int(value)
	}
	if sum != 30 {
		t.Fatalf("stat sum = %d, want 30 in %#v", sum, stats)
	}
	if stats[createStatStr]+stats[createStatInt] != 10 {
		t.Fatalf("STR+INT = %d, want 10", stats[createStatStr]+stats[createStatInt])
	}
	if stats[createStatAgi]+stats[createStatLuk] != 10 {
		t.Fatalf("AGI+LUK = %d, want 10", stats[createStatAgi]+stats[createStatLuk])
	}
	if stats[createStatVit]+stats[createStatDex] != 10 {
		t.Fatalf("VIT+DEX = %d, want 10", stats[createStatVit]+stats[createStatDex])
	}
}

func TestCharacterSelectPreviewFacesViewer(t *testing.T) {
	if got := spriteDirectionFromWorldDir(charSelectPreviewDirection); got != 0 {
		t.Fatalf("char select preview sprite direction = %d, want front-facing direction 0", got)
	}
	if charSelectPreviewScale <= 0.82 {
		t.Fatalf("char select preview scale = %.2f, want larger than old preview", charSelectPreviewScale)
	}
	if charSelectPreviewFeetLift <= 0 {
		t.Fatalf("char select preview feet lift = %d, want positive", charSelectPreviewFeetLift)
	}
}

func TestLoginFadeTransitionsThroughBlack(t *testing.T) {
	start := time.Unix(10, 0)
	mode := NewLoginMode()
	mode.startPhaseFade(loginPhaseCharacter, start)
	if got := mode.fadeAlpha(start); got != 0 {
		t.Fatalf("fade alpha at start = %d, want 0", got)
	}
	if mode.updateFade(client.Context{}, start.Add(loginTransitionDuration-time.Millisecond)) {
		t.Fatal("fade unexpectedly entered world")
	}
	if mode.phase != loginPhaseAccount {
		t.Fatalf("phase before black = %d, want account", mode.phase)
	}
	if got := mode.fadeAlpha(start.Add(loginTransitionDuration)); got != 255 {
		t.Fatalf("fade alpha at black = %d, want 255", got)
	}
	if mode.updateFade(client.Context{}, start.Add(loginTransitionDuration)) {
		t.Fatal("phase fade unexpectedly entered world")
	}
	if mode.phase != loginPhaseCharacter {
		t.Fatalf("phase after black = %d, want character", mode.phase)
	}
	fadeInStart := start.Add(loginTransitionDuration)
	if got := mode.fadeAlpha(fadeInStart); got != 255 {
		t.Fatalf("fade-in alpha at start = %d, want 255", got)
	}
	mode.updateFade(client.Context{}, fadeInStart.Add(loginTransitionDuration))
	if got := mode.fadeAlpha(fadeInStart.Add(loginTransitionDuration)); got != 0 {
		t.Fatalf("fade alpha after fade-in = %d, want 0", got)
	}
	if mode.fade.phase != loginFadeNone {
		t.Fatalf("fade phase after fade-in = %d, want none", mode.fade.phase)
	}
}

func TestSameLoginPhaseDoesNotRestartFadeIn(t *testing.T) {
	started := time.Unix(15, 0)
	mode := NewCharacterSelectMode(client.Context{}, gameui.ChatConsole{})
	mode.fade.started = started

	mode.startPhaseFade(loginPhaseCharacter, started.Add(100*time.Millisecond))

	if mode.fade.phase != loginFadeIn || !mode.fade.started.Equal(started) {
		t.Fatalf("same-phase request changed fade-in to %+v", mode.fade)
	}
}

func TestLoginEscapeOpensQuitConfirmation(t *testing.T) {
	mode := NewLoginMode()
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}

	inputState.SetKey(input.KeyEscape, true)
	if !mode.updatePhaseEscape(ctx, time.Unix(20, 0)) {
		t.Fatal("escape was not consumed by account phase")
	}
	if !mode.quitConfirm.IsOpen() {
		t.Fatal("quit confirmation did not open")
	}
}

func TestCredentialsEscapeReturnsToXMLConnectionSelector(t *testing.T) {
	mode := NewLoginMode()
	mode.accountStep = loginAccountCredentials
	inputState := input.NewState()
	ctx := client.Context{
		Input: inputState,
		Resources: loginTestResources(
			res.Connection{Display: "Local"},
			res.Connection{Display: "Internet"},
		),
		UIManager: &loginTestUIManager{},
		ScreenW:   800,
		ScreenH:   600,
	}
	mode.updateLoginWindow(ctx)
	inputState.SetKey(input.KeyEscape, true)

	if !mode.updatePhaseEscape(ctx, time.Unix(20, 0)) {
		t.Fatal("escape was not consumed by credentials")
	}
	if mode.quitConfirm.IsOpen() {
		t.Fatal("credentials escape opened quit confirmation instead of the server selector")
	}
	if mode.accountStep != loginAccountConnection || mode.serviceWindow == nil || mode.loginWindow != nil {
		t.Fatal("credentials escape did not restore the XML connection selector")
	}
}

func TestCharacterSelectEscapeReturnsToLogin(t *testing.T) {
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	now := time.Unix(20, 0)

	inputState.SetKey(input.KeyEscape, true)
	if !mode.updatePhaseEscape(ctx, now) {
		t.Fatal("escape was not consumed by character phase")
	}
	if mode.quitConfirm.IsOpen() {
		t.Fatal("character select escape opened quit confirmation")
	}
	if mode.fade.phase != loginFadeOut || !mode.fade.hasTarget || mode.fade.target != loginPhaseAccount {
		t.Fatalf("fade = %+v, want fade to account login", mode.fade)
	}
}

func TestCharacterCreateEscapeCancelsToSelect(t *testing.T) {
	mode := NewLoginMode()
	mode.phase = loginPhaseCreate
	mode.create = defaultCharCreateState(2)
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, ScreenW: 800, ScreenH: 600}
	now := time.Unix(20, 0)

	inputState.SetKey(input.KeyEscape, true)
	if !mode.updatePhaseEscape(ctx, now) {
		t.Fatal("escape was not consumed by create phase")
	}
	if mode.quitConfirm.IsOpen() {
		t.Fatal("character create escape opened quit confirmation")
	}
	if mode.fade.phase != loginFadeOut || !mode.fade.hasTarget || mode.fade.target != loginPhaseCharacter {
		t.Fatalf("fade = %+v, want fade to character select", mode.fade)
	}
}

func TestLoginQuitConfirmationEscapeAndEnter(t *testing.T) {
	mode := NewLoginMode()
	inputState := input.NewState()
	quit := false
	ctx := client.Context{
		Input:       inputState,
		ScreenW:     800,
		ScreenH:     600,
		RequestQuit: func() { quit = true },
	}
	mode.openQuitConfirm(ctx)

	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, true)
	if !mode.updateQuitConfirm(ctx) {
		t.Fatal("escape was not consumed")
	}
	if mode.quitConfirm.IsOpen() {
		t.Fatal("escape did not close quit confirmation")
	}
	if quit {
		t.Fatal("escape requested quit")
	}

	inputState.EndFrame()
	inputState.SetKey(input.KeyEscape, false)
	inputState.EndFrame()
	mode.openQuitConfirm(ctx)
	inputState.EndFrame()
	inputState.SetKey(input.KeyEnter, true)
	if !mode.updateQuitConfirm(ctx) {
		t.Fatal("enter was not consumed")
	}
	if mode.quitConfirm.IsOpen() {
		t.Fatal("enter did not close quit confirmation")
	}
	if !quit {
		t.Fatal("enter did not request quit")
	}
}

func TestLoginQuitConfirmationUsesSeparateOverlay(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		Resources: &res.Manager{},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateLoginWindow(ctx)
	inputState.SetKey(input.KeyEscape, true)
	if !mode.updatePhaseEscape(ctx, time.Unix(20, 0)) {
		t.Fatal("escape was not consumed")
	}
	if len(manager.overlays) != 2 {
		t.Fatalf("login overlays = %d, want login window plus confirm modal", len(manager.overlays))
	}
}

func TestLoginWorldFadeWaitsForBlack(t *testing.T) {
	start := time.Unix(20, 0)
	mode := NewLoginMode()
	mode.startWorldFade(start)
	if mode.updateFade(client.Context{}, start.Add(loginTransitionDuration-time.Millisecond)) {
		t.Fatal("world handoff completed before black")
	}
	if got := mode.fadeAlpha(start.Add(loginTransitionDuration)); got != 255 {
		t.Fatalf("world fade alpha at handoff = %d, want 255", got)
	}
	if mode.updateFade(client.Context{}, start.Add(loginTransitionDuration)) {
		t.Fatal("world handoff completed before an opaque frame was presented")
	}
	if mode.fade.phase != loginFadeHold {
		t.Fatalf("world fade phase = %d, want opaque hold", mode.fade.phase)
	}
	mode.recordCoveredLoginFrame()
	if !mode.updateFade(client.Context{}, start.Add(loginTransitionDuration)) {
		t.Fatal("world handoff did not complete after an opaque frame")
	}
}

func TestLoginToWorldPrewarmsBeforeFadeIn(t *testing.T) {
	ctx := client.Context{
		Resources: &res.Manager{},
		Session:   &session.Session{},
		World:     worldstate.New(),
	}
	next := NewLoginMode().nextWorldMode(ctx)
	next.Enter(ctx)
	if next.mapFade.phase != mapFadePrewarm {
		t.Fatalf("world fade phase after Enter = %d, want prewarm", next.mapFade.phase)
	}
	if got := next.mapFadeAlpha(time.Now()); got != 255 {
		t.Fatalf("world prewarm alpha = %d, want 255", got)
	}
}

func TestLoginCursorUsesGogpuPointerAsROHand(t *testing.T) {
	mode := NewLoginMode()
	inputState := input.NewState()
	ctx := client.Context{
		Input:   inputState,
		UIApp:   fakeCursorUIApp{cursor: widget.CursorPointer},
		ScreenW: 1280,
		ScreenH: 720,
	}

	if got := mode.cursorAction(ctx); got != cursorActionClick {
		t.Fatalf("cursor action = %d, want click hand", got)
	}
}

func TestLoginWindowUpdatesWithoutDiscoveredServers(t *testing.T) {
	mode := NewLoginMode()
	inputState := input.NewState()
	ctx := client.Context{Input: inputState, Resources: &res.Manager{}, ScreenW: 1280, ScreenH: 720}

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if mode.loginWindow == nil {
		t.Fatal("login window was not updated without discovered servers")
	}
	if mode.status != "no login servers discovered" {
		t.Fatalf("login status = %q, want no discovered servers", mode.status)
	}
}

func TestMultipleLoginConnectionsShowServerWindowBeforeLogin(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: loginTestResources(
			res.Connection{Display: "Local", Address: "127.0.0.1", Port: 6900},
			res.Connection{Display: "Internet", Address: "example.test", Port: 6900},
		),
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateAccountWindow(ctx)

	if mode.serviceWindow == nil || mode.loginWindow != nil {
		t.Fatal("multiple XML connections did not open the server selector first")
	}
	if mode.accountStep != loginAccountConnection {
		t.Fatalf("account step = %d, want connection selection", mode.accountStep)
	}
}

func TestSingleLoginConnectionOpensCredentialsDirectly(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: loginTestResources(
			res.Connection{Display: "Local", Address: "127.0.0.1", Port: 6900},
		),
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateAccountWindow(ctx)

	if mode.loginWindow == nil || mode.serviceWindow != nil {
		t.Fatal("single XML connection did not skip directly to credentials")
	}
}

func TestSelectingLoginConnectionOpensCredentialsForThatServer(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: loginTestResources(
			res.Connection{Display: "Local", Address: "127.0.0.1", Port: 6900},
			res.Connection{Display: "Internet", Address: "example.test", Port: 6900},
		),
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode.updateAccountWindow(ctx)

	mode.selectLoginServer(ctx, 1)

	if mode.selectedLoginServer != 1 {
		t.Fatalf("selected login server = %d, want 1", mode.selectedLoginServer)
	}
	if mode.accountStep != loginAccountCredentials || mode.loginWindow == nil || mode.serviceWindow != nil {
		t.Fatal("server selection did not transition to credentials")
	}
	connection, ok := mode.selectedLoginConnection(ctx)
	if !ok || connection.Address != "example.test" {
		t.Fatalf("selected connection = %+v, %t", connection, ok)
	}
}

func TestLoginUsesSelectedConnectionLangType(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	mode := NewLoginMode()
	mode.username = "Kivutar"
	mode.password = "secret"
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{},
	}

	mode.connectAndMaybeLogin(ctx, res.Connection{
		Address:  address.IP.String(),
		Port:     address.Port,
		Version:  23,
		LangType: 1,
	}, false)

	if tcpListener, ok := listener.(*net.TCPListener); ok {
		if err := tcpListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	serverConn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer serverConn.Close()
	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 55)
	if _, err := io.ReadFull(serverConn, packet); err != nil {
		t.Fatal(err)
	}
	if packet[54] != 1 {
		t.Fatalf("CA_LOGIN client type = %d, want 1", packet[54])
	}
}

func TestLoginIgnoresRepeatedSubmissionWhilePending(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	mode := NewLoginMode()
	mode.username = "Kivutar"
	mode.password = "wrong"
	ctx := client.Context{
		Network: netClient,
		Session: &session.Session{},
	}
	connection := res.Connection{
		Address: address.IP.String(),
		Port:    address.Port,
	}

	mode.connectAndMaybeLogin(ctx, connection, false)
	mode.connectAndMaybeLogin(ctx, connection, false)

	tcpListener := listener.(*net.TCPListener)
	if err := tcpListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	serverConn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer serverConn.Close()
	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(serverConn, make([]byte, 55)); err != nil {
		t.Fatal(err)
	}

	if err := tcpListener.SetDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	secondConn, err := listener.Accept()
	if err == nil {
		secondConn.Close()
		t.Fatal("repeated login submission opened a second connection")
	}
	if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
		t.Fatalf("second accept error = %v, want timeout", err)
	}
	if !mode.loginPending {
		t.Fatal("login attempt is not marked pending")
	}
}

func TestLoginClientTypeRejectsValuesOutsidePacketRange(t *testing.T) {
	for _, langType := range []int{-1, 256} {
		if got := loginClientType(langType); got != 0 {
			t.Fatalf("loginClientType(%d) = %d, want 0", langType, got)
		}
	}
	if got := loginClientType(240); got != 240 {
		t.Fatalf("loginClientType(240) = %d, want 240", got)
	}
}

func TestAutologinSkipsXMLConnectionSelector(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Config: config.Config{Login: config.LoginConfig{AutoLogin: true}},
		Resources: loginTestResources(
			res.Connection{Display: "Local"},
			res.Connection{Display: "Internet"},
		),
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateAccountWindow(ctx)

	if mode.loginWindow == nil || mode.serviceWindow != nil {
		t.Fatal("autologin did not bypass the XML connection selector")
	}
}

func TestAccountAcceptShowsCharacterServiceWithPlayerCount(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: &res.Manager{},
		Session:   &session.Session{},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.applyAccountAcceptLogin(ctx, network.AccountAcceptLogin{
		AccountID: 2000000,
		AuthCode:  123,
		Sex:       1,
		CharServer: []network.CharServer{
			{Name: "Chaos", Address: "127.0.0.1", Port: 6121, UserCount: 42},
		},
	})

	if mode.accountStep != loginAccountCharacterService {
		t.Fatalf("account step = %d, want character service", mode.accountStep)
	}
	if mode.serviceWindow == nil || mode.loginWindow != nil {
		t.Fatal("account acceptance did not replace credentials with the character-service list")
	}
	if len(ctx.Session.CharServers) != 1 || ctx.Session.CharServers[0].UserCount != 42 {
		t.Fatalf("character servers = %+v", ctx.Session.CharServers)
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("login overlays = %d, want character-service window only", len(manager.overlays))
	}
}

func TestCharacterServiceNameMatchesClassicServerList(t *testing.T) {
	tests := []struct {
		name   string
		server session.CharServer
		want   string
	}{
		{name: "players", server: session.CharServer{Name: "Chaos", UserCount: 42}, want: "Chaos (42)"},
		{name: "maintenance", server: session.CharServer{Name: "Chaos", UserCount: 42, State: 1}, want: "Chaos (On maintenance)"},
		{name: "new", server: session.CharServer{Name: "Chaos", UserCount: 42, Property: 1}, want: "New Chaos (42)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := characterServiceName(nil, test.server); got != test.want {
				t.Fatalf("character service label = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCharacterServiceCancelReturnsToCredentials(t *testing.T) {
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: &res.Manager{},
		Session: &session.Session{CharServers: []session.CharServer{
			{Name: "Chaos", UserCount: 42},
		}},
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode.showCharacterServiceSelection(ctx)

	mode.cancelCharacterServiceSelection(ctx)

	if mode.accountStep != loginAccountCredentials {
		t.Fatalf("account step = %d, want credentials", mode.accountStep)
	}
	if mode.serviceWindow != nil || mode.loginWindow == nil {
		t.Fatal("character-service cancel did not restore credentials")
	}
}

func TestCharacterServiceSelectionConnectsChosenServer(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	address := listener.Addr().(*net.TCPAddr)
	netClient := network.NewClient(20080910, false)
	defer netClient.Close()
	mode := NewLoginMode()
	ctx := client.Context{
		Network:   netClient,
		Resources: &res.Manager{},
		Session: &session.Session{
			AccountID: 0x11223344,
			AuthCode:  0x55667788,
			UserLevel: 3,
			Sex:       1,
			CharServers: []session.CharServer{
				{Name: "Wrong", Address: "127.0.0.1", Port: 1},
				{Name: "Chaos", Address: address.IP.String(), Port: uint16(address.Port)},
			},
		},
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode.updateLoginWindow(ctx)

	mode.selectCharacterService(ctx, 1, false)

	if mode.accountStep != loginAccountCharacterConnecting {
		t.Fatalf("account step = %d, want character connection", mode.accountStep)
	}
	if ctx.Session.CharServerIndex != 1 {
		t.Fatalf("selected character server = %d, want 1", ctx.Session.CharServerIndex)
	}
	if mode.loginWindow != nil || mode.serviceWindow != nil {
		t.Fatal("account windows remained visible while connecting to the character server")
	}
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		if err := tcpListener.SetDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	serverConn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	defer serverConn.Close()
	if err := serverConn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	packet := make([]byte, 17)
	if _, err := io.ReadFull(serverConn, packet); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(packet[:2]) != network.PacketCAEnter {
		t.Fatalf("char-server packet opcode = 0x%04X", binary.LittleEndian.Uint16(packet[:2]))
	}
	if binary.LittleEndian.Uint32(packet[2:6]) != ctx.Session.AccountID {
		t.Fatalf("char-server packet account = 0x%08X", binary.LittleEndian.Uint32(packet[2:6]))
	}
}

func loginTestResources(connections ...res.Connection) *res.Manager {
	return &res.Manager{ClientInfo: res.ClientInfo{Connections: connections}}
}

type fakeCursorUIApp struct {
	cursor  widget.CursorType
	hovered widget.Widget
}

func (fakeCursorUIApp) SetUIRoot(widget.Widget) {}
func (fakeCursorUIApp) Frame()                  {}
func (fakeCursorUIApp) Invalidate()             {}
func (a fakeCursorUIApp) Cursor() widget.CursorType {
	return a.cursor
}

func (a fakeCursorUIApp) HoveredWidget() widget.Widget {
	return a.hovered
}

type loginTestUIManager struct {
	overlays []widget.Widget
	adds     int
	clears   int
}

func (m *loginTestUIManager) AddOverlay(root widget.Widget) {
	m.overlays = append(m.overlays, root)
	m.adds++
}

func (m *loginTestUIManager) RemoveOverlay(root widget.Widget) {
	for i, overlay := range m.overlays {
		if overlay == root {
			m.overlays = append(m.overlays[:i], m.overlays[i+1:]...)
			return
		}
	}
}

func (m *loginTestUIManager) Clear() {
	m.overlays = nil
	m.clears++
}

func TestLoginWindowPublishesOnlyWhenWidgetChanges(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	ctx := client.Context{
		Resources: &res.Manager{},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.updateLoginWindow(ctx)
	mode.updateLoginWindow(ctx)
	mode.updateAccountInput(ctx)

	if manager.adds != 1 {
		t.Fatalf("login window AddOverlay calls = %d, want 1", manager.adds)
	}
	if manager.clears != 0 {
		t.Fatalf("login window Clear calls = %d, want 0", manager.clears)
	}
}

func TestLoginDrawDoesNotReplaceCharacterWindowContent(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.maxSlots = 9
	ctx := client.Context{
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{Slot: 0, Name: "Saved", Level: 1, JobLevel: 1},
		}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.showCharacterSelectWindow(ctx)
	root := mode.charSelectWindow.Widget()
	widget.ClearRedrawInTree(root)
	ctx.Session.Characters = []session.Character{
		{Slot: 0, Name: "Refreshed", Level: 1, JobLevel: 1},
	}

	mode.Draw(ctx, render.NewFrame(ctx.ScreenW, ctx.ScreenH))
	if widget.NeedsRedrawInTree(root) {
		t.Fatal("Draw mutated the character-select widget tree after the UI layout phase")
	}

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if !widget.NeedsRedrawInTree(root) {
		t.Fatal("Update did not schedule the refreshed character-select content")
	}
}

func TestCharacterSelectPublishesOnlyWhenWidgetChanges(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.maxSlots = 9
	ctx := client.Context{
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{Slot: 0, Name: "Kivutar", Level: 1, JobLevel: 1},
		}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.showCharacterSelectWindow(ctx)
	mode.showCharacterSelectWindow(ctx)
	mode.updateCharacterSelectInput(ctx)

	if manager.adds != 1 {
		t.Fatalf("character select AddOverlay calls = %d, want 1", manager.adds)
	}
	if manager.clears != 0 {
		t.Fatalf("character select Clear calls = %d, want 0", manager.clears)
	}
}

func TestCharacterSelectPublishesUIRootDuringFadeIn(t *testing.T) {
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.fade = loginFadeState{phase: loginFadeIn, started: time.Now()}
	mode.maxSlots = 9
	ctx := client.Context{
		Input:     input.NewState(),
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{ID: 1, Slot: 0, Name: "Kivutar", Level: 1, JobLevel: 1},
		}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("character select overlays = %d, want 1 during fade-in", len(manager.overlays))
	}
}

func TestDeleteCharacterAcceptRemovesSelectedCharacter(t *testing.T) {
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	mode.selectedSlot = 1
	mode.deleteCharID = 20
	ctx := client.Context{
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{ID: 10, Slot: 0, Name: "Kivutar"},
			{ID: 20, Slot: 1, Name: "DeleteMe"},
		}},
		UIManager: &loginTestUIManager{},
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.applyDeleteCharacterAccept(ctx)

	if len(ctx.Session.Characters) != 1 || ctx.Session.Characters[0].ID != 10 {
		t.Fatalf("characters = %+v", ctx.Session.Characters)
	}
	if mode.selectedSlot != 0 {
		t.Fatalf("selectedSlot = %d, want 0", mode.selectedSlot)
	}
	if mode.deleteCharID != 0 {
		t.Fatalf("deleteCharID = %d, want 0", mode.deleteCharID)
	}
}

func TestLoginToWorldClearsPublishedUIRoot(t *testing.T) {
	manager := &loginTestUIManager{overlays: []widget.Widget{primitives.Box()}}
	mode := NewLoginMode()
	mode.loginWindow = &gameui.LoginWindow{}
	mode.charSelectWindow = &gameui.CharacterSelectWindow{}
	ctx := client.Context{UIManager: manager}

	if next := mode.nextWorldMode(ctx); next == nil {
		t.Fatal("next world mode is nil")
	}
	if len(manager.overlays) != 0 {
		t.Fatalf("login UI overlays = %d, want 0 before entering world", len(manager.overlays))
	}
	if mode.loginWindow != nil || mode.charSelectWindow != nil {
		t.Fatal("login windows were not cleared before entering world")
	}
}

func TestLoginActorBootstrapRestoresWeddingVisualState(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard},
		Characters: []session.Character{
			{ID: 150000, Job: db.JobWizard},
		},
	}
	ctx := client.Context{Session: sessionState, World: world}
	data := make([]byte, 13)
	binary.LittleEndian.PutUint16(data[0:2], 0x0119)
	binary.LittleEndian.PutUint32(data[2:6], sessionState.AccountID)
	binary.LittleEndian.PutUint16(data[10:12], uint16(db.EffectStateWedding))

	if !mode.applyLoginActorBootstrapPacket(ctx, network.Packet{ID: 0x0119, Data: data}) {
		t.Fatal("wedding state packet was not handled during login")
	}
	if world.Player.EffectState&db.EffectStateWedding == 0 {
		t.Fatalf("player effect state = 0x%08X, want wedding", world.Player.EffectState)
	}
	if sessionState.Selected.Option&db.EffectStateWedding == 0 {
		t.Fatalf("selected option = 0x%08X, want wedding", sessionState.Selected.Option)
	}
	if got := localPlayerVisualJob(ctx); got != db.JobMarried {
		t.Fatalf("visual job = %d, want married job %d", got, db.JobMarried)
	}
}

func TestLoginActorBootstrapRestoresLocalLook(t *testing.T) {
	mode := NewLoginMode()
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 150000, Job: db.JobWizard}
	sessionState := &session.Session{
		AccountID: 2000000,
		CharID:    150000,
		Selected:  session.Character{ID: 150000, Job: db.JobWizard, Weapon: 1201},
	}
	ctx := client.Context{Session: sessionState, World: world}
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data[0:2], 0x01D7)
	binary.LittleEndian.PutUint32(data[2:6], sessionState.AccountID)
	data[6] = 2

	if !mode.applyLoginActorBootstrapPacket(ctx, network.Packet{ID: 0x01D7, Data: data}) {
		t.Fatal("look packet was not handled during login")
	}
	if sessionState.Selected.Weapon != 0 || world.Player.Weapon != 0 {
		t.Fatalf("restored look = selected weapon %d, player weapon %d", sessionState.Selected.Weapon, world.Player.Weapon)
	}
}

func TestCharacterSelectModePublishesRootOnEnter(t *testing.T) {
	staleRoot := primitives.Box()
	manager := &loginTestUIManager{overlays: []widget.Widget{staleRoot}}
	mode := NewCharacterSelectMode(client.Context{}, gameui.ChatConsole{})
	ctx := client.Context{
		Resources: &res.Manager{},
		Session: &session.Session{Characters: []session.Character{
			{ID: 1, Slot: 0, Name: "Kivutar", Level: 1, JobLevel: 1},
		}},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}

	mode.Enter(ctx)

	if len(manager.overlays) != 1 {
		t.Fatalf("character select mode overlays = %d, want 1 on enter", len(manager.overlays))
	}
	if manager.overlays[0] == staleRoot {
		t.Fatal("character select mode left stale UI root published")
	}
}

func TestCharacterSelectBackToLoginPublishesLoginRootAtFadeSwitch(t *testing.T) {
	start := time.Unix(30, 0)
	manager := &loginTestUIManager{}
	mode := NewLoginMode()
	mode.phase = loginPhaseCharacter
	ctx := client.Context{
		Resources: &res.Manager{},
		Input:     input.NewState(),
		Session:   &session.Session{},
		UIManager: manager,
		ScreenW:   1280,
		ScreenH:   720,
	}
	mode.charSelectWindow = gameui.NewCharacterSelectWindow(ctx, gameui.CharacterSelectWindowOptions{}, gameui.CharacterSelectWindowCallbacks{})
	staleRoot := mode.charSelectWindow.Widget()
	manager.overlays = []widget.Widget{staleRoot}
	mode.startPhaseFade(loginPhaseAccount, start)

	if mode.updateFade(ctx, start.Add(loginTransitionDuration)) {
		t.Fatal("phase fade unexpectedly entered world")
	}
	if mode.phase != loginPhaseAccount {
		t.Fatalf("phase = %d, want account", mode.phase)
	}
	if len(manager.overlays) != 1 {
		t.Fatalf("login overlays = %d, want 1 at phase switch", len(manager.overlays))
	}
	if manager.overlays[0] == staleRoot {
		t.Fatal("stale character select root stayed published after returning to login")
	}
	if mode.loginWindow == nil {
		t.Fatal("login window was not rebuilt at phase switch")
	}
	if mode.charSelectWindow != nil {
		t.Fatal("character select window was not cleared at phase switch")
	}
}

func TestLoginConfirmSFXUsesClassicButtonSound(t *testing.T) {
	if loginConfirmSFX != "버튼소리.wav" {
		t.Fatalf("confirm sfx = %q", loginConfirmSFX)
	}
}

func TestCharacterSelectSkinRealDataWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	for _, name := range []string{"login_interface/win_select.bmp", "login_interface/box_select.bmp"} {
		img, source, ok := loadLoginBackgroundImage(manager, name)
		if !ok {
			t.Skipf("%s not present in configured client data", name)
		}
		if img == nil || img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
			t.Fatalf("invalid char select skin from %s: %#v", source, img)
		}
	}
}
