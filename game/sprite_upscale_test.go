package game

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/kivutar/goro/render"
)

func TestSpriteUpscalingKeepsStraightAlpha(t *testing.T) {
	source := render.NewImage(3, 2)
	want := color.RGBA{R: 210, G: 120, B: 60, A: 80}
	for y := range 2 {
		for x := range 3 {
			source.RGBA().SetRGBA(x, y, want)
		}
	}
	before := bytes.Clone(source.RGBA().Pix)
	upscaled := upscaleSpriteImage(source)
	if upscaled.Bounds() != image.Rect(0, 0, 6, 4) {
		t.Fatalf("upscaled bounds = %v", upscaled.Bounds())
	}
	for y := range 4 {
		for x := range 6 {
			if got := upscaled.RGBA().RGBAAt(x, y); got != want {
				t.Fatalf("translucent pixel = %v, want %v", got, want)
			}
		}
	}
	if !bytes.Equal(before, source.RGBA().Pix) {
		t.Fatal("upscaling modified the shared source")
	}
}

func TestSpriteUpscalingCachesAndReleasesOnMapChange(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &WorldMode{}
		defer m.Leave()
		source := render.NewImage(13, 9)
		if m.spriteTexture(source) != source || m.spriteUpscaler != nil {
			t.Fatal("disabled upscaling started work")
		}
		m.textureUpscaling = true
		if m.spriteTexture(source) != source {
			t.Fatal("first draw must use the original immediately")
		}
		synctest.Wait()
		upscaled := m.spriteTexture(source)
		if upscaled == source || upscaled.Bounds() != image.Rect(0, 0, 26, 18) {
			t.Fatal("background result was not adopted")
		}
		if m.spriteTexture(source) != upscaled || len(m.spriteUpscaler.cache) != 1 {
			t.Fatal("repeated draws did not reuse the same texture")
		}
		m.Leave()
		synctest.Wait()
		if m.spriteUpscaler != nil || !upscaled.Bounds().Empty() || source.Bounds().Empty() {
			t.Fatal("map exit must release upscaled images without changing the originals")
		}
		// A new map can reuse an original image without reusing a released result.
		m.spriteTexture(source)
		synctest.Wait()
		if next := m.spriteTexture(source); next == upscaled || next.Bounds().Empty() {
			t.Fatal("new map reused a released texture")
		}
	})
}

func TestManagerCloseReleasesSpriteUpscaling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		world := &WorldMode{textureUpscaling: true}
		defer world.Leave()
		source := render.NewImage(16, 16)
		world.spriteTexture(source)
		synctest.Wait()
		upscaled := world.spriteTexture(source)
		if upscaled == source {
			t.Fatal("upscaled frame was not ready")
		}
		worker := world.spriteUpscaler
		manager := &Manager{mode: world}
		manager.Close() // Game.Close uses this path when Android recreates a surface.
		manager.Close() // Closing twice must not close the worker's channel twice.
		synctest.Wait()
		select {
		case <-worker.stop:
		default:
			t.Fatal("closing the manager left the sprite worker running")
		}
		if !upscaled.Bounds().Empty() || source.Bounds().Empty() {
			t.Fatal("closing must release upscaled pixels without modifying the source")
		}
		if manager.mode != nil || world.spriteUpscaler != nil || world.mapImages != nil {
			t.Fatal("closing retained the mode or its texture cache")
		}
	})
}

func TestSpriteUpscalingQueueDoesNotBlockAndStopsOnExit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// Hold the worker until the queue is full, then leave the map.
		s := &spriteUpscaler{
			queue: make(chan *spriteUpscaleJob, 1),
			stop:  make(chan struct{}),
			cache: make(map[*render.Image]*spriteUpscaleJob),
		}
		m := &WorldMode{textureUpscaling: true, spriteUpscaler: s}
		first, second := render.NewImage(4, 4), render.NewImage(5, 5)
		if m.spriteTexture(first) != first || m.spriteTexture(second) != second || len(s.cache) != 1 {
			t.Fatal("busy worker must leave the original visible without growing the queue")
		}
		job := <-s.queue
		if m.spriteTexture(second) != second || len(s.cache) != 2 {
			t.Fatal("visible sprite was not retried after the queue had room")
		}
		m.Leave()
		exited := false
		go func() { s.run(); exited = true }()
		synctest.Wait()
		if !exited || job.ready.Load() != nil || s.cache[second].ready.Load() != nil {
			t.Fatal("worker processed queued sprites after leaving the map")
		}
	})
}

func TestSpriteUpscalingChangesTextureWithoutMovingBillboard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &WorldMode{textureUpscaling: true}
		defer m.Leave()
		base := &spriteBillboard{image: render.NewImage(31, 23), anchorX: 12.5, anchorY: 19}
		billboard := cartOffsetBillboard(base, 7, -3)
		projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 3)
		draw := func() reflect.Value {
			frame := render.NewFrame(800, 600)
			m.drawActorSpriteBillboardTintAlpha3D(frame, projection, billboard, 10.5, 20.5, 3, 1.7, 0.6, 0.8, color.RGBA{R: 90, G: 160, B: 255, A: 200}, render.BlendSourceOver)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != 1 {
				t.Fatal("missing billboard command")
			}
			return commands.Index(0)
		}
		original := draw()
		synctest.Wait()
		upscaled := draw()
		if original.FieldByName("Texture").Pointer() == upscaled.FieldByName("Texture").Pointer() {
			t.Fatal("render command did not use the upscaled texture")
		}
		for _, name := range []string{"Width", "Height", "AnchorX", "AnchorY", "ColorR", "ColorG", "ColorB", "ColorA", "DepthBias"} {
			if original.FieldByName(name).Float() != upscaled.FieldByName(name).Float() {
				t.Errorf("upscaling changed %s", name)
			}
		}
		for _, name := range []string{"Center", "RightAxis", "UpAxis", "DepthUpAxis"} {
			for i := range 3 {
				if original.FieldByName(name).Index(i).Float() != upscaled.FieldByName(name).Index(i).Float() {
					t.Errorf("upscaling changed %s[%d]", name, i)
				}
			}
		}
		if got := m.spriteTexture(base.image); got != m.spriteTexture(billboard.image) || len(m.spriteUpscaler.cache) != 1 {
			t.Fatal("anchor offsets duplicated the processed texture")
		}
	})
}
