package actors

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const gr2ModelWorldScale = 0.2

type GR2ModelView struct {
	geometry       *res.GR2Geometry
	textures       []*render.Image
	emblemTextures map[int]struct{}
	pose           *gr2SkeletonPose
	bindPalette    []scene.Mat4
	clips          [5]*gr2AnimationClip
	source         string
	started        time.Time
}

func (m *System) nonPCActorHasGR2Model(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.Resources == nil {
		return false
	}
	resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job))
	return ok && res.IsGR2ResourceName(resourceName)
}

func (m *System) drawNonPCGR2Model3D(screen *render.Frame, ctx client.Context, entry scene.ActorDrawEntry, shadow scene.LightTint) bool {
	actor := entry.Actor
	view := m.NonPCGR2ModelView(ctx, actor)
	if view == nil || view.geometry == nil || len(view.geometry.Vertices) == 0 {
		return false
	}
	now := time.Now()
	state := m.NonPCSpriteState(actor, now)
	palette := view.paletteForActor(actor, state, now)
	scale := gr2ModelWorldScale * m.ActorBodySizeMultiplier(actor.ID, now)
	matrix := gr2ActorModelMatrix(entry.WorldX, entry.WorldY, entry.WorldZ, actor.Dir, scale)
	lighting := m.bindings.Map.Lighting(nil)
	if ctx.World != nil {
		lighting = m.bindings.Map.Lighting(ctx.World.RSW)
	}
	tint := entry.Stealth.Tint(m.actorRenderTint(actor, now))
	alpha := m.actorVisualAlpha(actor.ID, now)
	if alpha <= 0 {
		return true
	}
	if m.bindings.Map.WhitePixel == nil {
		m.bindings.Map.WhitePixel = render.NewImage(1, 1)
		m.bindings.Map.WhitePixel.Fill(color.White)
	}
	options := gr2ModelDrawOptions()
	options.Blend = entry.Stealth.Blend(m.ActorRenderBlend(actor.ID, now))
	if options.Blend == render.BlendLighter {
		options.DepthWrite = false
	}
	for _, batch := range view.geometry.Batches {
		texture := m.gr2BatchTexture(ctx, actor, view, batch.TextureIndex)
		if texture == nil {
			texture = m.bindings.Map.WhitePixel
		}
		bounds := texture.Bounds()
		textureWidth := float32(bounds.Dx())
		textureHeight := float32(bounds.Dy())
		drawBatch := scene.ModelDrawBatch{Screen: screen, Texture: texture, Options: *options}
		start := int(batch.StartIndex)
		end := start + int(batch.IndexCount)
		if start < 0 || start >= len(view.geometry.Indices) {
			continue
		}
		if end > len(view.geometry.Indices) {
			end = len(view.geometry.Indices)
		}
		for i := start; i+2 < end; i += 3 {
			a, okA := gr2ModelVertex3D(view.geometry, int(view.geometry.Indices[i]), matrix, palette, lighting, tint, shadow, alpha, textureWidth, textureHeight)
			b, okB := gr2ModelVertex3D(view.geometry, int(view.geometry.Indices[i+1]), matrix, palette, lighting, tint, shadow, alpha, textureWidth, textureHeight)
			c, okC := gr2ModelVertex3D(view.geometry, int(view.geometry.Indices[i+2]), matrix, palette, lighting, tint, shadow, alpha, textureWidth, textureHeight)
			if okA && okB && okC {
				drawBatch.AddTriangle(a, b, c)
			}
		}
		drawBatch.Flush()
	}
	return true
}

func (m *System) gr2BatchTexture(ctx client.Context, actor worldstate.Actor, view *GR2ModelView, index int) *render.Image {
	if view == nil || index < 0 || index >= len(view.textures) {
		return nil
	}
	if _, ok := view.emblemTextures[index]; ok {
		if emblem := m.bindings.Social.GuildFlagEmblem(ctx, actor); emblem != nil {
			return emblem
		}
	}
	return view.textures[index]
}

func (m *System) NonPCGR2ModelView(ctx client.Context, actor worldstate.Actor) *GR2ModelView {
	if ctx.Config.Headless {
		return nil
	}
	job := int(actor.Job)
	if _, ok := m.GR2ModelMiss[job]; ok {
		return nil
	}
	if m.GR2Models == nil {
		m.GR2Models = make(map[int]*GR2ModelView)
	}
	if view, ok := m.GR2Models[job]; ok {
		return view
	}
	if ctx.Resources == nil {
		return nil
	}
	resourceName, ok := ctx.Resources.NonPCResourceName(job)
	if !ok || !res.IsGR2ResourceName(resourceName) {
		return nil
	}
	loaded, status := loadGR2ModelView(ctx.Resources, resourceName)
	if loaded == nil {
		if m.GR2ModelMiss == nil {
			m.GR2ModelMiss = make(map[int]struct{})
		}
		m.GR2ModelMiss[job] = struct{}{}
		glog.Warnf("gr2 model unavailable id=%d job=%d resource=%s: %s", actor.ID, job, resourceName, status)
		return nil
	}
	m.GR2Models[job] = loaded
	glog.Debugf("gr2 model resources id=%d job=%d resource=%s %s", actor.ID, job, resourceName, status)
	return loaded
}

