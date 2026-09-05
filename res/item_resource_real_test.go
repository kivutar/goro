package res

import (
	"image"
	"testing"
)

func TestItemResourceRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	name, ok := manager.ItemDisplayName(909, true)
	if !ok || name == "" {
		t.Fatalf("item 909 display name missing: %q ok=%v", name, ok)
	}
	resource, ok := manager.ItemResourceName(909, true)
	if !ok || resource == "" {
		t.Fatalf("item 909 resource missing: %q ok=%v", resource, ok)
	}
	actSource, actData, ok := manager.ReadFirst(ItemSpriteResourceCandidates(resource, "act"))
	if !ok {
		t.Fatalf("item 909 act %q missing", resource)
	}
	if _, err := ParseACT(actData); err != nil {
		t.Fatalf("item 909 act %s parse failed: %v", actSource, err)
	}
	sprSource, sprData, ok := manager.ReadFirst(ItemSpriteResourceCandidates(resource, "spr"))
	if !ok {
		t.Fatalf("item 909 spr %q missing", resource)
	}
	if _, err := ParseSPR(sprData); err != nil {
		t.Fatalf("item 909 spr %s parse failed: %v", sprSource, err)
	}
	t.Logf("item 909 name=%q resource=%q act=%s spr=%s", name, resource, actSource, sprSource)
}

func TestAppleItemSpriteRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	name, ok := manager.ItemDisplayName(512, true)
	if !ok || name != "Apple" {
		t.Fatalf("item 512 display name = %q ok=%v, want Apple", name, ok)
	}
	resource, ok := manager.ItemResourceName(512, true)
	if !ok || resource == "" {
		t.Fatalf("item 512 resource missing: %q ok=%v", resource, ok)
	}
	actSource, actData, ok := manager.ReadFirst(ItemSpriteResourceCandidates(resource, "act"))
	if !ok {
		t.Fatalf("item 512 act %q missing", resource)
	}
	act, err := ParseACT(actData)
	if err != nil {
		t.Fatalf("item 512 act %s parse failed: %v", actSource, err)
	}
	sprSource, sprData, ok := manager.ReadFirst(ItemSpriteResourceCandidates(resource, "spr"))
	if !ok {
		t.Fatalf("item 512 spr %q missing", resource)
	}
	spr, err := ParseSPR(sprData)
	if err != nil {
		t.Fatalf("item 512 spr %s parse failed: %v", sprSource, err)
	}
	if len(act.Actions) == 0 || len(act.Actions[0].Animations) == 0 || len(act.Actions[0].Animations[0].Layers) == 0 {
		t.Fatalf("item 512 act %s has no drawable first frame", actSource)
	}
	layer := act.Actions[0].Animations[0].Layers[len(act.Actions[0].Animations[0].Layers)-1]
	img, ok := spr.FrameImage(int(layer.Index), int(layer.SPRType))
	if !ok {
		t.Fatalf("item 512 first visible frame image missing layer=%+v", layer)
	}
	red, green, blue := dominantImageRGB(img)
	t.Logf("item 512 name=%q resource=%q act=%s spr=%s dominant=%d,%d,%d", name, resource, actSource, sprSource, red, green, blue)
	if red <= green || red <= blue {
		t.Fatalf("item 512 dominant color = %d,%d,%d, want red-dominant", red, green, blue)
	}
}

func TestCardIllustrationRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	resource, ok := manager.ItemCardIllustrationName(4001)
	if !ok || resource == "" {
		t.Fatalf("card 4001 illustration resource missing: %q ok=%v", resource, ok)
	}
	img, source, err := LoadImage(manager, CardIllustrationTextureCandidates(resource))
	if err != nil {
		t.Fatalf("card 4001 illustration %q missing: %v", resource, err)
	}
	if got := img.Bounds().Size(); got.X != 300 || got.Y != 400 {
		t.Fatalf("card 4001 illustration %s size = %v, want 300x400", source, got)
	}
	t.Logf("card 4001 illustration=%q source=%s", resource, source)
}

func dominantImageRGB(img *image.NRGBA) (int, int, int) {
	var red, green, blue, count int
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			red += int(c.R)
			green += int(c.G)
			blue += int(c.B)
			count++
		}
	}
	if count == 0 {
		return 0, 0, 0
	}
	return red / count, green / count, blue / count
}
