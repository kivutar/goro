package ui

import (
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type minimapTestUIApp struct {
	invalidates int
}

func (a *minimapTestUIApp) SetUIRoot(widget.Widget) {}
func (a *minimapTestUIApp) Frame()                  {}
func (a *minimapTestUIApp) Invalidate()             { a.invalidates++ }
func (a *minimapTestUIApp) Cursor() widget.CursorType {
	return widget.CursorDefault
}
func (a *minimapTestUIApp) HoveredWidget() widget.Widget {
	return nil
}

func TestNormalizeMinimapMapName(t *testing.T) {
	if got := normalizeMinimapMapName(`data\prontera.rsw`); got != "prontera" {
		t.Fatalf("normalized map = %q, want prontera", got)
	}
	if got := normalizeMinimapMapName("izlude.gat"); got != "izlude" {
		t.Fatalf("normalized gat = %q, want izlude", got)
	}
}

func TestMinimapDragsFromTitleBar(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	inputState := input.NewState()
	manager := &escapeMenuTestUIManager{}
	app := &windowDragTestApp{}
	ctx := Context{
		World:     world,
		Input:     inputState,
		UIApp:     app,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}
	m := &Minimap{}
	if m.Update(ctx) {
		t.Fatal("initial minimap update was consumed")
	}
	startX, startY, _, _ := minimapBounds(ctx.ScreenW, ctx.ScreenH)

	inputState.SetMousePosition(startX+10, startY+minimapHeight-10)
	if m.Update(ctx) {
		t.Fatal("minimap hover was reported as an active drag")
	}

	inputState.SetMousePosition(startX+12, startY+10)
	inputState.SetMouseButton(input.MouseButtonLeft, true)
	if !m.Update(ctx) {
		t.Fatal("minimap title press did not start a drag")
	}
	root := m.window.published
	if app.beginToken != root {
		t.Fatal("minimap drag did not use the shared window drag layer")
	}
	overlay := root.(*positionedOverlay)
	widget.ClearRedrawInTree(root)
	app.rects = nil

	inputState.EndFrame()
	inputState.SetMousePosition(startX-28, startY+70)
	if !m.Update(ctx) {
		t.Fatal("minimap drag move was not consumed")
	}
	wantX, wantY := startX-40, startY+60
	if m.window.x != wantX || m.window.y != wantY {
		t.Fatalf("minimap position = %d,%d; want %d,%d", m.window.x, m.window.y, wantX, wantY)
	}
	if overlay.NeedsRedraw() {
		t.Fatal("minimap drag move dirtied the UI overlay")
	}
	if len(app.rects) != 0 {
		t.Fatalf("minimap drag move invalidated %d UI rects; want 0", len(app.rects))
	}

	inputState.EndFrame()
	inputState.SetMouseButton(input.MouseButtonLeft, false)
	if !m.Update(ctx) {
		t.Fatal("minimap drag release was not consumed")
	}
	if app.endToken != root {
		t.Fatal("minimap drag release did not clear the shared drag layer")
	}

	inputState.EndFrame()
	ctx.ScreenW = 1000
	if m.Update(ctx) {
		t.Fatal("idle minimap update was reported as an active drag")
	}
	if m.window.x != wantX || m.window.y != wantY {
		t.Fatalf("minimap position after resize = %d,%d; want dragged position %d,%d", m.window.x, m.window.y, wantX, wantY)
	}
}

func TestMinimapCellToScreenInvertsY(t *testing.T) {
	rect := minimapRect{x: 10, y: 20, w: 100, h: 100}

	_, topY, ok := minimapCellToScreen(rect, 10, 10, 5, 9)
	if !ok {
		t.Fatal("top cell did not project")
	}
	_, bottomY, ok := minimapCellToScreen(rect, 10, 10, 5, 0)
	if !ok {
		t.Fatal("bottom cell did not project")
	}
	if topY >= bottomY {
		t.Fatalf("topY=%d bottomY=%d, want map Y inverted onto screen", topY, bottomY)
	}
}

func TestMinimapProjectedMapRectCentersShortAxis(t *testing.T) {
	rect := minimapProjectedMapRect(minimapRect{x: 0, y: 0, w: 128, h: 128}, 64, 128)
	if rect.x != 32 || rect.y != 0 || rect.w != 64 || rect.h != 128 {
		t.Fatalf("projected rect = %+v, want x=32 y=0 w=64 h=128", rect)
	}

	rect = minimapProjectedMapRect(minimapRect{x: 0, y: 0, w: 128, h: 128}, 128, 64)
	if rect.x != 0 || rect.y != 32 || rect.w != 128 || rect.h != 64 {
		t.Fatalf("projected rect = %+v, want x=0 y=32 w=128 h=64", rect)
	}
}

func TestMinimapCellToScreenUsesCenteredProjection(t *testing.T) {
	rect := minimapRect{x: 0, y: 0, w: 128, h: 128}
	leftX, midY, ok := minimapCellToScreen(rect, 64, 128, 0, 64)
	if !ok {
		t.Fatal("left cell did not project")
	}
	rightX, _, ok := minimapCellToScreen(rect, 64, 128, 63, 64)
	if !ok {
		t.Fatal("right cell did not project")
	}
	if leftX != 32 || rightX != 95 || midY != 64 {
		t.Fatalf("projected points = leftX:%d rightX:%d midY:%d, want 32,95,64", leftX, rightX, midY)
	}
}

func TestMinimapUpdateDefersPlayerMarkerRedraw(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	manager := &escapeMenuTestUIManager{}
	app := &minimapTestUIApp{}
	ctx := Context{
		World:     world,
		Input:     input.NewState(),
		UIApp:     app,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}
	m := &Minimap{}
	m.Update(ctx)
	if m.widget == nil {
		t.Fatal("minimap widget was not created")
	}
	widget.ClearRedrawInTree(m.window.published)
	app.invalidates = 0

	world.SetPlayerPosition(11, 20, 4)
	m.Update(ctx)
	if m.widget.NeedsRedraw() {
		t.Fatal("player marker move dirtied minimap in the same update")
	}
	if !m.pendingMarker {
		t.Fatal("player marker move did not queue a deferred redraw")
	}
	if app.invalidates != 0 {
		t.Fatal("queued player marker redraw invalidated UI in the same update")
	}

	m.Update(ctx)
	if !m.widget.NeedsRedraw() {
		t.Fatal("deferred player marker move did not dirty minimap")
	}
	if app.invalidates == 0 {
		t.Fatal("deferred player marker move did not invalidate UI app")
	}
	widget.ClearRedrawInTree(m.window.published)
	app.invalidates = 0

	world.SetPlayerPosition(12, 20, 4)
	m.Update(ctx)
	if m.widget.NeedsRedraw() {
		t.Fatal("second player marker move dirtied minimap in the same update")
	}
	m.Update(ctx)
	if !m.widget.NeedsRedraw() {
		t.Fatal("second deferred player marker move did not dirty minimap")
	}
	if app.invalidates == 0 {
		t.Fatal("second deferred player marker move did not invalidate UI app")
	}
	widget.ClearRedrawInTree(m.window.published)
	app.invalidates = 0

	world.SetPlayerPosition(12, 20, 5)
	m.Update(ctx)
	if m.widget.NeedsRedraw() {
		t.Fatal("player marker direction change dirtied minimap in the same update")
	}
	m.Update(ctx)
	if !m.widget.NeedsRedraw() {
		t.Fatal("deferred player marker direction change did not dirty minimap")
	}
	if app.invalidates == 0 {
		t.Fatal("deferred player marker direction change did not invalidate UI app")
	}
}

func TestMinimapUpdateRedrawsWhenVisiblePartyMarkerChanges(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	world.SetPlayerPosition(10, 20, 4)
	sessionState := &session.Session{
		AccountID: 1,
		Party: session.Party{
			Name: "Party",
			Members: []session.PartyMember{
				{AccountID: 2, Name: "Alice", MapName: "prontera", State: 0, X: 30, Y: 40},
			},
		},
	}
	app := &minimapTestUIApp{}
	ctx := Context{
		Session:   sessionState,
		World:     world,
		Input:     input.NewState(),
		UIApp:     app,
		UIManager: &escapeMenuTestUIManager{},
		ScreenW:   800,
		ScreenH:   600,
	}
	m := &Minimap{}
	m.Update(ctx)
	widget.ClearRedrawInTree(m.window.published)
	app.invalidates = 0

	sessionState.Party.Members[0].X = 31
	m.Update(ctx)
	if !m.widget.NeedsRedraw() {
		t.Fatal("party marker move did not dirty minimap")
	}
	if app.invalidates == 0 {
		t.Fatal("party marker move did not invalidate UI app")
	}
}

func TestMinimapDrawDoesNotPaintInnerMapChrome(t *testing.T) {
	w := newMinimapWidget()
	w.SetBounds(geometry.NewRect(0, 0, minimapWidth, minimapHeight-ROWindowTitleHeight))
	canvas := &uitest.MockCanvas{}
	w.Draw(widget.NewContext(), canvas)

	if len(canvas.Rects) != 0 {
		t.Fatalf("minimap drew %d inner background rects, want none", len(canvas.Rects))
	}
	if len(canvas.StrokeRects) != 0 {
		t.Fatalf("minimap drew %d inner border rects, want none", len(canvas.StrokeRects))
	}
}

func TestMinimapWindowBackgroundIsTransparent(t *testing.T) {
	m := &Minimap{}
	m.ensureWindow(minimapWidth, minimapHeight)

	if m.window.background == nil || !m.window.background.IsTransparent() {
		t.Fatal("minimap window background is not transparent")
	}
}

func TestMinimapImageCandidatesIncludeROPaths(t *testing.T) {
	candidates := minimapImageCandidates("prontera")
	if len(candidates) == 0 {
		t.Fatal("no minimap candidates")
	}
	want := "data\\texture\\interface\\map\\prontera.bmp"
	for _, candidate := range candidates {
		if candidate == want {
			return
		}
	}
	t.Fatalf("candidate %q missing from %#v", want, candidates)
}

func TestMinimapArrowCandidatesIncludeROPath(t *testing.T) {
	candidates := minimapArrowCandidates()
	want := "data\\texture\\유저인터페이스\\map\\map_arrow.bmp"
	for _, candidate := range candidates {
		if candidate == want {
			return
		}
	}
	t.Fatalf("candidate %q missing from %#v", want, candidates)
}

func TestMinimapPlayerArrowMatchesROBrowserAngle(t *testing.T) {
	cases := []struct {
		name string
		dir  int
		want float64
	}{
		{name: "north", dir: 0, want: 0},
		{name: "northwest", dir: 1, want: 7 * math.Pi / 4},
		{name: "west", dir: 2, want: 3 * math.Pi / 2},
		{name: "southwest", dir: 3, want: 5 * math.Pi / 4},
		{name: "south", dir: 4, want: math.Pi},
		{name: "southeast", dir: 5, want: 3 * math.Pi / 4},
		{name: "east", dir: 6, want: math.Pi / 2},
		{name: "northeast", dir: 7, want: math.Pi / 4},
	}
	for _, tc := range cases {
		if got := minimapPlayerArrowAngle(tc.dir); math.Abs(got-tc.want) > 0.000001 {
			t.Fatalf("%s arrow angle = %f, want %f", tc.name, got, tc.want)
		}
	}
}

func TestMinimapPlayerArrowCachesDirectionVariants(t *testing.T) {
	m := &Minimap{arrow: image.NewNRGBA(image.Rect(0, 0, 5, 3))}
	first := m.playerArrow(0)
	second := m.playerArrow(8)
	if first == nil || second == nil {
		t.Fatal("player arrow variant was not created")
	}
	if first != second {
		t.Fatal("normalized direction should reuse cached player arrow variant")
	}
	if third := m.playerArrow(2); third == nil || third == first {
		t.Fatal("different direction should create a distinct player arrow variant")
	}
}

func TestMinimapCompassLifecycle(t *testing.T) {
	now := time.Unix(100, 0)
	m := &Minimap{}
	m.ApplyCompass(7, 0, 10, 20, 0x00AABBCC, now)
	if len(m.compass) != 1 {
		t.Fatalf("compass markers = %d, want 1", len(m.compass))
	}
	marker := m.compass[7]
	if marker.typ != 0 || marker.x != 10 || marker.y != 20 {
		t.Fatalf("marker = %+v, want decoded location", marker)
	}
	if marker.color != (color.RGBA{R: 0xAA, G: 0xBB, B: 0xCC, A: 255}) {
		t.Fatalf("marker color = %+v", marker.color)
	}
	if !marker.expires.Equal(now.Add(minimapCompassDuration)) {
		t.Fatalf("marker expires = %v, want %v", marker.expires, now.Add(minimapCompassDuration))
	}
	if m.pruneCompassMarkers(now.Add(minimapCompassDuration - time.Millisecond)) {
		t.Fatal("temporary compass marker pruned too early")
	}
	if !m.pruneCompassMarkers(now.Add(minimapCompassDuration)) {
		t.Fatal("temporary compass marker was not pruned at expiry")
	}

	m.ApplyCompass(8, 1, 30, 40, 0x00010203, now)
	if m.pruneCompassMarkers(now.Add(time.Hour)) {
		t.Fatal("persistent compass marker should not expire")
	}
	m.ApplyCompass(8, 2, 0, 0, 0, now)
	if len(m.compass) != 0 {
		t.Fatalf("compass markers after remove = %d, want 0", len(m.compass))
	}
}

func TestMinimapGuildMarkerLifecycle(t *testing.T) {
	m := &Minimap{}
	m.ApplyGuildMemberPosition(20, 30, 40)
	if m.guildRevision != 1 || len(m.guild) != 1 {
		t.Fatalf("first guild marker revision=%d markers=%+v", m.guildRevision, m.guild)
	}
	m.ApplyGuildMemberPosition(20, 30, 40)
	if m.guildRevision != 1 {
		t.Fatalf("unchanged guild marker advanced revision to %d", m.guildRevision)
	}
	m.ApplyGuildMemberPosition(10, 11, 12)
	snapshot := m.guildSnapshot()
	if len(snapshot) != 2 || snapshot[0].accountID != 10 || snapshot[1].accountID != 20 {
		t.Fatalf("guild marker snapshot = %+v", snapshot)
	}
	if cached := m.guildSnapshot(); &cached[0] != &snapshot[0] {
		t.Fatal("unchanged guild marker snapshot was allocated again")
	}
	m.ApplyGuildMemberPosition(20, 31, 41)
	updated := m.guildSnapshot()
	if &updated[0] != &snapshot[0] {
		t.Fatal("changed guild marker snapshot did not reuse its allocation")
	}
	m.ApplyGuildMemberPosition(20, -1, -1)
	if m.guildRevision != 4 || len(m.guild) != 1 {
		t.Fatalf("removed guild marker revision=%d markers=%+v", m.guildRevision, m.guild)
	}
	m.ApplyGuildMemberPosition(20, -1, -1)
	if m.guildRevision != 4 {
		t.Fatalf("missing marker removal advanced revision to %d", m.guildRevision)
	}
	if !m.clearGuildMarkers() || len(m.guild) != 0 || m.guildRevision != 5 {
		t.Fatalf("clear guild markers revision=%d markers=%+v", m.guildRevision, m.guild)
	}
}

func TestMinimapMapChangeClearsGuildMarkers(t *testing.T) {
	world := worldstate.New()
	world.MapName = "prontera"
	ctx := Context{
		Session:   &session.Session{AccountID: 1},
		World:     world,
		Input:     input.NewState(),
		UIApp:     &minimapTestUIApp{},
		UIManager: &escapeMenuTestUIManager{},
		ScreenW:   800,
		ScreenH:   600,
	}
	m := &Minimap{}
	m.Update(ctx)
	m.ApplyGuildMemberPosition(2, 10, 20)
	world.MapName = "payon"
	m.Update(ctx)
	if len(m.guild) != 0 {
		t.Fatalf("guild markers survived map change: %+v", m.guild)
	}
}

func TestMinimapMemberColorStable(t *testing.T) {
	first := minimapMemberColor(1234)
	second := minimapMemberColor(1234)
	if first != second {
		t.Fatalf("member color changed: %+v != %+v", first, second)
	}
	if first.A != 255 {
		t.Fatalf("member color alpha = %d, want 255", first.A)
	}
}
