package battle

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/world"
)

const (
	damageNumberSPR = "data\\sprite\\이팩트\\숫자.spr"
	damageNumberACT = "data\\sprite\\이팩트\\숫자.act"
	damageMsgSPR    = "data\\sprite\\이팩트\\msg.spr"
	damageMsgACT    = "data\\sprite\\이팩트\\msg.act"
)

// Colours passed to CGameActor::Am_Make_Number and AM_MAKE_MISS.
var (
	DamageFloaterWhite      = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterCritical   = color.RGBA{R: 255, G: 255, B: 0, A: 255}  // 0xffffff00
	damageFloaterComboTotal = color.RGBA{R: 255, G: 255, B: 85, A: 255} // 0xffffff55
	damageFloaterRedMiss    = color.RGBA{R: 255, G: 0, B: 0, A: 255}    // misses on players
)

func (m *System) damageNumberSprite(ctx client.Context) *scene.SpriteView {
	if m.DamageNumberView != nil || m.DamageNumberMiss || ctx.Resources == nil {
		return m.DamageNumberView
	}
	view, status := scene.LoadSpriteView(ctx.Resources,
		[]string{damageNumberACT, strings.ReplaceAll(damageNumberACT, "\\", "/")},
		[]string{damageNumberSPR, strings.ReplaceAll(damageNumberSPR, "\\", "/")},
		nil,
		"damage number",
	)
	if view == nil {
		m.DamageNumberMiss = true
		glog.Warnf("damage number sprite unavailable: %s", status)
		return nil
	}
	m.DamageNumberView = view
	glog.Debugf("damage number sprite resources %s", status)
	return view
}

func (m *System) DamageNumberBillboard(ctx client.Context, text string) (*scene.Billboard, bool) {
	if text == "" || !allASCIIDigits(text) {
		return nil, false
	}
	if m.DamageNumbers == nil {
		m.DamageNumbers = make(map[string]*scene.Billboard)
	}
	if billboard, ok := m.DamageNumbers[text]; ok {
		return billboard, true
	}
	view := m.damageNumberSprite(ctx)
	if view == nil {
		return nil, false
	}
	billboard, ok := composeDamageNumberBillboard(view, text)
	if !ok {
		return nil, false
	}
	m.DamageNumbers[text] = billboard
	return billboard, true
}

func (m *System) damageMessageSprite(ctx client.Context) *scene.SpriteView {
	if m.DamageMsgView != nil || m.DamageMsgMiss || ctx.Resources == nil {
		return m.DamageMsgView
	}
	view, status := scene.LoadSpriteView(ctx.Resources,
		[]string{damageMsgACT, strings.ReplaceAll(damageMsgACT, "\\", "/")},
		[]string{damageMsgSPR, strings.ReplaceAll(damageMsgSPR, "\\", "/")},
		nil,
		"damage message",
	)
	if view == nil {
		m.DamageMsgMiss = true
		glog.Warnf("damage message sprite unavailable: %s", status)
		return nil
	}
	m.DamageMsgView = view
	glog.Debugf("damage message sprite resources %s", status)
	return view
}

func (m *System) damageMessageBillboard(ctx client.Context, actionIndex, motion int) (*scene.Billboard, bool) {
	view := m.damageMessageSprite(ctx)
	if view == nil || view.Act == nil {
		return nil, false
	}
	if actionIndex < 0 || actionIndex >= len(view.Act.Actions) {
		return nil, false
	}
	action := view.Act.Actions[actionIndex]
	if motion < 0 || motion >= len(action.Animations) {
		return nil, false
	}
	key := scene.SpriteBillboardKey{ActionIndex: actionIndex, Motion: motion}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, action.Animations[motion])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

