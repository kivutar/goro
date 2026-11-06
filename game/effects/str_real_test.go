package effects

import (
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/res"
)

func TestRandomSTRLifetimesUseTheRenderedVariant(t *testing.T) {
	ctx := testutil.KnightEffectContext()
	strs := map[string]*res.STR{
		"pong1": {FPS: 60, MaxKey: 69},
		"pong2": {FPS: 60, MaxKey: 100},
		"pong3": {FPS: 60, MaxKey: 90},
	}
	spec, _ := Lookup(Pong)
	for _, path := range []string{"actor", "cell", "cell lifetime"} {
		t.Run(path, func(t *testing.T) {
			m := &System{StrEffects: make(map[string]*res.STR)}
			for file, str := range strs {
				m.StrEffects[`__str_data\texture\effect\`+file+`.str|`] = str
			}
			seen := make(map[string]bool)
			for i := 0; i < 32; i++ {
				start := time.Unix(int64(10+i), 0)
				var added bool
				switch path {
				case "actor":
					added = m.AddAt(ctx, Pong, 300, start)
				case "cell":
					added = m.AddCell(ctx, Pong, 300, 30, 20, start, 0, 0)
				case "cell lifetime":
					added = m.AddCellLifetime(ctx, Pong, 300, 30, 20, start, 0, false)
				}
				if !added {
					t.Fatal("effect was not added")
				}
				effect := m.Active[len(m.Active)-1]
				file := ResolveEffectSTRFile(spec.Components[0], effect, false)
				seen[file] = true
				str := strs[file]
				want := time.Duration(float64(str.MaxKey)/float64(str.FPS)*float64(time.Second)) + 100*time.Millisecond
				if got := effect.Expires.Sub(start); got != want {
					t.Fatalf("%s lifetime = %s, want %s", file, got, want)
				}
				if got := m.WorldEffectResolvedComponentDuration(ctx, spec, spec.Components[0], effect); got != want {
					t.Fatalf("%s draw duration = %s, want %s", file, got, want)
				}
			}
			if len(seen) != len(strs) {
				t.Fatalf("did not exercise all variants: %v", seen)
			}
		})
	}
}

func TestSTRAnimationUnknownAniTypeKeepsSourceTextureFrame(t *testing.T) {
	layer := res.STRLayer{
		Textures: []string{"success.bmp", "failed.bmp"},
		Animations: []res.STRAnimation{
			{Frame: 55, Type: 0, AniFrame: 1},
			{Frame: 55, Type: 1, AniType: 0},
		},
	}

	anim, ok := calculateSTRAnimation(layer, 60)
	if !ok {
		t.Fatal("animation not visible")
	}
	if anim.AniFrame != 1 {
		t.Fatalf("animation frame = %.0f, want source frame 1", anim.AniFrame)
	}
}

func TestRealPharmacyFailedSTRSelectsFailedBanner(t *testing.T) {
	manager := testutil.RealDataManager(t)
	str := loadRealSTR(t, manager, `data\texture\effect\p_failed.str`)

	for _, key := range []float64{60, 80} {
		textures := activeSTRTextureBasenames(str, key)
		if !containsString(textures, "failed.bmp") {
			t.Fatalf("key %.0f textures = %v, want failed.bmp", key, textures)
		}
		if containsString(textures, "success.bmp") {
			t.Fatalf("key %.0f textures = %v, did not expect success.bmp in failed result banner", key, textures)
		}
	}
}

func loadRealSTR(t *testing.T, manager *res.Manager, path string) *res.STR {
	t.Helper()
	data, err := manager.ReadFileExact(path)
	if err != nil {
		t.Fatalf("read exact %s: %v", path, err)
	}
	str, err := res.ParseSTR(data, "")
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return str
}

func activeSTRTextureBasenames(str *res.STR, keyIndex float64) []string {
	var out []string
	for _, layer := range str.Layers {
		anim, ok := calculateSTRAnimation(layer, keyIndex)
		if !ok {
			continue
		}
		textureIndex := int(anim.AniFrame)
		if textureIndex < 0 || textureIndex >= len(layer.Textures) {
			continue
		}
		out = append(out, basenameBackslash(layer.Textures[textureIndex]))
	}
	return out
}

func basenameBackslash(path string) string {
	index := strings.LastIndexAny(path, `\/`)
	if index < 0 {
		return path
	}
	return path[index+1:]
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
