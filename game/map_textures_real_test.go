package game

import (
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	worldstate "github.com/kivutar/goro/world"
)

func TestMapTexturePreparationRealData(t *testing.T) {
	resources := realDataManager(t)
	for _, name := range []string{"prontera", "payon", "payon_in01", "yuno", "airplane", "geffen", "prt_fild08"} {
		t.Run(name, func(t *testing.T) {
			start := time.Now()
			w := worldstate.New()
			var err error
			w.GND, _, err = loadGND(resources, name)
			if err != nil {
				t.Skipf("map unavailable: %v", err)
			}
			w.RSW, _, err = loadRSW(resources, name)
			if err != nil {
				t.Fatal(err)
			}
			w.RSM, w.RSMFail = loadRSMModels(resources, w.RSW)
			m := &WorldMode{textures: make(map[string]*render.Image), textureMiss: make(map[string]struct{})}
			m.preloadMapTextures(client.Context{Resources: resources, World: w})
			checkTexture := func(name string) {
				t.Helper()
				_, missing := m.textureMiss[name]
				if name != "" && m.textures[name] == nil && !missing {
					t.Fatalf("texture %q would load during gameplay", name)
				}
			}
			for _, name := range w.GND.Textures {
				checkTexture(name)
			}
			for _, placement := range w.RSW.Models {
				model, attempted := w.RSM[placement.Filename]
				if placement.Filename != "" && !attempted {
					t.Fatalf("model %q was not prepared", placement.Filename)
				}
				if model != nil {
					for _, name := range model.Textures {
						checkTexture(name)
					}
				}
			}
			bytes := 0
			images := append([]*render.Image(nil), m.mapTextureUploads...)
			for _, img := range images {
				bytes += len(img.RGBA().Pix)
			}
			t.Logf("models=%d failed_models=%d textures=%d missing=%d pixels=%.2f MiB prepare=%s", len(w.RSM), w.RSMFail, len(images), len(m.textureMiss), float64(bytes)/(1<<20), time.Since(start))
			m.Leave()
			for _, img := range images {
				if img.RGBA() != nil {
					t.Fatal("map left CPU pixels allocated")
				}
			}
		})
	}
}