// composeDamageNumberBillboard lays digits out as Am_Make_Number does: each
// digit is its own sprite, shifted 8 source pixels from its neighbour around
// the number's centre.
func composeDamageNumberBillboard(view *scene.SpriteView, text string) (*scene.Billboard, bool) {
	const fontWidth = 8
	digits := make([]*scene.Billboard, 0, len(text))
	for _, ch := range text {
		billboard, ok := damageDigitBillboard(view, int(ch-'0'))
		if !ok {
			return nil, false
		}
		digits = append(digits, billboard)
	}
	n := len(digits)
	if n == 0 {
		return nil, false
	}
	shift := func(index int) float64 { return float64(fontWidth*index - fontWidth/2*(n-1)) }
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for index, digit := range digits {
		bounds := digit.Image.Bounds()
		left, top := shift(index)-digit.AnchorX, -digit.AnchorY
		minX, minY = math.Min(minX, left), math.Min(minY, top)
		maxX, maxY = math.Max(maxX, left+float64(bounds.Dx())), math.Max(maxY, top+float64(bounds.Dy()))
	}
	width, height := int(math.Ceil(maxX-minX)), int(math.Ceil(maxY-minY))
	if width <= 0 || height <= 0 {
		return nil, false
	}
	target := render.NewImage(width, height)
	for index, digit := range digits {
		var opts render.DrawImageOptions
		opts.GeoM.Translate(shift(index)-digit.AnchorX-minX, -digit.AnchorY-minY)
		opts.Filter = scene.SpriteCompositionFilter()
		target.DrawImage(digit.Image, &opts)
	}
	return &scene.Billboard{Image: target, AnchorX: -minX, AnchorY: -minY}, true
}

func damageDigitBillboard(view *scene.SpriteView, digit int) (*scene.Billboard, bool) {
	if view == nil || view.Act == nil || len(view.Act.Actions) == 0 || digit < 0 || digit > 9 {
		return nil, false
	}
	action := view.Act.Actions[0]
	if digit >= len(action.Animations) {
		return nil, false
	}
	key := scene.SpriteBillboardKey{ActionIndex: 0, Motion: digit}
	if billboard, ok := view.Billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := scene.ComposeSingleSpriteBillboard(view, action.Animations[digit])
	if !ok {
		return nil, false
	}
	view.Billboards[key] = billboard
	return billboard, true
}

func allASCIIDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// damageFloaterTicks is CRenderObject::GetStateCnt: elapsed milliseconds / 24.
func damageFloaterTicks(floater Floater, now time.Time) float64 {
	return float64(now.Sub(floater.Starts)) / float64(time.Millisecond) / 24
}

// damageFloaterPlacement follows CMsgEffect::OnProcess. Offsets are in Goro
// units (five client units each); up is the height above the actor's feet.
func damageFloaterPlacement(kind FloaterKind, dir int, ticks float64) (dx, dy, up, zoom, alpha float64) {
	t := math.Max(0, ticks)
	switch kind {
	case FloaterMiss:
		// MET_MISS: 12 units up, rising 0.54 per tick.
		return 0, 0, (12 + .54*t) / 5, 1, (250 - 3*t) / 255
	case FloaterRecoveryHP, FloaterRecoverySP:
		// MET_NUMBER + 100: 12 units up, rising 0.18 per tick.
		return 0, 0, (12 + .18*t) / 5, math.Max(5-.24*t, 1), (250 - 2*t) / 255
	case FloaterCombo:
		// MET_NUMBER_TOTAL: 30 units up, rising 0.1 per tick, fading after 90.
		alpha = 250 - math.Max(0, (t-90)*8)
		return 0, 0, (30 + .1*t) / 5, damageFloaterComboZoom(ticks), alpha / 255
	}
	mov, minZoom := .8, 1.2
	if kind == FloaterCritical {
		mov, minZoom = .5, 1.3
	}
	mx, mz := damageFloaterDrift(dir, mov)
	drift := t/3 + 3
	// Created 15 units up, drawn 8 lower, then an arc peaking at tick 30;
	// m_destPos.y lowers it by another 4/3.
	up = (15 - 8 + t*(2-t/30) - 4.0/3) / 5
	return mx * drift / 5, mz * drift / 5, up, math.Max(5-.24*t, minZoom), (250 - 3.4*t) / 255
}

