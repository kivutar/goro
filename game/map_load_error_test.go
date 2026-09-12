package game

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func mapRecoveryTestContext(t *testing.T) client.Context {
	t.Helper()
	world := worldstate.New()
	world.MapName = "new_1-1.gat"
	return client.Context{
		Resources: &res.Manager{Root: t.TempDir()},
		Session: &session.Session{Playing: true, Characters: []session.Character{
			{ID: 150000, Slot: 0, Name: "Novice"},
		}},
		World: world, Input: input.NewState(),
		UIManager: &worldModeTestUIManager{}, ScreenW: 800, ScreenH: 600,
	}
}

func TestMissingMapReturnsToLoginWithoutGameAssets(t *testing.T) {
	for _, kind := range []string{"missing", "malformed", "no resources", "wide map name"} {
		t.Run(kind, func(t *testing.T) {
			ctx := mapRecoveryTestContext(t)
			if kind == "malformed" {
				if err := os.WriteFile(filepath.Join(ctx.Resources.Root, "new_1-1.gat"), []byte("broken"), 0600); err != nil {
					t.Fatal(err)
				}
			} else if kind == "no resources" {
				ctx.Resources = nil
			} else if kind == "wide map name" {
				ctx.World.MapName = strings.Repeat("W", 12) + ".gat"
			}
			mapName := ctx.World.MapName
			ctx.World.GAT = &res.GAT{Width: 100, Height: 100}
			ctx.World.GND = &res.GND{}
			ctx.World.RSW = &res.RSW{}
			ui := ctx.UIManager.(*worldModeTestUIManager)
			ui.AddOverlay(primitives.Box())
			manager := NewManager(ctx, NewWorldMode())
			mode, ok := manager.mode.(*LoginMode)
			if !ok || mode.phase != loginPhaseAccount || !mode.disconnectDialog.IsOpen() {
				t.Fatalf("failed map did not open an alert on login: %T", manager.mode)
			}
			messages := mode.console.Messages()
			if len(messages) == 0 || !strings.Contains(messages[len(messages)-1].Text, mapName) {
				t.Fatal("map error details were not retained in console history")
			}
			if kind == "malformed" && !strings.Contains(messages[len(messages)-1].Text, "gat too short") {
				t.Fatal("the GAT parse error was lost")
			}
			if ctx.World.GAT != nil || ctx.World.GND != nil || ctx.World.RSW != nil || ctx.Session.Playing {
				t.Fatal("failed map retained the previous world")
			}
			if len(ui.overlays) != 1 || mode.fade.phase != loginFadeNone {
				t.Fatal("old windows or loading cover obscured the error")
			}
			ctx.Resources, ctx.World = nil, nil
			ctx.UIApp = fakeCursorUIApp{cursor: widget.CursorPointer}
			manager.UpdateContext(ctx)
			frame := render.NewFrame(ctx.ScreenW, ctx.ScreenH)
			manager.Draw(frame)
			manager.DrawUIOverlay(frame)
			manager.DrawOverlay(frame)
			if mode.cursor.fallback == nil || mode.cursor.action != cursorActionClick {
				t.Fatal("error dialog did not inherit the login cursor")
			}
			canvas := drawMapErrorWindow(mode)
			for _, text := range []string{"Map unavailable", mapName, "Check your game data.", "Please log in again.", "OK"} {
				uitest.AssertDrawnText(t, canvas, text)
			}
			bounds := mode.disconnectDialog.Widget().(interface{ Bounds() geometry.Rect }).Bounds()
			for _, line := range canvas.StyledTexts {
				width := canvas.MeasureStyledText(line.Text, line.Style)
				if line.Bounds.Min.X < bounds.Min.X || line.Bounds.Min.X+width > bounds.Max.X || line.Bounds.Max.Y > bounds.Max.Y {
					t.Fatalf("text %q overflows error dialog", line.Text)
				}
			}
		})
	}
}

