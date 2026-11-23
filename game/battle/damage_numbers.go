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

var (
	DamageFloaterWhite  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterYellow = color.RGBA{R: 230, G: 230, B: 38, A: 255}
	damageFloaterRed    = color.RGBA{R: 255, G: 64, B: 64, A: 255}
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

func composeDamageNumberBillboard(view *scene.SpriteView, text string) (*scene.Billboard, bool) {
	const digitGap = 2
	digits := make([]*scene.Billboard, 0, len(text))
	totalWidth := 0
	maxHeight := 0
	for _, ch := range text {
		digit := int(ch - '0')
		billboard, ok := damageDigitBillboard(view, digit)
		if !ok {
			return nil, false
		}
		bounds := billboard.Image.Bounds()
		if len(digits) > 0 {
			totalWidth += digitGap
		}
		totalWidth += bounds.Dx()
		maxHeight = scene.MaxInt(maxHeight, bounds.Dy())
		digits = append(digits, billboard)
	}
	if totalWidth <= 0 || maxHeight <= 0 {
		return nil, false
	}
	target := render.NewImage(totalWidth, maxHeight)
	x := 0
	for index, digit := range digits {
		if index > 0 {
			x += digitGap
		}
		bounds := digit.Image.Bounds()
		var opts render.DrawImageOptions
		opts.GeoM.Translate(float64(x), float64((maxHeight-bounds.Dy())/2))
		opts.Filter = scene.SpriteCompositionFilter()
		target.DrawImage(digit.Image, &opts)
		x += bounds.Dx()
	}
	return &scene.Billboard{
		Image:   target,
		AnchorX: float64(totalWidth) * 0.5,
		AnchorY: float64(maxHeight) * 0.5,
	}, true
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

func damageFloaterPlacement(kind FloaterKind, progress float64) (dx, dy, zLift, scale, alpha float64) {
	progress = scene.ClampFloat(progress, 0, 1)
	alpha = 1 - progress
	switch kind {
	case FloaterCombo:
		scale = math.Min(progress, 0.05) * 70
		zLift = 7.0 + progress
	case FloaterRecoveryHP, FloaterRecoverySP:
		scale = math.Max((1-progress*2)*3, 0.8)
		zLift = 2.0
		if progress >= 0.4 {
			zLift += (progress - 0.4) * 5.0
		}
	case FloaterMiss:
		scale = 0.5
		zLift = 3.5 + progress*7.0
	default:
		scale = math.Max((1-progress)*4, 0)
		arc := math.Sin(-math.Pi/2+math.Pi*(0.5+progress*1.5)) * 5.0
		dx = progress * 4.0
		zLift = 2.0 + arc
	}
	return dx, dy, zLift, scale, alpha
}

func damageFloaterColor(kind FloaterKind, fallback color.RGBA) color.RGBA {
	switch kind {
	case FloaterCritical:
		return damageFloaterYellow
	case FloaterCombo:
		return damageFloaterYellow
	case FloaterIncoming:
		return damageFloaterRed
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

func damageFloaterProgress(floater Floater, now time.Time) float64 {
	duration := floater.Duration
	if duration <= 0 {
		duration = floater.Expires.Sub(floater.Starts)
	}
	if duration <= 0 {
		return 1
	}
	return float64(now.Sub(floater.Starts)) / float64(duration)
}

type Floater struct {
	ActorID  uint32
	X        int
	Y        int
	Text     string
	Color    color.RGBA
	Kind     FloaterKind
	Starts   time.Time
	Expires  time.Time
	Duration time.Duration
}

type FloaterKind int

const (
	FloaterNormal FloaterKind = iota
	FloaterCritical
	FloaterIncoming
	FloaterRecoveryHP
	FloaterRecoverySP
	FloaterMiss
	FloaterCombo
)

func (m *System) AddActionDamageFloaters(ctx client.Context, action network.ActorActionNotify, target world.Actor, targetOK, targetLocal, sourceLocal bool, x, y int, hitAt time.Time) {
	if ctx.Config.Headless {
		return
	}
	text, kind, floaterColor := actionDamageFloater(action, targetLocal, sourceLocal)
	if text == "" {
		return
	}
	if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(kind) {
		return
	}
	count := scene.ActionVisualHitCount(action)
	total := action.Damage + action.LeftDamage
	if count <= 1 || total <= 0 {
		m.AddDamageFloater(action.TargetID, x, y, text, floaterColor, kind, hitAt)
		return
	}
	values := actionDamageFloaterValues(action, count)
	combo := actionShowsComboDamageFloaters(action, target, targetOK, targetLocal)
	for i := 0; i < count; i++ {
		starts := hitAt.Add(scene.MultiHitDelay * time.Duration(i))
		m.AddDamageFloater(action.TargetID, x, y, strconv.Itoa(values[i]), floaterColor, kind, starts)
		if combo {
			m.addComboDamageFloater(action.TargetID, x, y, actionComboDamageFloaterValue(action, count, i), starts, i+1 == count)
		}
	}
}

func damageFloaterHiddenInSiege(kind FloaterKind) bool {
	switch kind {
	case FloaterNormal, FloaterCritical, FloaterIncoming, FloaterCombo:
		return true
	default:
		return false
	}
}

func (m *System) AddDamageFloater(actorID uint32, x, y int, text string, floaterColor color.RGBA, kind FloaterKind, starts time.Time) {
	duration := DamageFloaterDuration(kind)
	m.addDamageFloaterWithTiming(actorID, x, y, text, floaterColor, kind, starts, duration, duration)
}

func (m *System) addComboDamageFloater(actorID uint32, x, y int, value int, starts time.Time, final bool) {
	if value <= 0 {
		return
	}
	visible := DamageFloaterComboTransientDuration()
	if final {
		visible = DamageFloaterDuration(FloaterCombo)
	}
	m.addDamageFloaterWithTiming(actorID, x, y, strconv.Itoa(value), damageFloaterYellow, FloaterCombo, starts, visible, DamageFloaterDuration(FloaterCombo))
}

func (m *System) addDamageFloaterWithTiming(actorID uint32, x, y int, text string, floaterColor color.RGBA, kind FloaterKind, starts time.Time, visible, duration time.Duration) {
	if visible <= 0 {
		visible = DamageFloaterDuration(kind)
	}
	if duration <= 0 {
		duration = visible
	}
	m.DamageFloaters = append(m.DamageFloaters, Floater{
		ActorID:  actorID,
		X:        x,
		Y:        y,
		Text:     text,
		Color:    floaterColor,
		Kind:     kind,
		Starts:   starts,
		Expires:  starts.Add(visible),
		Duration: duration,
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

func actionShowsComboDamageFloaters(action network.ActorActionNotify, target world.Actor, targetOK, targetLocal bool) bool {
	if !targetOK || targetLocal || action.Damage <= 0 || scene.ActionVisualHitCount(action) <= 1 {
		return false
	}
	if action.SkillID != 0 {
		return !scene.ActorIsPlayerObject(target)
	}
	switch action.Action {
	case 8, 9, 13:
		return actors.ActorHasMobObjectType(target)
	default:
		return false
	}
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
	m.DamageFloaters = append(m.DamageFloaters, Floater{
		ActorID:  actorID,
		X:        ctx.World.Player.X,
		Y:        ctx.World.Player.Y,
		Text:     fmt.Sprintf("%d", amount),
		Color:    floaterColor,
		Kind:     kind,
		Starts:   now,
		Expires:  now.Add(DamageFloaterDuration(kind)),
		Duration: DamageFloaterDuration(kind),
	})
}

func actionDamageFloater(action network.ActorActionNotify, targetLocal, sourceLocal bool) (string, FloaterKind, color.RGBA) {
	total := action.Damage + action.LeftDamage
	if total > 0 {
		if action.Action == 10 || action.Action == 13 {
			return strconv.Itoa(int(total)), FloaterCritical, damageFloaterYellow
		}
		if targetLocal && !sourceLocal {
			return strconv.Itoa(int(total)), FloaterIncoming, damageFloaterRed
		}
		return strconv.Itoa(int(total)), FloaterNormal, DamageFloaterWhite
	}
	if action.Action == 11 {
		return "miss", FloaterMiss, DamageFloaterWhite
	}
	if action.Action == 0 || action.Action == 7 || (action.SkillID > 0 && total == 0) {
		return "miss", FloaterMiss, DamageFloaterWhite
	}
	return "", FloaterNormal, color.RGBA{}
}

func DamageFloaterDuration(kind FloaterKind) time.Duration {
	switch kind {
	case FloaterMiss:
		return 800 * time.Millisecond
	case FloaterCombo:
		return 3000 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
}

func DamageFloaterComboTransientDuration() time.Duration {
	const robrowserComboTransient = 450 * time.Millisecond
	if scene.MultiHitDelay < robrowserComboTransient {
		return scene.MultiHitDelay
	}
	return robrowserComboTransient
}

func (m *System) DrawDamageFloaters(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if len(m.DamageFloaters) == 0 {
		return
	}
	active := m.DamageFloaters[:0]
	for _, floater := range m.DamageFloaters {
		if now.After(floater.Expires) {
			continue
		}
		active = append(active, floater)
		if now.Before(floater.Starts) {
			continue
		}
		if ctx.World != nil && ctx.World.MapProperty.IsSiege() && damageFloaterHiddenInSiege(floater.Kind) {
			continue
		}
		x, y := float64(floater.X), float64(floater.Y)
		if actor, ok := ctx.World.Actors[floater.ActorID]; ok {
			x, y = scene.ActorRenderPosition(actor, now)
		} else if scene.IsLocalActor(ctx, floater.ActorID) {
			x, y = scene.ActorRenderPosition(ctx.World.Player, now)
		}
		progress := damageFloaterProgress(floater, now)
		dx, dy, zLift, scale, alpha := damageFloaterPlacement(floater.Kind, progress)
		floaterColor := damageFloaterColor(floater.Kind, floater.Color)
		terrainZ := scene.TerrainHeightAt(ctx.World, x, y)
		worldX := scene.CellCenter(x) + dx
		worldY := scene.CellCenter(y) + dy
		screenScale := scene.ActorBillboardScreenScale(projection, worldX, worldY, terrainZ)
		if floater.Kind == FloaterMiss {
			if billboard, ok := m.damageMessageBillboard(ctx, 0, 0); ok {
				sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, scene.FullLight, floaterColor)
				continue
			}
		}
		if floater.Kind == FloaterCritical {
			if billboard, ok := m.damageMessageBillboard(ctx, 2, 0); ok {
				sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift+0.05, screenScale*scale*0.6, alpha, scene.FullLight, color.RGBA{R: 168, G: 168, B: 168, A: 255})
			}
		}
		if billboard, ok := m.DamageNumberBillboard(ctx, floater.Text); ok {
			sprites.DrawOverlay3D(screen, projection, billboard, worldX, worldY, terrainZ+zLift, screenScale*scale, alpha, scene.FullLight, floaterColor)
			continue
		}
		point := projection.Project(worldX, worldY, terrainZ+zLift)
		render.DrawBitmapTextAtColor(screen, floater.Text, int(point.X)-8, int(point.Y)-40, scene.WithAlpha(floaterColor, alpha))
	}
	m.DamageFloaters = active
}
