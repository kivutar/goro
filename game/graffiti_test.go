package game

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"testing"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

func TestGraffitiBitmapAlphabet(t *testing.T) {
	atlas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			atlas.SetRGBA(x, y, color.RGBA{R: uint8(y/32*8 + x/32), A: 255})
		}
	}
	texture := graffitiTexture(atlas, "Az9?!")
	y := 5 * 36 // native vertical placement for a five-character message
	for _, tc := range []struct{ x, glyph int }{{0, 0}, {32, 51}, {64, 63}, {96, 53}, {112, 54}} {
		if got := texture.RGBA().RGBAAt(tc.x, y); got.R != uint8(tc.glyph) || got.A != 255 {
			t.Fatalf("glyph at x=%d = %+v, want %d", tc.x, got, tc.glyph)
		}
	}
	if texture.RGBA().RGBAAt(128, y).A != 0 || texture.RGBA().RGBAAt(0, 0).A != 0 {
		t.Fatal("Scribble must keep the area outside its letters transparent")
	}
	// Server text can contain a full 80-byte field and unsupported glyphs.
	graffitiTexture(atlas, "Some long words followed by unknown bytes: \xff\xfe and punctuation. 0123456789 abcdefghij")
}

func TestGraffitiLifetimeAndRetainedTerrainMesh(t *testing.T) {
	ctx := rogueSkillContext()
	ctx.World.GND = &res.GND{Width: 20, Height: 20, Cells: make([]res.GNDCell, 400)}
	for i := range ctx.World.GND.Cells {
		ctx.World.GND.Cells[i].Heights = [4]float32{3, 3, 3, 3}
	}
	m := &WorldMode{textures: map[string]*render.Image{
		"__effectfile_effect/alpabet.bmp|blackkey=true": render.NewImage(256, 256),
	}}
	entry := network.SkillUnitEntry{ID: 900, CreatorID: 2000000, X: 20, Y: 20, UnitID: 176, Visible: true, HasMessage: true, Message: "Hello"}
	m.applySkillUnitEntry(ctx, entry)
	projection := newSceneProjectionForTarget(800, 600, 20, 20, 0)
	frame := render.NewFrame(800, 600)
	m.drawGraffiti(frame, ctx, projection)
	writing := m.graffiti[900]
	if writing == nil || writing.mesh == nil || writing.mesh.VertexCount() != 18*18*4 || writing.mesh.IndexCount() != 18*18*6 {
		t.Fatalf("missing 18-cell ground writing: %+v", writing)
	}
	if math.Abs(writing.angle-math.Pi) > 1e-9 {
		t.Fatalf("native facing conversion = %g, want pi", writing.angle)
	}
	vertices := reflect.ValueOf(writing.mesh).Elem().FieldByName("vertices")
	for i := 0; i < vertices.Len(); i++ {
		if math.Abs(vertices.Index(i).FieldByName("Y").Float()-3.01) > .0001 {
			t.Fatal("graffiti did not follow the ground")
		}
	}
	cached := writing.mesh
	m.drawGraffiti(render.NewFrame(800, 600), ctx, projection)
	if m.graffiti[900].mesh != cached {
		t.Fatal("rebuilt unchanged graffiti")
	}
	entry.Message = "Replaced"
	m.applySkillUnitEntry(ctx, entry)
	if len(m.graffiti) != 1 || m.graffiti[900].message != "Replaced" || m.graffiti[900].mesh != nil {
		t.Fatal("entry refresh did not replace its text and geometry")
	}
	m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 900})
	if len(m.graffiti) != 0 {
		t.Fatal("Remover/expiry left the writing visible")
	}
	entry.Visible = false
	m.applySkillUnitEntry(ctx, entry)
	if len(m.graffiti) != 0 {
		t.Fatal("hidden writing became visible")
	}
}

func TestGraffitiReleasesImages(t *testing.T) {
	ctx := rogueSkillContext()
	ctx.World.GND = &res.GND{Width: 1, Height: 1, Cells: []res.GNDCell{{}}}
	for _, event := range []string{"replacement", "removal", "hidden", "ordinary unit", "map change", "leave"} {
		t.Run(event, func(t *testing.T) {
			m := &WorldMode{textures: map[string]*render.Image{
				"__effectfile_effect/alpabet.bmp|blackkey=true": render.NewImage(256, 256),
			}}
			entry := network.SkillUnitEntry{ID: 900, Visible: true, HasMessage: true, Message: "Hello"}
			m.applySkillUnitEntry(ctx, entry)
			m.drawGraffiti(render.NewFrame(800, 600), ctx, newSceneProjectionForTarget(800, 600, 0, 0, 0))
			pixels := reflect.ValueOf(m.graffiti[900].mesh).Elem().FieldByName("texture").Elem().FieldByName("pix")
			if pixels.IsNil() {
				t.Fatal("writing has no pixels before removal")
			}
			switch event {
			case "replacement":
				entry.Message = "Replaced"
				m.applySkillUnitEntry(ctx, entry)
			case "removal":
				m.applySkillUnitDisappear(network.SkillUnitDisappear{ID: entry.ID})
			case "hidden":
				entry.Visible = false
				m.applySkillUnitEntry(ctx, entry)
			case "ordinary unit":
				entry.HasMessage = false
				m.applySkillUnitEntry(ctx, entry)
			case "map change":
				m.releaseMapTextures()
			case "leave":
				m.Leave()
			}
			if !pixels.IsNil() {
				t.Fatal("writing retained its pixels instead of releasing its image group")
			}
			m.Leave()
		})
	}
}

func TestGraffitiFollowsGroundTriangles(t *testing.T) {
	for _, tc := range []struct {
		name    string
		heights [4]float32
		grid    [3][3]float64
	}{
		{"raised diagonal", [4]float32{0, 0, 0, 4}, [3][3]float64{{0, 0, 0}, {0, 2, 2}, {0, 2, 4}}},
		{"lowered diagonal", [4]float32{0, 4, 4, 0}, [3][3]float64{{0, 2, 4}, {2, 0, 2}, {4, 2, 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := rogueSkillContext()
			ctx.World.GND = &res.GND{Width: 1, Height: 1, Cells: []res.GNDCell{{Heights: tc.heights}}}
			texture := render.NewImage(512, 512)
			writing := &groundGraffiti{x: 9, y: 9}
			mesh := graffitiMesh(ctx, writing, texture)
			if mesh.VertexCount() != 16 {
				t.Fatalf("expected four quads covering one GND cell, got %d vertices", mesh.VertexCount())
			}
			vertices := reflect.ValueOf(mesh).Elem().FieldByName("vertices")
			for i := 0; i < vertices.Len(); i++ {
				v := vertices.Index(i)
				x, y := int(v.FieldByName("X").Float()), int(v.FieldByName("Z").Float())
				if got, want := v.FieldByName("Y").Float(), tc.grid[y][x]+.01; math.Abs(got-want) > .0001 {
					t.Fatalf("height at %d,%d = %g, want terrain height + lift = %g", x, y, got, want)
				}
			}
			ctx.World.GND.Cells[0].Top = -1
			if mesh := graffitiMesh(ctx, writing, texture); mesh.VertexCount() != 0 {
				t.Fatal("writing appeared over a missing ground surface")
			}
		})
	}
}
