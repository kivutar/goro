package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func mapTextureFixture(t *testing.T) (*WorldMode, client.Context) {
	t.Helper()
	root := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, B: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	names := []string{"ground.png", "model.png"}
	for i := 0; i < 32; i++ {
		names = append(names, fmt.Sprintf("water700%02d.jpg", i))
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(root, name), encoded.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	w := worldstate.New()
	w.GND = &res.GND{Width: 1, Height: 1, Textures: []string{"ground.png", "ground.png", "missing.png"}, Cells: []res.GNDCell{{Top: 0}}, Lightmaps: []res.GNDLightmap{{}}}
	w.RSW = &res.RSW{Water: res.RSWWater{Level: 1, Type: 700}, Models: []res.RSWModel{{Filename: "test.rsm"}, {Filename: "test.rsm"}}}
	w.RSM = map[string]*res.RSM{"test.rsm": {Textures: []string{"ground.png", "model.png", "missing.png"}}}
	m := &WorldMode{textures: make(map[string]*render.Image), textureMiss: make(map[string]struct{})}
	return m, client.Context{Resources: &res.Manager{Root: root}, World: w}
}

func TestMapTexturesPreloadAllSourcesAndReuseDrawCaches(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	m.preloadMapTextures(ctx)
	if len(m.mapTextureUploads) != 35 || len(m.textures) != 34 || len(m.textureMiss) != 1 {
		t.Fatalf("uploads=%d textures=%d missing=%d; want 2 shared/model + 32 water + lightmap", len(m.mapTextureUploads), len(m.textures), len(m.textureMiss))
	}
	// Even if the file becomes unreadable, normal rendering uses preparation.
	if err := os.WriteFile(filepath.Join(ctx.Resources.Root, "ground.png"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	texture := m.groundTexture(ctx.Resources, "ground.png")
	if texture != m.mapTextureUploads[0] || texture.RGBA().RGBAAt(0, 0) != (color.RGBA{}) || texture.RGBA().RGBAAt(1, 0) != (color.RGBA{G: 255, A: 255}) {
		t.Fatal("normal loader did not reuse preloaded pixels or transparency changed")
	}
	for frame := 0; frame < 32; frame++ {
		if img := m.waterTexture(ctx.Resources, 700, frame); img == nil {
			t.Fatalf("water frame %d missing", frame)
		}
	}
	if atlas := m.gndMeshCache.lightmapAtlas.image; atlas == nil || atlas != m.mapTextureUploads[1] {
		t.Fatal("lightmap preparation was not retained")
	}
}

func TestMapTexturesSkipDryMapsAndHeadless(t *testing.T) {
	for _, headless := range []bool{false, true} {
		m, ctx := mapTextureFixture(t)
		ctx.Config.Headless = headless
		ctx.World.RSW.Water.Level = -1
		m.preloadMapTextures(ctx)
		want := 3
		if headless {
			want = 0
		}
		if len(m.mapTextureUploads) != want {
			t.Fatalf("headless=%t: prepared %d textures, want %d", headless, len(m.mapTextureUploads), want)
		}
		// A dry map must not start lazy water loads during subsequent draws.
		// A nil resource manager makes any attempted file read fail this test.
		m.drawGNDWater(render.NewFrame(100, 100), nil, ctx.World.GND, ctx.World.RSW, sceneProjection{}, time.Now(), sceneFog{})
	}
}

func TestMapTextureUploadsWaitForSubmissionBeforeFade(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	m.preloadMapTextures(ctx)
	m.startMapPrewarm()
	f := render.NewFrame(100, 100)
	now := time.Now()
	for retry := 0; retry < 2; retry++ {
		f.BeginFrame()
		if !m.prepareMapTextureUploads(f) || m.mapUploadBatch != mapTextureUploadCount {
			t.Fatal("did not request the bounded first batch")
		}
		m.advanceMapPrewarm(now)
		if m.mapFade.phase != mapFadePrewarm || len(m.mapTextureUploads) != 35 {
			t.Fatal("skipped submission advanced loading")
		}
	}
	m.FrameSubmitted()
	if len(m.mapTextureUploads) != 3 || m.mapFade.coveredFrames != 0 {
		t.Fatal("upload counted as a rendered scene frame")
	}
	f.BeginFrame()
	m.prepareMapTextureUploads(f)
	m.FrameSubmitted()
	if len(m.mapTextureUploads) != 0 || m.mapFade.coveredFrames != 0 {
		t.Fatal("last upload did not finish before scene warmup")
	}
	for i := 0; i < mapFadePrewarmFrames; i++ {
		m.advanceMapPrewarm(now)
		if m.mapFade.phase != mapFadePrewarm {
			t.Fatal("fade started before scene warmup finished")
		}
		m.FrameSubmitted()
	}
	m.advanceMapPrewarm(now)
	if m.mapFade.phase != mapFadeIn {
		t.Fatal("fade did not start after all uploads and scene frames")
	}
}

func TestMapTextureUploadsRespectBytesAndAllowOneOversizedImage(t *testing.T) {
	m := &WorldMode{mapTextureUploads: []*render.Image{render.NewImage(1024, 1024), render.NewImage(1024, 1024), render.NewImage(2048, 2048)}}
	f := render.NewFrame(100, 100)
	m.prepareMapTextureUploads(f)
	if m.mapUploadBatch != 2 {
		t.Fatal("upload byte budget exceeded")
	}
	m.FrameSubmitted()
	f.BeginFrame()
	m.prepareMapTextureUploads(f)
	if m.mapUploadBatch != 1 {
		t.Fatal("single large texture cannot make progress")
	}
}

func TestLeavingWorldReleasesMapTextures(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	m.preloadMapTextures(ctx)
	old := append([]*render.Image(nil), m.mapTextureUploads...)
	manager := &Manager{ctx: ctx, mode: m}
	manager.enter(&mapTextureExitMode{})
	for _, img := range old {
		if img.RGBA() != nil {
			t.Fatal("mode change kept old map pixels alive")
		}
	}
	if m.mapImages != nil || len(m.mapTextureUploads) != 0 {
		t.Fatal("mode change did not cancel pending uploads")
	}
}

type mapTextureExitMode struct{}

func (*mapTextureExitMode) Name() string                        { return "test" }
func (*mapTextureExitMode) Enter(client.Context) Mode           { return nil }
func (*mapTextureExitMode) Update(client.Context) (Mode, error) { return nil, nil }
func (*mapTextureExitMode) Draw(client.Context, *render.Frame)  {}

func TestLoadMapModelsHasNo128ModelLimitAndCachesFailures(t *testing.T) {
	root := t.TempDir()
	// Valid RSM 1.5 with no nodes or textures.
	data := make([]byte, 83)
	copy(data, []byte{'G', 'R', 'S', 'M', 1, 5})
	rsw := &res.RSW{}
	for i := 0; i < 130; i++ {
		name := fmt.Sprintf("model%d.rsm", i)
		if err := os.WriteFile(filepath.Join(root, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
		rsw.Models = append(rsw.Models, res.RSWModel{Filename: name})
	}
	rsw.Models = append(rsw.Models, res.RSWModel{Filename: "missing.rsm"}, res.RSWModel{Filename: "missing.rsm"})
	models, failures := loadRSMModels(&res.Manager{Root: root}, rsw)
	if len(models) != 131 || models["model129.rsm"] == nil || failures != 1 {
		t.Fatalf("models=%d failures=%d; late model=%v", len(models), failures, models["model129.rsm"])
	}
}