// damageFloaterDrift picks the drift direction from the target's facing
// quadrant. CGameActor stores a facing as dir * 45 + 180 degrees.
func damageFloaterDrift(dir int, mov float64) (x, z float64) {
	roty := (dir*45 + 180) % 360
	switch {
	case roty < 90:
		return -mov, mov
	case roty < 180:
		return -mov, -mov
	case roty < 270:
		return mov, -mov
	default:
		return mov, mov
	}
}

// damageFloaterComboZoom replays the combo total's growth: it starts at 0.5
// and each 60 Hz update adds 0.18 times the elapsed ticks, up to 3.
func damageFloaterComboZoom(ticks float64) float64 {
	frames := math.Floor(ticks * 24 / (1000.0 / 60))
	tickPerFrame := 1000.0 / 60 / 24
	return math.Min(3, .5+.18*tickPerFrame*frames*(frames+1)/2)
}

func damageFloaterColor(kind FloaterKind, fallback color.RGBA) color.RGBA {
	switch kind {
	case FloaterCritical:
		return damageFloaterCritical
	case FloaterCombo:
		return damageFloaterComboTotal
	case FloaterRecoveryHP:
		return color.RGBA{R: 0, G: 255, B: 0, A: 255}
	case FloaterRecoverySP:
		return color.RGBA{R: 0, G: 0, B: 255, A: 255}
	default:
		if fallback.A != 0 {
			return fallback
		}
		return DamageFloaterWhite
	}
}

type Floater struct {
	ActorID uint32
	X       int
	Y       int
	// Dir is the target's facing when hit; it sets the drift direction.
	Dir     int
	Text    string
	Color   color.RGBA
	Kind    FloaterKind
	Starts  time.Time
	Expires time.Time
}

type FloaterKind int

const (
	FloaterNormal FloaterKind = iota
	FloaterCritical
	FloaterRecoveryHP
	FloaterRecoverySP
	FloaterMiss
	FloaterCombo
)

// The classic client shows the same numbers whoever attacks or is hit, so
// the local-actor flags are unused.
func (m *System) AddActionDamageFloaters(ctx client.Context, action network.ActorActionNotify, target world.Actor, targetOK, _, _ bool, x, y int, hitAt time.Time) {
	if ctx.Config.Headless {
		return
	}
	text, kind, floaterColor := actionDamageFloater(action, target, targetOK)
	if text == "" {
		return
	}
	if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(kind) {
		return
	}
	dir := 0
	if targetOK {
		dir = target.Dir
	}
	count := scene.ActionVisualHitCount(action)
	total := action.Damage + action.LeftDamage
	if count <= 1 || total <= 0 {
		m.addDamageFloater(action.TargetID, x, y, dir, text, floaterColor, kind, hitAt)
		return
	}
	// Multi-hit attacks show each hit plus a running total (AM_ATTACKED_MANYTIMES).
	values := actionDamageFloaterValues(action, count)
	for i := 0; i < count; i++ {
		starts := hitAt.Add(scene.MultiHitDelay * time.Duration(i))
		m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(values[i]), floaterColor, kind, starts)
		if action.Damage > 0 {
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(actionComboDamageFloaterValue(action, count, i)), damageFloaterComboTotal, FloaterCombo, starts)
		}
	}
}

func damageFloaterHiddenInSiege(kind FloaterKind) bool {
	switch kind {
	case FloaterNormal, FloaterCritical, FloaterCombo:
		return true
	default:
		return false
	}
}

func (m *System) AddDamageFloater(actorID uint32, x, y int, text string, floaterColor color.RGBA, kind FloaterKind, starts time.Time) {
	m.addDamageFloater(actorID, x, y, 0, text, floaterColor, kind, starts)
}

func (m *System) addDamageFloater(actorID uint32, x, y, dir int, text string, floaterColor color.RGBA, kind FloaterKind, starts time.Time) {
	m.DamageFloaters = append(m.DamageFloaters, Floater{
		ActorID: actorID,
		X:       x,
		Y:       y,
		Dir:     dir,
		Text:    text,
		Color:   floaterColor,
		Kind:    kind,
		Starts:  starts,
		Expires: starts.Add(DamageFloaterDuration(kind)),
	})
}

