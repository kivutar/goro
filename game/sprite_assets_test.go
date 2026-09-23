package game

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestSharedSpriteResourcesKeepViewStateAndPalettesIndependent(t *testing.T) {
	root := t.TempDir()
	act := make([]byte, 16)
	copy(act, []byte{'A', 'C', 1, 1})
	spr := append([]byte{'S', 'P', 1, 1, 1, 0, 2, 0, 1, 0, 1, 0}, make([]byte, 1024)...)
	spr[12+4] = 255 // Default palette: red.
	pal := make([]byte, 1024)
	pal[5] = 255 // Override palette: green.
	for name, data := range map[string][]byte{"body.act": act, "body.spr": spr, "green.pal": pal} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manager := &res.Manager{Root: root}
	load := func(palettes []string) *spriteView {
		t.Helper()
		view, status := loadSpriteView(manager, []string{"body.act"}, []string{"body.spr"}, palettes, "test")
		if view == nil {
			t.Fatal(status)
		}
		return view
	}
	first := load(nil)
	first.started = time.Unix(100, 0)
	first.billboards[singleSpriteBillboardKey{}] = &spriteBillboard{}
	first.images[spriteFrameKey{index: 100}] = render.NewImage(1, 1)
	second := load([]string{"green.pal"})
	if first == second || first.spr != second.spr || first.act != second.act {
		t.Fatal("views did not share just their parsed resources")
	}
	if first.started == second.started || len(second.billboards) != 0 || len(second.images) != 0 {
		t.Fatal("animation or rendering state leaked between views")
	}
	for view, want := range map[*spriteView]color.RGBA{
		first: {R: 255, A: 255}, second: {G: 255, A: 255},
	} {
		img, ok := spriteViewImage(view, 0, res.SPRFramePalette)
		if !ok || img.RGBA().RGBAAt(0, 0) != want || img.RGBA().RGBAAt(1, 0) != (color.RGBA{}) {
			t.Fatalf("wrong color or transparency for palette %q", view.paletteSource)
		}
	}
	// A fresh view, as created after a map change, still gets the same CPU
	// resources and default palette, without retaining the old view's images.
	third := load(nil)
	if third.spr != first.spr || third.act != first.act || third.palette != nil || len(third.images) != 0 {
		t.Fatal("fresh view did not reuse the unmodified resources")
	}
	if fourth := load([]string{"green.pal"}); fourth.palette != second.palette {
		t.Fatal("palette override not shared")
	}
}

func TestHumanoidViewsShareResourcesRealData(t *testing.T) {
	manager := realDataManager(t)
	first, status := loadHumanoidSpriteView(manager, 4, 1, 1, 0, 0, "first")
	if first == nil {
		t.Fatal(status)
	}
	second, status := loadHumanoidSpriteView(manager, 4, 2, 1, 0, 0, "second")
	if second == nil {
		t.Fatal(status)
	}
	if first.body == second.body || first.body.spr != second.body.spr || first.body.act != second.body.act || first.imf != second.imf {
		t.Fatal("different appearances did not share their common resources")
	}
	if first.head == nil || second.head == nil || first.head.spr == second.head.spr {
		t.Fatal("different heads lost their own resources")
	}
	for _, view := range []*humanoidSpriteView{first, second} {
		if _, ok := humanoidBillboardForState(view, spriteState{actionFamily: spriteActionIdle, direction: 0}, time.Now()); !ok {
			t.Fatal("shared resources could not render")
		}
	}
}

func TestMonsterViewsShareUpgradedResourcesRealData(t *testing.T) {
	manager := realDataManager(t)
	first, status := loadNonPCSpriteView(manager, 1002, "poring")
	if first == nil || len(first.act.Actions) < 40 {
		t.Fatalf("richer monster resource missing: %s", status)
	}
	second, status := loadNonPCSpriteView(manager, 1002, "poring again")
	if second == nil || first == second || first.act != second.act || first.spr != second.spr {
		t.Fatalf("upgraded monster resources not shared: %s", status)
	}
}
