package game

import (
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
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
	if got := charSelectPage(5); got != 1 {
		t.Fatalf("page = %d, want 1", got)
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

func TestCharacterCreateGraphDrawOrderIsValidHexagon(t *testing.T) {
	points := charCreateGraphPoints(0, 0, 64)
	order := charCreateGraphDrawOrder()
	seen := map[int]bool{}
	for _, stat := range order {
		if stat < 0 || stat >= createStatCount {
			t.Fatalf("stat index outside range in graph order: %d", stat)
		}
		if seen[stat] {
			t.Fatalf("duplicate stat index in graph order: %d", stat)
		}
		seen[stat] = true
	}

	if points[createStatDex][0] >= 0 || points[createStatDex][1] <= 0 {
		t.Fatalf("DEX graph point = %#v, want lower-left", points[createStatDex])
	}
	if points[createStatLuk][0] <= 0 || points[createStatLuk][1] <= 0 {
		t.Fatalf("LUK graph point = %#v, want lower-right", points[createStatLuk])
	}

	for i := 0; i < createStatCount; i++ {
		a1 := points[order[i]]
		a2 := points[order[(i+1)%createStatCount]]
		for j := i + 1; j < createStatCount; j++ {
			if graphEdgesAdjacent(i, j) {
				continue
			}
			b1 := points[order[j]]
			b2 := points[order[(j+1)%createStatCount]]
			if graphSegmentsCross(a1, a2, b1, b2) {
				t.Fatalf("graph edges %d and %d cross", i, j)
			}
		}
	}
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

func graphEdgesAdjacent(a, b int) bool {
	if a == b {
		return true
	}
	if a+1 == b || b+1 == a {
		return true
	}
	return (a == 0 && b == createStatCount-1) || (b == 0 && a == createStatCount-1)
}

func graphSegmentsCross(a1, a2, b1, b2 [2]float64) bool {
	o1 := graphOrientation(a1, a2, b1)
	o2 := graphOrientation(a1, a2, b2)
	o3 := graphOrientation(b1, b2, a1)
	o4 := graphOrientation(b1, b2, a2)
	return o1*o2 < 0 && o3*o4 < 0
}

func graphOrientation(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
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

	mode.drawLoginWindow(ctx)
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
	if !mode.updateFade(client.Context{}, start.Add(loginTransitionDuration)) {
		t.Fatal("world handoff did not complete at black")
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
}

type fakeCursorUIApp struct {
	cursor  widget.CursorType
	hovered widget.Widget
}

func (fakeCursorUIApp) SetUIRoot(widget.Widget) {}
func (fakeCursorUIApp) Frame()                  {}
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

	mode.drawLoginWindow(ctx)
	mode.drawLoginWindow(ctx)
	mode.updateFormInput(ctx)

	if manager.adds != 1 {
		t.Fatalf("login window AddOverlay calls = %d, want 1", manager.adds)
	}
	if manager.clears != 1 {
		t.Fatalf("login window Clear calls = %d, want 1", manager.clears)
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

	mode.publishCharacterSelectWindow(ctx)
	mode.publishCharacterSelectWindow(ctx)
	mode.updateCharacterSelectInput(ctx)

	if manager.adds != 1 {
		t.Fatalf("character select AddOverlay calls = %d, want 1", manager.adds)
	}
	if manager.clears != 1 {
		t.Fatalf("character select Clear calls = %d, want 1", manager.clears)
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

func TestLoginToWorldClearsPublishedUIRoot(t *testing.T) {
	manager := &loginTestUIManager{overlays: []widget.Widget{primitives.Box()}}
	mode := NewLoginMode()
	mode.loginWindow = &gameui.LoginWindow{}
	mode.charSelectWindow = &gameui.CharacterSelectWindow{}
	ctx := client.Context{UIManager: manager}

	if next := mode.nextWorldMode(ctx, time.Now()); next == nil {
		t.Fatal("next world mode is nil")
	}
	if len(manager.overlays) != 0 {
		t.Fatalf("login UI overlays = %d, want 0 before entering world", len(manager.overlays))
	}
	if mode.loginWindow != nil || mode.charSelectWindow != nil {
		t.Fatal("login windows were not cleared before entering world")
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

func TestLoginConfirmSFXCandidatesPreferClassicButtonSound(t *testing.T) {
	candidates := loginConfirmSFXCandidates()
	if len(candidates) < 4 {
		t.Fatalf("confirm sfx candidates = %#v", candidates)
	}
	if candidates[0] != "\xB9\xF6\xC6\xB0\xBC\xD2\xB8\xAE.wav" {
		t.Fatalf("first confirm sfx candidate = %q", candidates[0])
	}
	if candidates[2] != "click.wav" || candidates[3] != "button.wav" {
		t.Fatalf("confirm sfx fallbacks = %#v, want click/button after classic Korean sound", candidates)
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