func TestMissingMapClosesConnectionAndAllowsExplicitLogin(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	// A configured login server makes unwanted automatic reconnects observable.
	ctx.Resources.ClientInfo.Connections = []res.Connection{{Address: "127.0.0.1", Port: 1}}
	ctx.Config.Login.AutoLogin = true
	ctx.Config.Login.CharSlot = 0
	ctx.Input.SetKey(input.KeyEnter, true)
	manager := NewManager(ctx, NewWorldMode())
	mode := manager.mode.(*LoginMode)
	if ctx.Network.Status() != "offline" {
		t.Fatal("failed map connection remained active while showing the error")
	}
	if err := server.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if n, err := server.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Fatalf("failed map sent a packet or kept the server connected: n=%d, err=%v", n, err)
	}
	for range 2 {
		if err := manager.Update(); err != nil {
			t.Fatal(err)
		}
		if !mode.disconnectDialog.IsOpen() || mode.loginPending {
			t.Fatal("opening/held Enter dismissed the error or reconnected")
		}
		ctx.Input.EndFrame()
	}
	ctx.Input.SetKey(input.KeyEnter, false)
	ctx.Input.SetKey(input.KeyEnter, true)
	if err := manager.Update(); err != nil {
		t.Fatal(err)
	}
	if mode.disconnectDialog.IsOpen() || mode.loginWindow == nil || mode.loginPending || !mode.autoAttempted || !mode.autoCharAttempted {
		t.Fatal("acknowledging the error did not leave login ready for an explicit retry")
	}
}

func TestMissingMapWarpRedirectsToLoginBeforeDrawing(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	ctx.World.MapName = "prontera.gat"
	ctx.Network = network.NewClient(20080910, false)
	t.Cleanup(ctx.Network.Close)
	writeTestGAT(t, ctx.Resources.Root, ctx.World.MapName)
	manager := NewManager(ctx, NewWorldMode())
	world, ok := manager.mode.(*WorldMode)
	if !ok {
		t.Fatalf("valid map did not enter world mode: %T", manager.mode)
	}
	world.startMapFadeOut(network.MapChange{MapName: "new_1-1.gat", X: 1, Y: 1}, time.Now().Add(-mapFadeOutDuration))
	if err := manager.Update(); err != nil {
		t.Fatal(err)
	}
	manager.FrameSubmitted()
	if err := manager.Update(); err != nil {
		t.Fatal(err)
	}
	login, ok := manager.mode.(*LoginMode)
	if !ok || login.phase != loginPhaseAccount || !login.disconnectDialog.IsOpen() || ctx.World.GAT != nil {
		t.Fatalf("failed warp did not redirect to recovery before drawing: %T", manager.mode)
	}
	// All rendering now belongs to login and must not consult the world.
	ctx.World = nil
	manager.UpdateContext(ctx)
	frame := render.NewFrame(ctx.ScreenW, ctx.ScreenH)
	manager.Draw(frame)
	manager.DrawUIOverlay(frame)
	manager.DrawOverlay(frame)
	if login.cursor.fallback == nil {
		t.Fatal("recovery did not inherit the login cursor")
	}
}

func TestLoadGATPreservesReadErrors(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	path := filepath.Join(ctx.Resources.Root, "new_1-1.gat")
	if err := os.WriteFile(path, []byte("unreadable"), 0000); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(path); err == nil {
		t.Skip("current user can read files without read permission")
	}
	_, _, err := loadGAT(ctx.Resources, ctx.World.MapName)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("loader replaced the read error: %v", err)
	}
}

func TestWorldEnterWithGATStillAcknowledgesSuccessfulLoad(t *testing.T) {
	ctx := mapRecoveryTestContext(t)
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	writeTestGAT(t, ctx.Resources.Root, "new_1-1.gat")
	mode := NewWorldMode()
	if next := mode.Enter(ctx); next != nil || ctx.World.GAT == nil {
		t.Fatal("a valid GAT with optional rendering assets missing triggered recovery")
	}
	readBotTestPackets(t, server, network.BuildLoadEndAckPacket())
}

func writeTestGAT(t *testing.T, root, name string) {
	t.Helper()
	data := make([]byte, 34)
	copy(data, "GRAT")
	data[4], data[5] = 1, 2
	binary.LittleEndian.PutUint32(data[6:10], 1)
	binary.LittleEndian.PutUint32(data[10:14], 1)
	if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}

// MockCanvas records local text bounds; capture screen bounds for real button
// input, including transforms introduced by the window and its nested boxes.
type mapRecoveryCanvas struct{ uitest.MockCanvas }

func (c *mapRecoveryCanvas) DrawStyledText(text string, bounds geometry.Rect, style widget.TextStyle) {
	offset := c.TransformOffset()
	c.MockCanvas.DrawStyledText(text, bounds.TranslateXY(offset.X, offset.Y), style)
}

func drawMapErrorWindow(mode *LoginMode) *uitest.MockCanvas {
	root := mode.disconnectDialog.Widget()
	wc := widget.NewContext()
	root.Layout(wc, geometry.Tight(geometry.Sz(800, 600)))
	canvas := &mapRecoveryCanvas{}
	root.Draw(wc, canvas)
	for _, text := range canvas.StyledTexts {
		canvas.Texts = append(canvas.Texts, uitest.DrawTextCall{Text: text.Text, Bounds: text.Bounds})
	}
	return &canvas.MockCanvas
}
