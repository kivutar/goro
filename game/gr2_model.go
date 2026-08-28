package game

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const gr2ModelWorldScale = 0.2

type gr2ModelView struct {
	geometry       *res.GR2Geometry
	textures       []*render.Image
	emblemTextures map[int]struct{}
	pose           *gr2SkeletonPose
	bindPalette    []mat4
	clips          [5]*gr2AnimationClip
	source         string
	started        time.Time
}

func (m *WorldMode) nonPCActorHasGR2Model(ctx client.Context, actor worldstate.Actor) bool {
	if ctx.Resources == nil {
		return false
	}
	resourceName, ok := ctx.Resources.NonPCResourceName(int(actor.Job))
	return ok && res.IsGR2ResourceName(resourceName)
}

func (m *WorldMode) drawNonPCGR2Model3D(screen *render.Frame, ctx client.Context, entry sceneActorDrawEntry, shadow float64) bool {
	actor := entry.actor
	view := m.nonPCGR2ModelView(ctx, actor)
	if view == nil || view.geometry == nil || len(view.geometry.Vertices) == 0 {
		return false
	}
	now := time.Now()
	state := m.nonPCSpriteState(actor, now)
	palette := view.paletteForActor(actor, state, now)
	scale := gr2ModelWorldScale * m.actorBodySizeMultiplier(actor.ID, now)
	matrix := gr2ActorModelMatrix(entry.worldX, entry.worldY, entry.worldZ, actor.Dir, scale)
	lighting := m.sceneLighting(nil)
	if ctx.World != nil {
		lighting = m.sceneLighting(ctx.World.RSW)
	}
	tint := m.actorRenderTint(actor, now)
	alpha := m.actorVisualAlpha(actor.ID, now)
	if alpha <= 0 {
		return true
	}
	if m.whitePixel == nil {
		m.whitePixel = render.NewImage(1, 1)
		m.whitePixel.Fill(color.White)
	}
	options := gr2ModelDrawOptions()
	for _, batch := range view.geometry.Batches {
		texture := m.gr2BatchTexture(ctx, actor, view, batch.TextureIndex)
		if texture == nil {
			texture = m.whitePixel
		}
		bounds := texture.Bounds()
		textureWidth := float32(bounds.Dx())
		textureHeight := float32(bounds.Dy())
		drawBatch := animatedRSMDrawBatch{screen: screen, texture: texture, options: *options}
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
				drawBatch.addTriangle(a, b, c)
			}
		}
		drawBatch.flush()
	}
	return true
}

func (m *WorldMode) gr2BatchTexture(ctx client.Context, actor worldstate.Actor, view *gr2ModelView, index int) *render.Image {
	if view == nil || index < 0 || index >= len(view.textures) {
		return nil
	}
	if _, ok := view.emblemTextures[index]; ok {
		if emblem := m.gr2ActorGuildEmblemTexture(ctx, actor); emblem != nil {
			return emblem
		}
	}
	return view.textures[index]
}

func (m *WorldMode) gr2ActorGuildEmblemTexture(ctx client.Context, actor worldstate.Actor) *render.Image {
	if actor.GuildID == 0 || actor.EmblemVersion == 0 {
		return nil
	}
	if m.guildEmblems == nil {
		m.requestActorGuildEmblem(ctx, actor.GuildID, actor.EmblemVersion)
		return nil
	}
	emblem := m.guildEmblems[actor.GuildID]
	if emblem.flagImage == nil || emblem.version < actor.EmblemVersion {
		m.requestActorGuildEmblem(ctx, actor.GuildID, actor.EmblemVersion)
		return nil
	}
	return emblem.flagImage
}

func (m *WorldMode) nonPCGR2ModelView(ctx client.Context, actor worldstate.Actor) *gr2ModelView {
	job := int(actor.Job)
	if _, ok := m.gr2ModelMiss[job]; ok {
		return nil
	}
	if m.gr2Models == nil {
		m.gr2Models = make(map[int]*gr2ModelView)
	}
	if view, ok := m.gr2Models[job]; ok {
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
		if m.gr2ModelMiss == nil {
			m.gr2ModelMiss = make(map[int]struct{})
		}
		m.gr2ModelMiss[job] = struct{}{}
		glog.Warnf("gr2 model unavailable id=%d job=%d resource=%s: %s", actor.ID, job, resourceName, status)
		return nil
	}
	m.gr2Models[job] = loaded
	glog.Debugf("gr2 model resources id=%d job=%d resource=%s %s", actor.ID, job, resourceName, status)
	return loaded
}

func loadGR2ModelView(manager *res.Manager, resourceName string) (*gr2ModelView, string) {
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
	return &gr2ModelView{
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

func gr2ModelVertex3D(geometry *res.GR2Geometry, index int, matrix mat4, palette []mat4, lighting sceneLighting, tint color.RGBA, shadow, alpha float64, textureWidth, textureHeight float32) (render.Vertex3D, bool) {
	if geometry == nil || index < 0 || index >= len(geometry.Vertices) {
		return render.Vertex3D{}, false
	}
	vertex := geometry.Vertices[index]
	point := mat4TransformPoint(matrix, gr2SkinnedPoint(vertex, palette))
	normal := normalize3(mat4TransformVector(matrix, gr2SkinnedNormal(vertex, palette)))
	if normal == (modelPoint3{}) {
		normal = modelPoint3{y: 1}
	}
	scale := lighting.modelScaleNormalized(normal)
	color := color.RGBA{
		R: clampColor(float64(tint.R) * scale.x * shadow),
		G: clampColor(float64(tint.G) * scale.y * shadow),
		B: clampColor(float64(tint.B) * scale.z * shadow),
		A: clampColor(float64(tint.A) * alpha),
	}
	return texturedSurfaceVertex3D(point, texturePoint{u: vertex.UV[0], v: vertex.UV[1]}, color, textureWidth, textureHeight), true
}

func gr2ActorModelMatrix(worldX, worldY, worldZ float64, direction int, scale float64) mat4 {
	matrix := mat4Identity()
	matrix = mat4Translate(matrix, modelPoint3{x: worldX, y: worldZ, z: worldY})
	// Classic stands GR2 models upright with a 90 degree X rotation. Goro flips
	// the sign so GR2 +Z points up in its positive-height world axis; that also
	// reverses the local front axis, so yaw needs a 180 degree compensation.
	matrix = mat4RotateY(matrix, gr2ModelFacingYaw(direction)+math.Pi)
	matrix = mat4RotateX(matrix, -math.Pi/2)
	matrix = mat4Scale(matrix, modelPoint3{x: scale, y: scale, z: scale})
	return matrix
}

func gr2ModelFacingYaw(direction int) float64 {
	return math.Pi - float64(normalizeDirectionIndex(direction))*(2*math.Pi/8)
}

func mat4TransformVector(matrix mat4, point modelPoint3) modelPoint3 {
	return modelPoint3{
		x: matrix[0]*point.x + matrix[4]*point.y + matrix[8]*point.z,
		y: matrix[1]*point.x + matrix[5]*point.y + matrix[9]*point.z,
		z: matrix[2]*point.x + matrix[6]*point.y + matrix[10]*point.z,
	}
}

func gr2ModelDrawOptions() *render.DrawTrianglesOptions {
	options := worldOpaqueTriangleDrawOptions(render.FilterLinear, render.AddressRepeat)
	options.Blend = render.BlendSourceOver
	return options
}