func actionDamageFloaterValues(action network.ActorActionNotify, count int) []int {
	values := make([]int, count)
	if count <= 0 {
		return values
	}
	total := int(action.Damage + action.LeftDamage)
	if action.SkillID != 0 && action.Damage > 0 {
		value := int(math.Floor(float64(action.Damage) / float64(count)))
		for i := range values {
			values[i] = value
		}
		return values
	}
	base := total / count
	rem := total % count
	for i := range values {
		values[i] = base
		if i < rem {
			values[i]++
		}
	}
	return values
}

func actionComboDamageFloaterValue(action network.ActorActionNotify, count, index int) int {
	if action.SkillID != 0 && action.Damage > 0 && count > 0 {
		return int(math.Floor((float64(action.Damage) / float64(count)) * float64(index+1)))
	}
	values := actionDamageFloaterValues(action, count)
	total := 0
	for i := 0; i <= index && i < len(values); i++ {
		total += values[i]
	}
	return total
}

func (m *System) AddLocalRecoveryFloater(ctx client.Context, amount int, floaterColor color.RGBA, kind FloaterKind) {
	if ctx.Config.Headless || ctx.World == nil || amount <= 0 {
		return
	}
	now := time.Now()
	actorID := uint32(0)
	if ctx.Session != nil {
		actorID = ctx.Session.AccountID
		if actorID == 0 {
			actorID = ctx.Session.CharID
		}
	}
	m.addDamageFloater(actorID, ctx.World.Player.X, ctx.World.Player.Y, ctx.World.Player.Dir, fmt.Sprintf("%d", amount), floaterColor, kind, now)
}

// actionDamageFloater chooses the number or miss for an attack. Damage is
// white whoever is hit, criticals yellow; misses are red on players and
// white on monsters and NPCs (CGameActor AM_ATTACKED handling).
func actionDamageFloater(action network.ActorActionNotify, target world.Actor, targetOK bool) (string, FloaterKind, color.RGBA) {
	total := action.Damage + action.LeftDamage
	if total > 0 {
		if action.Action == 10 || action.Action == 13 {
			return strconv.Itoa(int(total)), FloaterCritical, damageFloaterCritical
		}
		return strconv.Itoa(int(total)), FloaterNormal, DamageFloaterWhite
	}
	if action.Action == 11 || action.Action == 0 || action.Action == 7 || (action.SkillID > 0 && total == 0) {
		missColor := DamageFloaterWhite
		if targetOK && damageFloaterPlayerJob(int(target.Job)) {
			missColor = damageFloaterRedMiss
		}
		return "miss", FloaterMiss, missColor
	}
	return "", FloaterNormal, color.RGBA{}
}

// damageFloaterPlayerJob is the complement of JT_WARPNPC <= job < JT_2004_JOB_BEGIN.
func damageFloaterPlayerJob(job int) bool {
	return job < scene.ActorJobWarpPortal || job >= 4000
}

// DamageFloaterDuration is each CMsgEffect's lifetime in 24 ms ticks.
func DamageFloaterDuration(kind FloaterKind) time.Duration {
	ticks := 70
	switch kind {
	case FloaterMiss:
		ticks = 80
	case FloaterCombo, FloaterRecoveryHP, FloaterRecoverySP:
		ticks = 120
	}
	return time.Duration(ticks) * 24 * time.Millisecond
}

