package game

import (
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
)

const (
	damageNumberSPR = "data\\sprite\\이팩트\\숫자.spr"
	damageNumberACT = "data\\sprite\\이팩트\\숫자.act"
	damageMsgSPR    = "data\\sprite\\이팩트\\msg.spr"
	damageMsgACT    = "data\\sprite\\이팩트\\msg.act"
)

var (
	damageFloaterWhite  = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	damageFloaterYellow = color.RGBA{R: 230, G: 230, B: 38, A: 255}
	damageFloaterRed    = color.RGBA{R: 255, G: 64, B: 64, A: 255}
)

func (m *WorldMode) damageNumberSprite(ctx client.Context) *spriteView {
	if m.damageNumberView != nil || m.damageNumberMiss || ctx.Resources == nil {
		return m.damageNumberView
	}
	view, status := loadSpriteView(ctx.Resources,
		[]string{damageNumberACT, strings.ReplaceAll(damageNumberACT, "\\", "/")},
		[]string{damageNumberSPR, strings.ReplaceAll(damageNumberSPR, "\\", "/")},
		nil,
		"damage number",
	)
	if view == nil {
		m.damageNumberMiss = true
		glog.Warnf("damage number sprite unavailable: %s", status)
		return nil
	}
	m.damageNumberView = view
	glog.Debugf("damage number sprite resources %s", status)
	return view
}

func (m *WorldMode) damageNumberBillboard(ctx client.Context, text string) (*spriteBillboard, bool) {
	if text == "" || !allASCIIDigits(text) {
		return nil, false
	}
	if m.damageNumbers == nil {
		m.damageNumbers = make(map[string]*spriteBillboard)
	}
	if billboard, ok := m.damageNumbers[text]; ok {
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
	m.damageNumbers[text] = billboard
	return billboard, true
}

func (m *WorldMode) damageMessageSprite(ctx client.Context) *spriteView {
	if m.damageMsgView != nil || m.damageMsgMiss || ctx.Resources == nil {
		return m.damageMsgView
	}
	view, status := loadSpriteView(ctx.Resources,
		[]string{damageMsgACT, strings.ReplaceAll(damageMsgACT, "\\", "/")},
		[]string{damageMsgSPR, strings.ReplaceAll(damageMsgSPR, "\\", "/")},
		nil,
		"damage message",
	)
	if view == nil {
		m.damageMsgMiss = true
		glog.Warnf("damage message sprite unavailable: %s", status)
		return nil
	}
	m.damageMsgView = view
	glog.Debugf("damage message sprite resources %s", status)
	return view
}

func (m *WorldMode) damageMessageBillboard(ctx client.Context, actionIndex, motion int) (*spriteBillboard, bool) {
	view := m.damageMessageSprite(ctx)
	if view == nil || view.act == nil {
		return nil, false
	}
	if actionIndex < 0 || actionIndex >= len(view.act.Actions) {
		return nil, false
	}
	action := view.act.Actions[actionIndex]
	if motion < 0 || motion >= len(action.Animations) {
		return nil, false
	}
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeSingleSpriteBillboard(view, action.Animations[motion])
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
	return billboard, true
}

func composeDamageNumberBillboard(view *spriteView, text string) (*spriteBillboard, bool) {
	const digitGap = 2
	digits := make([]*spriteBillboard, 0, len(text))
	totalWidth := 0
	maxHeight := 0
	for _, ch := range text {
		digit := int(ch - '0')
		billboard, ok := damageDigitBillboard(view, digit)
		if !ok {
			return nil, false
		}
		bounds := billboard.image.Bounds()
		if len(digits) > 0 {
			totalWidth += digitGap
		}
		totalWidth += bounds.Dx()
		maxHeight = maxInt(maxHeight, bounds.Dy())
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
		bounds := digit.image.Bounds()
		var opts render.DrawImageOptions
		opts.GeoM.Translate(float64(x), float64((maxHeight-bounds.Dy())/2))
		opts.Filter = spriteCompositionFilter()
		target.DrawImage(digit.image, &opts)
		x += bounds.Dx()
	}
	return &spriteBillboard{
		image:   target,
		anchorX: float64(totalWidth) * 0.5,
		anchorY: float64(maxHeight) * 0.5,
	}, true
}

func damageDigitBillboard(view *spriteView, digit int) (*spriteBillboard, bool) {
	if view == nil || view.act == nil || len(view.act.Actions) == 0 || digit < 0 || digit > 9 {
		return nil, false
	}
	action := view.act.Actions[0]
	if digit >= len(action.Animations) {
		return nil, false
	}
	key := singleSpriteBillboardKey{actionIndex: 0, motion: digit}
	if billboard, ok := view.billboards[key]; ok {
		return billboard, true
	}
	billboard, ok := composeSingleSpriteBillboard(view, action.Animations[digit])
	if !ok {
		return nil, false
	}
	view.billboards[key] = billboard
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

func damageFloaterPlacement(kind damageFloaterKind, progress float64) (dx, dy, zLift, scale, alpha float64) {
	progress = clampFloat(progress, 0, 1)
	alpha = 1 - progress
	switch kind {
	case damageFloaterRecoveryHP, damageFloaterRecoverySP:
		scale = math.Max((1-progress*2)*3, 0.8)
		zLift = 2.0
		if progress >= 0.4 {
			zLift += (progress - 0.4) * 3.0
		}
	case damageFloaterMiss:
		scale = 0.7
		zLift = 3.0 + progress*3.0
	default:
		scale = math.Max((1-progress)*4, 0.35)
		arc := math.Sin(-math.Pi/2+math.Pi*(0.5+progress*1.5)) * 1.2
		dx = progress * 0.35
		dy = -progress * 0.35
		zLift = 2.0 + arc
	}
	return dx, dy, zLift, scale, alpha
}

func damageFloaterColor(kind damageFloaterKind, fallback color.RGBA) color.RGBA {
	switch kind {
	case damageFloaterCritical:
		return damageFloaterYellow
	case damageFloaterIncoming:
		return damageFloaterRed
	case damageFloaterRecoveryHP:
		return color.RGBA{R: 0, G: 255, B: 0, A: 255}
	case damageFloaterRecoverySP:
		return color.RGBA{R: 0, G: 0, B: 255, A: 255}
	default:
		if fallback.A != 0 {
			return fallback
		}
		return damageFloaterWhite
	}
}

func withAlpha(c color.RGBA, alpha float64) color.RGBA {
	alpha = clampFloat(alpha, 0, 1)
	c.A = uint8(math.Round(float64(c.A) * alpha))
	return c
}

func damageFloaterProgress(floater damageFloater, now time.Time) float64 {
	duration := floater.expires.Sub(floater.starts)
	if duration <= 0 {
		return 1
	}
	return float64(now.Sub(floater.starts)) / float64(duration)
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
