package battle

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
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
	blueMsgSPR      = "data\\sprite\\이팩트\\bluemsg.spr"
)

// Colours passed to CGameActor::Am_Make_Number and AM_MAKE_MISS.
var (
	DamageFloaterWhite      = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterCritical   = color.RGBA{R: 255, G: 255, B: 0, A: 255}   // 0xffffff00
	damageFloaterComboTotal = color.RGBA{R: 255, G: 255, B: 85, A: 255}  // 0xffffff55
	damageFloaterRedMiss    = color.RGBA{R: 255, G: 0, B: 0, A: 255}     // misses on players
	damageFloaterBlue       = color.RGBA{R: 150, G: 150, B: 255, A: 255} // MET_BLUE_EFFECT's SetArgb
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
	return loadMessageSprite(ctx, &m.DamageMsgView, &m.DamageMsgMiss, damageMsgSPR, "damage message")
}

// blueMessageSprite is bluemsg.spr with msg.act (MAKETYPE_BLUE_EFFECT).
func (m *System) blueMessageSprite(ctx client.Context) *scene.SpriteView {
	return loadMessageSprite(ctx, &m.BlueMsgView, &m.BlueMsgMiss, blueMsgSPR, "blue message")
}

func loadMessageSprite(ctx client.Context, cached **scene.SpriteView, missing *bool, spr, label string) *scene.SpriteView {
	if *cached != nil || *missing || ctx.Resources == nil {
		return *cached
	}
	view, status := scene.LoadSpriteView(ctx.Resources,
		[]string{damageMsgACT, strings.ReplaceAll(damageMsgACT, "\\", "/")},
		[]string{spr, strings.ReplaceAll(spr, "\\", "/")},
		nil,
		label,
	)
	if view == nil {
		*missing = true
		glog.Warnf("%s sprite unavailable: %s", label, status)
		return nil
	}
	*cached = view
	glog.Debugf("%s sprite resources %s", label, status)
	return view
}

func (m *System) damageMessageBillboard(ctx client.Context, actionIndex, motion int) (*scene.Billboard, bool) {
	return messageBillboard(m.damageMessageSprite(ctx), actionIndex, motion)
}

func messageBillboard(view *scene.SpriteView, actionIndex, motion int) (*scene.Billboard, bool) {
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
	case FloaterBlue:
		// MET_BLUE_EFFECT: 12 units up plus 18, rising 0.1 per tick; its
		// zoom of 3 is clamped to 2 on the first update.
		alpha = math.Max(0, 210-math.Max(0, (t-90)*8))
		return 0, 0, (30 + .1*t) / 5, 2, alpha / 255
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
		if fallback.A != 0 {
			return fallback
		}
		return damageFloaterComboTotal
	case FloaterBlue:
		return damageFloaterBlue
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
	// FloaterBlue is the bluemsg flash shown with Bash-type totals.
	FloaterBlue
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
	if kind == FloaterMiss {
		m.addDamageFloater(action.TargetID, x, y, dir, text, floaterColor, kind, hitAt)
		return
	}
	for _, hit := range actionDamageHits(action) {
		if hit.value <= 0 {
			continue
		}
		starts := hitAt.Add(hit.delay)
		switch hit.mode {
		case damageHitSingle:
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(hit.value), floaterColor, kind, starts)
		case damageHitCritical:
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(hit.value), damageFloaterCritical, FloaterCritical, starts)
		case damageHitMany:
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(hit.value), DamageFloaterWhite, FloaterNormal, starts)
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(hit.total), damageFloaterComboTotal, FloaterCombo, starts)
		case damageHitMany2, damageHitMany2Dash:
			m.addDamageFloater(action.TargetID, x, y, dir, strconv.Itoa(hit.total), DamageFloaterWhite, FloaterCombo, starts)
			if hit.mode == damageHitMany2 {
				m.addDamageFloater(action.TargetID, x, y, dir, "", damageFloaterBlue, FloaterBlue, starts)
			}
		}
	}
}

// damageHitMode is the message a scheduled hit (WBA) reaches the target with.
type damageHitMode int

const (
	damageHitSingle    damageHitMode = iota // AM_ATTACKED: one number
	damageHitCritical                       // AM_ATTACKED_CRITICAL
	damageHitMany                           // AM_ATTACKED_MANYTIMES: number and yellow total
	damageHitMany2                          // AM_ATTACKED_MANYTIMES2: white total and blue flash
	damageHitMany2Dash                      // AM_ATTACKED_MANYTIMES2DASH: white total
)

type damageHit struct {
	delay time.Duration
	value int
	total int
	mode  damageHitMode
}

const (
	actionSplash                 = 5
	actionSkill                  = 6
	actionAttackMultiple         = 8
	actionAttackMultipleNoMotion = 9
	actionAttackCritical         = 10
	// doubleAttackTerm separates the hits of one action, in milliseconds.
	doubleAttackTerm = 200 * time.Millisecond
)

