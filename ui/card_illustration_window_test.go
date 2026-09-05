package ui

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"golang.org/x/image/bmp"
)

func TestCardIllustrationWindowLoadsMappedArtwork(t *testing.T) {
	manager := cardIllustrationTestManager(t)
	ctx := Context{Resources: manager, ScreenW: 800, ScreenH: 600}
	var window CardIllustrationWindow

	if err := window.Open(ctx, 4001, "Poring Card"); err != nil {
		t.Fatal(err)
	}
	if !window.IsOpen() {
		t.Fatal("card illustration window did not open")
	}
	if window.title != "Poring Card" {
		t.Fatalf("card illustration title = %q", window.title)
	}
	if window.width != cardIllustrationWidth || window.height != cardIllustrationWindowHeight {
		t.Fatalf("card illustration window size = %dx%d", window.width, window.height)
	}
	if window.image == nil || window.image.Bounds().Dx() != cardIllustrationWidth || window.image.Bounds().Dy() != cardIllustrationHeight {
		t.Fatalf("card illustration image = %v", window.image)
	}
}

func TestItemInfoCardViewButtonRequestsIllustration(t *testing.T) {
	resources := cardIllustrationTestManager(t)
	app := uiapp.New()
	manager := NewManager()
	manager.SetUIApp(basicMenuTestApp{app: app})
	inputState := input.NewState()
	ctx := client.Context{
		Input:     inputState,
		Resources: resources,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}
	var window ItemInfoWindow
	window.openItem(ctx, session.InventoryItem{ItemID: 4001, Type: db.ItemTypeCard, Identified: true}, 100, 100)

	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	buttonX := float32(window.x + ROWindowFooterPadding + 20)
	buttonY := float32(window.y + window.height - ROWindowFooterHeight/2)
	app.Window().HandleEvent(uitest.Click(buttonX, buttonY))
	app.Window().HandleEvent(uitest.Release(buttonX, buttonY))

	request := window.PopCardIllustrationRequest()
	if request.ItemID != 4001 || request.Title != window.title {
		t.Fatalf("card illustration request = %+v", request)
	}
}

func TestItemInfoCardViewRequest(t *testing.T) {
	manager := cardIllustrationTestManager(t)
	ctx := Context{Resources: manager}
	window := ItemInfoWindow{
		item:  session.InventoryItem{ItemID: 4001, Type: db.ItemTypeCard, Identified: true},
		title: "Poring Card",
	}
	if !itemInfoShowsCardIllustration(ctx, window.item) {
		t.Fatal("mapped card does not expose its illustration action")
	}

	window.requestCardIllustration()
	request := window.PopCardIllustrationRequest()
	if request.ItemID != 4001 || request.Title != "Poring Card" {
		t.Fatalf("card illustration request = %+v", request)
	}
	if request := window.PopCardIllustrationRequest(); request.ItemID != 0 {
		t.Fatalf("card illustration request was not cleared: %+v", request)
	}
}

func cardIllustrationTestManager(t *testing.T) *res.Manager {
	t.Helper()
	root := t.TempDir()
	tablePath := filepath.Join(root, "data", "num2cardillustnametable.txt")
	if err := os.MkdirAll(filepath.Dir(tablePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tablePath, []byte("4001#poring_card#\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	imagePath := filepath.Join(root, "data", "texture", "유저인터페이스", "cardbmp", "poring_card.bmp")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := bmp.Encode(file, image.NewNRGBA(image.Rect(0, 0, cardIllustrationWidth, cardIllustrationHeight))); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
