package actors

import (
	"math"
	"time"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

type gr2BindTransform struct {
	position   scene.Point3
	rotation   [4]float32
	scaleShear scene.Mat4
}

type gr2SkeletonPose struct {
	names         []string
	parents       []int32
	bind          []gr2BindTransform
	inverseWorld  []scene.Mat4
	rootPlacement scene.Mat4
}

type gr2AnimationTrack struct {
	name        string
	position    res.GR2Curve
	orientation res.GR2Curve
	scaleShear  res.GR2Curve
}

type gr2AnimationClip struct {
	duration float64
	tracks   map[string]gr2AnimationTrack
}

func gr2SkeletonPoseFromModel(file *res.GR2File, modelIndex int) *gr2SkeletonPose {
	if file == nil || modelIndex < 0 || modelIndex >= len(file.Models) {
		return nil
	}
	model := file.Models[modelIndex]
	if model.SkeletonIndex == nil || *model.SkeletonIndex < 0 || *model.SkeletonIndex >= len(file.Skeletons) {
		return nil
	}
	skeleton := file.Skeletons[*model.SkeletonIndex]
	pose := &gr2SkeletonPose{
		names:         make([]string, 0, len(skeleton.Bones)),
		parents:       make([]int32, 0, len(skeleton.Bones)),
		bind:          make([]gr2BindTransform, 0, len(skeleton.Bones)),
		inverseWorld:  make([]scene.Mat4, 0, len(skeleton.Bones)),
		rootPlacement: gr2TransformMatrix(model.InitialPlacement),
	}
	for _, bone := range skeleton.Bones {
		pose.names = append(pose.names, bone.Name)
		pose.parents = append(pose.parents, bone.ParentIndex)
		pose.bind = append(pose.bind, gr2BindTransformFromTransform(bone.Transform))
		pose.inverseWorld = append(pose.inverseWorld, gr2Mat4FromColumnsOrIdentity(bone.InverseWorld))
	}
	return pose
}

func gr2BindTransformFromTransform(transform res.GR2Transform) gr2BindTransform {
	return gr2BindTransform{
		position:   gr2Position(transform.Position),
		rotation:   transform.Rotation,
		scaleShear: gr2Mat4FromRowMajorMat3(transform.ScaleShear),
	}
}

func gr2TransformMatrix(transform res.GR2Transform) scene.Mat4 {
	bind := gr2BindTransformFromTransform(transform)
	return gr2LocalMatrix(bind.position, bind.rotation, bind.scaleShear)
}

func (p *gr2SkeletonPose) boneCount() int {
	if p == nil {
		return 0
	}
	return len(p.parents)
}

func (p *gr2SkeletonPose) bindPalette() []scene.Mat4 {
	if p == nil {
		return nil
	}
	local := make([]scene.Mat4, len(p.bind))
	for i, bind := range p.bind {
		local[i] = gr2LocalMatrix(bind.position, bind.rotation, bind.scaleShear)
	}
	return p.palette(local)
}

func (p *gr2SkeletonPose) palette(local []scene.Mat4) []scene.Mat4 {
	if p == nil {
		return nil
	}
	world := make([]scene.Mat4, len(p.parents))
	for i := range p.parents {
		parent := p.parents[i]
		if parent >= 0 && int(parent) < i {
			world[i] = scene.Mat4Multiply(world[parent], local[i])
		} else {
			world[i] = scene.Mat4Multiply(p.rootPlacement, local[i])
		}
	}
	palette := make([]scene.Mat4, len(world))
	for i := range world {
		palette[i] = scene.Mat4Multiply(world[i], p.inverseWorld[i])
	}
	return palette
}

func gr2AnimationClipFromFile(file *res.GR2File, animIndex int) *gr2AnimationClip {
	if file == nil || animIndex < 0 || animIndex >= len(file.Animations) {
		return nil
	}
	anim := file.Animations[animIndex]
	clip := &gr2AnimationClip{
		duration: float64(anim.Duration),
		tracks:   make(map[string]gr2AnimationTrack),
	}
	for _, trackGroupIndex := range anim.TrackGroupIndices {
		if trackGroupIndex < 0 || trackGroupIndex >= len(file.TrackGroups) {
			continue
		}
		group := file.TrackGroups[trackGroupIndex]
		for _, track := range group.TransformTracks {
			if _, exists := clip.tracks[track.Name]; exists {
				continue
			}
			clip.tracks[track.Name] = gr2AnimationTrack{
				name:        track.Name,
				position:    track.Position,
				orientation: track.Orientation,
				scaleShear:  track.ScaleShear,
			}
		}
	}
	if clip.duration < 0 {
		clip.duration = 0
	}
	return clip
}

func (c *gr2AnimationClip) skinningPalette(pose *gr2SkeletonPose, t float64) []scene.Mat4 {
	if c == nil || pose == nil {
		return nil
	}
	local := make([]scene.Mat4, pose.boneCount())
	for i := range local {
		bind := pose.bind[i]
		track, ok := c.tracks[pose.names[i]]
		if !ok {
			local[i] = gr2LocalMatrix(bind.position, bind.rotation, bind.scaleShear)
			continue
		}
		local[i] = gr2LocalMatrix(
			gr2EvalVec3(track.position, t, bind.position),
			gr2EvalQuat(track.orientation, t, bind.rotation),
			gr2EvalMat3(track.scaleShear, t, bind.scaleShear),
		)
	}
	return pose.palette(local)
}

func gr2ActionForSpriteState(state sprites.State) res.GR2Action {
	switch state.ActionFamily {
	case scene.SpriteActionWalk:
		return res.GR2ActionMove
	case sprites.ActionNonPCAttack, sprites.ActionPCAttack1, sprites.ActionPCAttack2, sprites.ActionPCAttack3, sprites.ActionPCSkill:
		return res.GR2ActionAttack
	case sprites.ActionNonPCHurt, sprites.ActionPCHurt:
		return res.GR2ActionDamage
	case sprites.ActionNonPCDeath, sprites.ActionPCDeath:
		return res.GR2ActionDead
	default:
		return res.GR2ActionStand
	}
}

func gr2ActionForActionFamily(actionFamily int) (res.GR2Action, bool) {
	action := gr2ActionForSpriteState(sprites.State{ActionFamily: actionFamily})
	return action, action != res.GR2ActionStand
}

func (v *GR2ModelView) paletteForActor(actor worldstate.Actor, state sprites.State, now time.Time) []scene.Mat4 {
	if v == nil || v.pose == nil {
		return nil
	}
	action := gr2ActionForSpriteState(state)
	start := v.started
	if action == res.GR2ActionMove && !actor.MoveStarted.IsZero() {
		start = actor.MoveStarted
	} else if action != res.GR2ActionStand && !state.Started.IsZero() {
		start = state.Started
	}
	if v.clip(action) == nil {
		action = res.GR2ActionStand
		start = v.started
	}
	elapsed := 0.0
	if !start.IsZero() {
		elapsed = now.Sub(start).Seconds()
	}
	return v.skinningPalette(action, elapsed)
}

func (v *GR2ModelView) clip(action res.GR2Action) *gr2AnimationClip {
	if v == nil || int(action) < 0 || int(action) >= len(v.clips) {
		return nil
	}
	return v.clips[action]
}

func (v *GR2ModelView) skinningPalette(action res.GR2Action, elapsed float64) []scene.Mat4 {
	if v == nil || v.pose == nil {
		return nil
	}
	clip := v.clip(action)
	if clip == nil {
		if action != res.GR2ActionStand {
			return v.skinningPalette(res.GR2ActionStand, elapsed)
		}
		return v.bindPalette
	}
	t := 0.0
	if clip.duration > 0 {
		if elapsed < 0 {
			elapsed = 0
		}
		if action == res.GR2ActionDead {
			t = math.Min(elapsed, math.Max(clip.duration-0.0001, 0))
		} else {
			t = math.Mod(elapsed, clip.duration)
			if t < 0 {
				t += clip.duration
			}
		}
	}
	return clip.skinningPalette(v.pose, t)
}

func (v *GR2ModelView) actionDuration(action res.GR2Action) time.Duration {
	if v == nil {
		return 0
	}
	if clip := v.clip(action); clip != nil && clip.duration > 0 {
		return time.Duration(clip.duration * float64(time.Second))
	}
	return 0
}

func gr2SkinnedPoint(vertex res.GR2ModelVertex, palette []scene.Mat4) scene.Point3 {
	base := scene.Point3{
		X: float64(vertex.Position[0]),
		Y: float64(vertex.Position[1]),
		Z: float64(vertex.Position[2]),
	}
	if len(palette) == 0 {
		return base
	}
	var out scene.Point3
	applied := false
	for i, rawWeight := range vertex.BoneWeights {
		if rawWeight == 0 {
			continue
		}
		boneIndex := int(vertex.BoneIndices[i])
		if boneIndex < 0 || boneIndex >= len(palette) {
			continue
		}
		weight := float64(rawWeight) / 255
		point := scene.Mat4TransformPoint(palette[boneIndex], base)
		out = scene.Add3(out, scene.Mul3(point, weight))
		applied = true
	}
	if !applied {
		return base
	}
	return out
}

func gr2SkinnedNormal(vertex res.GR2ModelVertex, palette []scene.Mat4) scene.Point3 {
	base := scene.Point3{
		X: float64(vertex.Normal[0]),
		Y: float64(vertex.Normal[1]),
		Z: float64(vertex.Normal[2]),
	}
	if len(palette) == 0 {
		return base
	}
	var out scene.Point3
	applied := false
	for i, rawWeight := range vertex.BoneWeights {
		if rawWeight == 0 {
			continue
		}
		boneIndex := int(vertex.BoneIndices[i])
		if boneIndex < 0 || boneIndex >= len(palette) {
			continue
		}
		weight := float64(rawWeight) / 255
		normal := scene.Mat4TransformVector(palette[boneIndex], base)
		out = scene.Add3(out, scene.Mul3(normal, weight))
		applied = true
	}
	if !applied {
		return base
	}
	return out
}

func gr2LocalMatrix(position scene.Point3, rotation [4]float32, scaleShear scene.Mat4) scene.Mat4 {
	matrix := scene.Mat4Identity()
	matrix = scene.Mat4Translate(matrix, position)
	matrix = scene.Mat4RotateQuat(matrix, rotation)
	matrix = scene.Mat4Multiply(matrix, scaleShear)
	return matrix
}

func gr2Position(position [3]float32) scene.Point3 {
	return scene.Point3{X: float64(position[0]), Y: float64(position[1]), Z: float64(position[2])}
}

func gr2Mat4FromRowMajorMat3(values [9]float32) scene.Mat4 {
	out := scene.Mat4Identity()
	out[0] = float64(values[0])
	out[1] = float64(values[3])
	out[2] = float64(values[6])
	out[4] = float64(values[1])
	out[5] = float64(values[4])
	out[6] = float64(values[7])
	out[8] = float64(values[2])
	out[9] = float64(values[5])
	out[10] = float64(values[8])
	return out
}

func gr2Mat4FromColumnsOrIdentity(values [16]float32) scene.Mat4 {
	zero := true
	var out scene.Mat4
	for i, value := range values {
		if value != 0 {
			zero = false
		}
		out[i] = float64(value)
	}
	if zero {
		return scene.Mat4Identity()
	}
	return out
}

func gr2EvalVec3(curve res.GR2Curve, t float64, fallback scene.Point3) scene.Point3 {
	values, ok := gr2EvalScalars(curve, 3, t)
	if !ok {
		return fallback
	}
	return scene.Point3{X: values[0], Y: values[1], Z: values[2]}
}

func gr2EvalQuat(curve res.GR2Curve, t float64, fallback [4]float32) [4]float32 {
	values, ok := gr2EvalScalars(curve, 4, t)
	if !ok {
		return fallback
	}
	length := math.Sqrt(values[0]*values[0] + values[1]*values[1] + values[2]*values[2] + values[3]*values[3])
	if length <= 0 {
		return fallback
	}
	return [4]float32{
		float32(values[0] / length),
		float32(values[1] / length),
		float32(values[2] / length),
		float32(values[3] / length),
	}
}

func gr2EvalMat3(curve res.GR2Curve, t float64, fallback scene.Mat4) scene.Mat4 {
	values, ok := gr2EvalScalars(curve, 9, t)
	if !ok {
		return fallback
	}
	return gr2Mat4FromRowMajorMat3([9]float32{
		float32(values[0]), float32(values[1]), float32(values[2]),
		float32(values[3]), float32(values[4]), float32(values[5]),
		float32(values[6]), float32(values[7]), float32(values[8]),
	})
}

func gr2EvalScalars(curve res.GR2Curve, dim int, t float64) ([]float64, bool) {
	if dim <= 0 || len(curve.Controls) < dim {
		return nil, false
	}
	if curve.Degree < 2 || len(curve.Knots) < 3 {
		out := make([]float64, dim)
		for i := range out {
			out[i] = float64(curve.Controls[i])
		}
		return out, true
	}
	return gr2Deboor2(curve.Knots, curve.Controls, dim, t)
}

func gr2Deboor2(knots, controls []float32, dim int, t float64) ([]float64, bool) {
	pointCount := len(controls) / dim
	if len(knots) == 0 || pointCount == 0 {
		return nil, false
	}
	span := 0
	for span < len(knots) && float64(knots[span]) <= t {
		span++
	}
	span = scene.ClampInt(span, 2, len(knots)-1)
	knot := func(i int) float64 {
		i = scene.ClampInt(i, 0, len(knots)-1)
		return float64(knots[i])
	}
	control := func(point, d int) float64 {
		point = scene.ClampInt(point, 0, pointCount-1)
		return float64(controls[point*dim+d])
	}
	ka := knot(span - 2)
	kb := knot(span - 1)
	kc := knot(span)
	kd := knot(span + 1)
	a := gr2SafeDiv(t-kb, kc-kb)
	b := gr2SafeDiv(t-ka, kc-ka)
	c := gr2SafeDiv(t-kb, kd-kb)
	i0, i1, i2 := span-2, span-1, span
	out := make([]float64, dim)
	for d := range out {
		e1 := (1-b)*control(i0, d) + b*control(i1, d)
		e2 := (1-c)*control(i1, d) + c*control(i2, d)
		out[d] = (1-a)*e1 + a*e2
	}
	return out, true
}

func gr2SafeDiv(num, den float64) float64 {
	if math.Abs(den) < 1e-12 {
		return 0
	}
	return num / den
}