func (m *System) DrawDamageFloaters(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if len(m.DamageFloaters) == 0 {
		return
	}
	active := m.DamageFloaters[:0]
	for _, floater := range m.DamageFloaters {
		if now.After(floater.Expires) || m.damageFloaterReplaced(floater, now) {
			continue
		}
		active = append(active, floater)
		if now.Before(floater.Starts) {
			continue
		}
		if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(floater.Kind) {
			continue
		}
		// Numbers follow their actor horizontally but keep the height they
		// started at (CMsgEffect::ProcessPosition).
		x, y := float64(floater.X), float64(floater.Y)
		if actor, ok := ctx.World.Actors[floater.ActorID]; ok {
			x, y = scene.ActorRenderPosition(actor, now)
		} else if scene.IsLocalActor(ctx, floater.ActorID) {
			x, y = scene.ActorRenderPosition(ctx.World.Player, now)
		}
		ticks := damageFloaterTicks(floater, now)
		dx, dy, up, zoom, alpha := damageFloaterPlacement(floater.Kind, floater.Dir, ticks)
		if alpha <= 0 {
			continue
		}
		floaterColor := damageFloaterColor(floater.Kind, floater.Color)
		baseZ := scene.TerrainHeightAt(ctx.World, float64(floater.X), float64(floater.Y))
		worldX := scene.CellCenter(x) + dx
		worldY := scene.CellCenter(y) + dy
		screenScale := scene.ActorBillboardScreenScale(projection, worldX, worldY, baseZ)
		// Message effects are render objects, tinted by the lightmap below them.
		light := actors.ActorShadowFactor(ctx.World, worldX-.5, worldY-.5)
		if floater.Kind == FloaterMiss {
			if billboard, ok := m.damageMessageBillboard(ctx, 0, 0); ok {
				sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, baseZ+up, screenScale*zoom, alpha, light, floaterColor)
			}
			continue
		}
		if floater.Kind == FloaterCritical {
			m.drawCriticalBurst(screen, ctx, projection, floater, x, y, baseZ, ticks, light)
		}
		if billboard, ok := m.DamageNumberBillboard(ctx, floater.Text); ok {
			sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, baseZ+up, screenScale*zoom, alpha, light, floaterColor)
			continue
		}
		point := projection.Project(worldX, worldY, baseZ+up)
		render.DrawBitmapTextAtColor(screen, floater.Text, int(point.X)-8, int(point.Y)-40, scene.WithAlpha(floaterColor, alpha))
	}
	m.DamageFloaters = active
}

// damageFloaterReplaced reports a combo total superseded by a newer one: a
// new total deletes the previous ones (CGameActor::DeleteTotalNumber).
func (m *System) damageFloaterReplaced(floater Floater, now time.Time) bool {
	if floater.Kind != FloaterCombo {
		return false
	}
	for _, other := range m.DamageFloaters {
		if other.Kind == FloaterCombo && other.ActorID == floater.ActorID && other.Starts.After(floater.Starts) && !now.Before(other.Starts) {
			return true
		}
	}
	return false
}

// drawCriticalBurst is MET_CRITICAL_EFFECT: msg.spr action 2 tinted 0xaaaaaa,
// drifting like a critical number 3 units higher, shrinking from zoom 3 and
// playing its motions once, two ticks each.
func (m *System) drawCriticalBurst(screen *render.Frame, ctx client.Context, projection scene.Projection, floater Floater, x, y, baseZ, ticks float64, light scene.LightTint) {
	t := math.Max(0, ticks)
	view := m.damageMessageSprite(ctx)
	if view == nil || view.Act == nil || len(view.Act.Actions) <= 2 || len(view.Act.Actions[2].Animations) == 0 {
		return
	}
	motions := len(view.Act.Actions[2].Animations)
	motion := min(int(t/2), motions-1)
	billboard, ok := m.damageMessageBillboard(ctx, 2, motion)
	if !ok {
		return
	}
	mx, mz := damageFloaterDrift(floater.Dir, .5)
	drift := t/3 + 3
	worldX := scene.CellCenter(x) + mx*drift/5
	worldY := scene.CellCenter(y) + mz*drift/5
	up := (12 - 8 + t*(2-t/30) - 4.0/3 + 3) / 5
	zoom := math.Max(3-.24*t*3/5, 1)
	alpha := (255 - 3.45*t) / 255
	if alpha <= 0 {
		return
	}
	screenScale := scene.ActorBillboardScreenScale(projection, worldX, worldY, baseZ)
	sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, baseZ+up, screenScale*zoom, alpha, light,
		color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 255})
}
