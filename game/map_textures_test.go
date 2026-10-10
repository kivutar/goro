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
	for _, name := range []string{"ground.png", "model.png"} {
		if img := m.textures[name]; img.ByteSize() != 12 {
			t.Fatalf("%s: missing 1x1 mip below the 2x1 base", name)
		}
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
		if img := m.waterTexture(ctx.Resources, 700, frame); img == nil || img.ByteSize() != 8 {
			t.Fatalf("water frame %d missing or mipmapped", frame)
		}
	}
	if atlas := m.gndMeshCache.lightmapAtlas.image; atlas == nil || atlas != m.mapTextureUploads[1] || atlas.ByteSize() != len(atlas.RGBA().Pix) {
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

type mapTextureRuntime struct {
	client.RuntimeSettings
	upscale bool
}

func (r *mapTextureRuntime) TextureUpscaling() bool { return r.upscale }

func TestMapTextureUpscalingIsCachedAndAppliesOnNextMap(t *testing.T) {
	m, ctx := mapTextureFixture(t)
	runtime := &mapTextureRuntime{upscale: true}
	ctx.Runtime = runtime
	m.preloadMapTextures(ctx)
	for _, name := range []string{"ground.png", "model.png"} {
		img := m.textures[name]
		if img.Bounds() != image.Rect(0, 0, 4, 2) || img.ByteSize() != 44 {
			t.Fatalf("%s was not upscaled with mipmaps", name)
		}
		if m.groundTexture(ctx.Resources, name) != img {
			t.Fatal("draw-time access did not reuse the upscaled texture")
		}
	}
	if m.waterTexture(ctx.Resources, 700, 0).ByteSize() != 8 {
		t.Fatal("upscaling changed a water texture")
	}
	if atlas := m.gndMeshCache.lightmapAtlas.image; atlas.ByteSize() != len(atlas.RGBA().Pix) {
		t.Fatal("upscaling changed the lightmap")
	}
	runtime.upscale = false
	delete(m.textures, "model.png")
	if m.groundTexture(ctx.Resources, "model.png").Bounds().Dx() != 4 {
		t.Fatal("late model load ignored the current map's captured upscale setting")
	}
	m.Leave()
	m.textures = make(map[string]*render.Image)
	ctx.Config.Render.TextureUpscaling = true // The runtime setting must take precedence.
	m.preloadMapTextures(ctx)
	if m.textures["ground.png"].Bounds().Dx() != 2 {
		t.Fatal("next map did not pick up the disabled runtime setting")
	}
	m.Leave()
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

func TestMapTextureUploadBudgetIncludesMipmaps(t *testing.T) {
	texture := render.NewImageWithMipmaps(image.NewRGBA(image.Rect(0, 0, 1024, 1024)))
	m := &WorldMode{mapTextureUploads: []*render.Image{texture, texture}}
	m.prepareMapTextureUploads(render.NewFrame(100, 100))
	if m.mapUploadBatch != 1 {
		t.Fatal("two 4 MiB base levels fit the budget, but their mip chains do not")
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