// actionDamageHits schedules the numbers of an attack as CGameActor::Am_Action
// and Am_Skill queue their hits. Damage is split evenly with integer
// division (RandGenerator adds nothing) and totals are the running sum of
// the shown hits.
func actionDamageHits(action network.ActorActionNotify) []damageHit {
	count := int(action.HitCount)
	if count <= 0 || count > 36 {
		count = 1
	}
	if action.SkillID != 0 {
		return skillDamageHits(action, count)
	}
	if action.Action != actionAttackMultiple && action.Action != actionAttackMultipleNoMotion {
		count = 1
	}
	single := damageHitSingle
	if action.Action == actionAttackCritical || action.Action == 13 {
		single = damageHitCritical
	}
	each := int(action.Damage) / count
	hits := make([]damageHit, 0, count+1)
	add := func(delay time.Duration, value int, many bool) {
		total := value
		if len(hits) > 0 {
			total += hits[len(hits)-1].total
		}
		mode := single
		if many {
			mode = damageHitMany
		}
		hits = append(hits, damageHit{delay: delay, value: value, total: total, mode: mode})
	}
	add(0, each, count > 1)
	if action.LeftDamage > 0 {
		// Dual wielding: the other right-hand hits land together half a term
		// later, the left hand one and three quarter terms after the first.
		for i := 1; i < count; i++ {
			add(doubleAttackTerm/2, each, true)
		}
		add(doubleAttackTerm*7/4, int(action.LeftDamage), count > 1)
		return hits
	}
	for i := 1; i < count; i++ {
		add(doubleAttackTerm*time.Duration(i), each, true)
	}
	return hits
}

func skillDamageHits(action network.ActorActionNotify, count int) []damageHit {
	each := int(action.Damage) / count
	first := damageHit{value: each, total: each, mode: damageHitSingle}
	if count > 1 {
		first.mode = damageHitMany
	}
	if action.Action == actionSplash {
		switch action.SkillID {
		case db.SkillACShower, db.SkillMGNapalmbeat, db.SkillTKStormkick, db.SkillMOCombofinish:
		default:
			first.mode = damageHitSingle
			return []damageHit{first}
		}
	}
	taekwonKick := false
	switch action.SkillID {
	case db.SkillTKDownkick, db.SkillTKTurnkick:
		first.delay = 300 * time.Millisecond
		first.mode = damageHitMany2
		taekwonKick = true
	case db.SkillTKStormkick, db.SkillTKCounter, db.SkillTKJumpkick:
		first.mode = damageHitMany2
		taekwonKick = true
	case db.SkillSMBash, db.SkillKNBowlingbash:
		first.mode = damageHitMany2
	case db.SkillASCBreaker, db.SkillMOExtremityfist:
		first.mode = damageHitCritical
	case db.SkillMOCombofinish, db.SkillChTigerfist, db.SkillNPCBlooddrain:
		first.mode = damageHitMany
	}
	hits := []damageHit{first}
	switch action.Action {
	case actionSkill, actionAttackMultiple, actionAttackMultipleNoMotion:
	default:
		return hits
	}
	delay, total := first.delay, first.total
	for i := 0; i < count-1; i++ {
		if action.SkillID == db.SkillKNPierce {
			delay += 80 * time.Millisecond
		}
		delay += doubleAttackTerm
		total += each
		mode := damageHitMany
		if taekwonKick {
			mode = damageHitMany2Dash
		}
		hits = append(hits, damageHit{delay: delay, value: each, total: total, mode: mode})
	}
	return hits
}

func damageFloaterHiddenInSiege(kind FloaterKind) bool {
	switch kind {
	case FloaterNormal, FloaterCritical, FloaterCombo, FloaterBlue:
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
	case FloaterCombo, FloaterBlue, FloaterRecoveryHP, FloaterRecoverySP:
		ticks = 120
	}
	return time.Duration(ticks) * 24 * time.Millisecond
}

func (m *System) DrawDamageFloaters(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if len(m.DamageFloaters) == 0 {
		return
	}
	active := m.DamageFloaters[:0]
	for i, floater := range m.DamageFloaters {
		if now.After(floater.Expires) || m.damageFloaterReplaced(i, now) {
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
		if floater.Kind == FloaterBlue {
			if billboard, ok := messageBillboard(m.blueMessageSprite(ctx), 2, 0); ok {
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
// new total deletes the previous ones (CGameActor::DeleteTotalNumber). Totals
// starting together are replaced in the order they were queued.
func (m *System) damageFloaterReplaced(index int, now time.Time) bool {
	floater := m.DamageFloaters[index]
	if floater.Kind != FloaterCombo {
		return false
	}
	for j, other := range m.DamageFloaters {
		if other.Kind != FloaterCombo || other.ActorID != floater.ActorID || now.Before(other.Starts) {
			continue
		}
		if other.Starts.After(floater.Starts) || (j > index && other.Starts.Equal(floater.Starts)) {
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