func loadGR2ModelView(manager *res.Manager, resourceName string) (*GR2ModelView, string) {
	data, source, err := readFirstResource(manager, res.GR2ModelResourceCandidates(resourceName))
	if err != nil {
		return nil, "model=missing"
	}
	file, err := res.ParseGR2(data)
	if err != nil {
		return nil, "model=" + source + " parse-error=" + err.Error()
	}
	geometry, err := res.BuildGR2Geometry(file, 0)
	if err != nil {
		return nil, "model=" + source + " geometry-error=" + err.Error()
	}
	pose, err := gr2SkeletonPoseFromModel(file, 0)
	if err != nil {
		return nil, "model=" + source + " pose-error=" + err.Error()
	}
	if pose == nil {
		return nil, "model=" + source + " pose=missing"
	}
	clips, clipStatus := loadGR2ModelAnimationClips(manager, resourceName, file)
	textures := make([]*render.Image, len(file.Textures))
	emblemTextures := make(map[int]struct{})
	loadedTextures := 0
	for i, texture := range file.Textures {
		if strings.Contains(strings.ToLower(texture.FromFileName), "emblem") {
			emblemTextures[i] = struct{}{}
		}
		img, err := texture.Image()
		if err != nil {
			glog.Warnf("gr2 texture unavailable model=%s texture=%s encoding=%d: %v", source, texture.FromFileName, texture.Encoding, err)
			continue
		}
		textures[i] = render.NewImageFromImage(img)
		loadedTextures++
	}
	return &GR2ModelView{
		geometry:       geometry,
		textures:       textures,
		emblemTextures: emblemTextures,
		pose:           pose,
		bindPalette:    pose.bindPalette(),
		clips:          clips,
		source:         source,
		started:        time.Now(),
	}, fmt.Sprintf("model=%s vertices=%d indices=%d batches=%d textures=%d/%d%s", source, len(geometry.Vertices), len(geometry.Indices), len(geometry.Batches), loadedTextures, len(file.Textures), clipStatus)
}

func loadGR2ModelAnimationClips(manager *res.Manager, resourceName string, modelFile *res.GR2File) ([5]*gr2AnimationClip, string) {
	var clips [5]*gr2AnimationClip
	loaded := 0
	if clip := gr2AnimationClipFromFile(modelFile, 0); clip != nil {
		clips[res.GR2ActionStand] = clip
		loaded++
	}
	boneType, ok := res.GR2BoneTypeFromName(resourceName)
	if !ok {
		return clips, fmt.Sprintf(" clips=%d/5 bone=unknown", loaded)
	}
	for _, action := range []res.GR2Action{res.GR2ActionMove, res.GR2ActionAttack, res.GR2ActionDead, res.GR2ActionDamage} {
		data, source, err := readFirstResource(manager, res.GR2AnimationResourceCandidates(boneType, action))
		if err != nil {
			continue
		}
		file, err := res.ParseGR2(data)
		if err != nil {
			glog.Warnf("gr2 animation unavailable bone=%d action=%d source=%s: %v", boneType, action, source, err)
			continue
		}
		clip := gr2AnimationClipFromFile(file, 0)
		if clip == nil {
			glog.Warnf("gr2 animation has no clip bone=%d action=%d source=%s", boneType, action, source)
			continue
		}
		clips[action] = clip
		loaded++
	}
	return clips, fmt.Sprintf(" clips=%d/5 bone=%d", loaded, boneType)
}

func gr2ModelVertex3D(geometry *res.GR2Geometry, index int, matrix scene.Mat4, palette []scene.Mat4, lighting maps.Lighting, tint color.RGBA, shadow scene.LightTint, alpha float64, textureWidth, textureHeight float32) (render.Vertex3D, bool) {
	if geometry == nil || index < 0 || index >= len(geometry.Vertices) {
		return render.Vertex3D{}, false
	}
	vertex := geometry.Vertices[index]
	point := scene.Mat4TransformPoint(matrix, gr2SkinnedPoint(vertex, palette))
	normal := scene.Normalize3(scene.Mat4TransformVector(matrix, gr2SkinnedNormal(vertex, palette)))
	if normal == (scene.Point3{}) {
		normal = scene.Point3{Y: 1}
	}
	scale := lighting.ModelScaleNormalized(normal)
	shadow = shadow.Clamped()
	color := color.RGBA{
		R: scene.ClampColor(float64(tint.R) * scale.X * shadow.R),
		G: scene.ClampColor(float64(tint.G) * scale.Y * shadow.G),
		B: scene.ClampColor(float64(tint.B) * scale.Z * shadow.B),
		A: scene.ClampColor(float64(tint.A) * alpha),
	}
	return scene.TexturedSurfaceVertex3D(point, scene.UV{U: vertex.UV[0], V: vertex.UV[1]}, color, textureWidth, textureHeight), true
}

func gr2ActorModelMatrix(worldX, worldY, worldZ float64, direction int, scale float64) scene.Mat4 {
	matrix := scene.Mat4Identity()
	matrix = scene.Mat4Translate(matrix, scene.Point3{X: worldX, Y: worldZ, Z: worldY})
	// Classic stands GR2 models upright with a 90 degree X rotation. Goro flips
	// the sign so GR2 +Z points up in its positive-height world axis; that also
	// reverses the local front axis, so yaw needs a 180 degree compensation.
	matrix = scene.Mat4RotateY(matrix, gr2ModelFacingYaw(direction)+math.Pi)
	matrix = scene.Mat4RotateX(matrix, -math.Pi/2)
	matrix = scene.Mat4Scale(matrix, scene.Point3{X: scale, Y: scale, Z: scale})
	return matrix
}

func gr2ModelFacingYaw(direction int) float64 {
	return math.Pi - float64(scene.NormalizeDirectionIndex(direction))*(2*math.Pi/8)
}

func gr2ModelDrawOptions() *render.DrawTrianglesOptions {
	options := maps.WorldOpaqueTriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendSourceOver
	return options
}

func readFirstResource(manager *res.Manager, candidates []string) ([]byte, string, error) {
	for _, candidate := range candidates {
		data, err := manager.ReadFile(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("not found")
}
