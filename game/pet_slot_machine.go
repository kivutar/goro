package game

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

const (
	petSlotMachineWidth   = 270
	petSlotMachineHeight  = 260
	petSlotMachineAnchorX = 140
	petSlotMachineAnchorY = 165
)

type petSlotMachinePhase int

const (
	petSlotMachineWaiting petSlotMachinePhase = iota
	petSlotMachineSpinning
	petSlotMachineSuccess
	petSlotMachineFail
)

type petSlotMachineState struct {
	active    bool
	targetID  uint32
	phase     petSlotMachinePhase
	started   time.Time
	attempted bool
	result    bool
}

func (s *petSlotMachineState) setResult(success bool) {
	s.phase = petSlotMachineSpinning
	s.started = time.Now()
	s.result = success
}

func (m *WorldMode) openPetSlotMachine(ctx client.Context, targetID uint32) {
	if ctx.Config.Render.NoUI {
		if ctx.Network != nil {
			if err := ctx.Network.SendTryCaptureMonster(targetID); err != nil {
				glog.Warnf("pet capture send failed target=%d: %v", targetID, err)
			}
		}
		return
	}
	if m.loadPetSlotMachineView(ctx.Resources) == nil {
		if ctx.Network != nil {
			if err := ctx.Network.SendTryCaptureMonster(targetID); err != nil {
				glog.Warnf("pet capture send failed target=%d: %v", targetID, err)
			}
		}
		return
	}
	m.petSlotMachine = petSlotMachineState{
		active:   true,
		targetID: targetID,
		phase:    petSlotMachineWaiting,
		started:  time.Now(),
	}
}

func (m *WorldMode) updatePetSlotMachine(ctx client.Context) bool {
	if !m.petSlotMachine.active || ctx.Input == nil {
		return false
	}
	if ctx.Input.JustPressed(input.KeyEscape) || ctx.Input.MouseJustPressed(input.MouseButtonRight) {
		m.petSlotMachine = petSlotMachineState{}
		return true
	}
	if !ctx.Input.MouseJustPressed(input.MouseButtonLeft) {
		return false
	}
	x, y := petSlotMachineOrigin(ctx)
	if !petSlotMachineContains(ctx.Input.MouseX, ctx.Input.MouseY, x, y) {
		return true
	}
	if m.petSlotMachine.phase != petSlotMachineWaiting || m.petSlotMachine.attempted {
		return true
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Capture failed: not connected.")
		return true
	}
	if err := ctx.Network.SendTryCaptureMonster(m.petSlotMachine.targetID); err != nil {
		m.ui.console.AddErrorMessage("Capture failed.")
		glog.Warnf("pet capture send failed target=%d: %v", m.petSlotMachine.targetID, err)
		return true
	}
	m.petSlotMachine.attempted = true
	glog.Debugf("pet capture try target=%d", m.petSlotMachine.targetID)
	return true
}

func (m *WorldMode) drawPetSlotMachine(screen *render.Frame, ctx client.Context, now time.Time) {
	if !m.petSlotMachine.active {
		return
	}
	view := m.loadPetSlotMachineView(ctx.Resources)
	if view == nil || view.act == nil || len(view.act.Actions) == 0 {
		return
	}
	actionIndex, motion, visible := m.petSlotMachineFrame(view.act, now)
	if !visible {
		return
	}
	if actionIndex < 0 || actionIndex >= len(view.act.Actions) {
		return
	}
	action := view.act.Actions[actionIndex]
	if motion < 0 || motion >= len(action.Animations) {
		return
	}
	key := singleSpriteBillboardKey{actionIndex: actionIndex, motion: motion}
	billboard, ok := view.billboards[key]
	if !ok {
		var baseOK bool
		billboard, baseOK = composeSingleSpriteBillboard(view, action.Animations[motion])
		if !baseOK {
			return
		}
		view.billboards[key] = billboard
	}
	x, y := petSlotMachineOrigin(ctx)
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(x)+petSlotMachineAnchorX-billboard.anchorX, float64(y)+petSlotMachineAnchorY-billboard.anchorY)
	opts.Filter = render.FilterNearest
	opts.Blend = render.BlendSourceOver
	screen.DrawImage(billboard.image, &opts)
}

func (m *WorldMode) petSlotMachineFrame(act *res.ACT, now time.Time) (int, int, bool) {
	phase := m.petSlotMachine.phase
	actionIndex := int(phase)
	if actionIndex < 0 || actionIndex >= len(act.Actions) {
		return 0, 0, false
	}
	action := act.Actions[actionIndex]
	if len(action.Animations) == 0 {
		return 0, 0, false
	}
	delay := float64(action.DelayMS)
	if delay <= 0 {
		delay = 150
	}
	elapsed := float64(now.Sub(m.petSlotMachine.started) / time.Millisecond)
	switch phase {
	case petSlotMachineWaiting:
		motion := int(math.Floor(elapsed/delay*2)) % len(action.Animations)
		return actionIndex, motion, true
	case petSlotMachineSpinning:
		maxFrame := len(action.Animations)
		if m.petSlotMachine.result {
			maxFrame += 7
		} else {
			maxFrame += 3
		}
		motion := int(math.Floor(elapsed / delay))
		if motion >= maxFrame {
			if m.petSlotMachine.result {
				m.petSlotMachine.phase = petSlotMachineSuccess
			} else {
				m.petSlotMachine.phase = petSlotMachineFail
			}
			m.petSlotMachine.started = now
			return m.petSlotMachineFrame(act, now)
		}
		return actionIndex, motion % len(action.Animations), true
	case petSlotMachineSuccess, petSlotMachineFail:
		motion := int(math.Floor(elapsed / delay))
		if motion >= len(action.Animations) {
			if elapsed >= float64(len(action.Animations))*delay+500 {
				m.petSlotMachine = petSlotMachineState{}
			}
			return actionIndex, len(action.Animations) - 1, false
		}
		return actionIndex, motion, true
	default:
		return 0, 0, false
	}
}

func (m *WorldMode) loadPetSlotMachineView(manager *res.Manager) *spriteView {
	if manager == nil || m.slotMachineMiss {
		return nil
	}
	if m.slotMachineView != nil {
		return m.slotMachineView
	}
	view, status := loadSlotMachineSpriteView(manager)
	if view == nil {
		m.slotMachineMiss = true
		glog.Warnf("pet slot machine sprite unavailable: %s", status)
		return nil
	}
	m.slotMachineView = view
	glog.Debugf("pet slot machine sprite resources %s", status)
	return view
}

func loadSlotMachineSpriteView(manager *res.Manager) (*spriteView, string) {
	return loadSpriteView(manager,
		[]string{"data\\sprite\\slotmachine.act", "data/sprite/slotmachine.act"},
		[]string{"data\\sprite\\slotmachine.spr", "data/sprite/slotmachine.spr"},
		nil,
		"pet slotmachine",
	)
}

func petSlotMachineOrigin(ctx client.Context) (int, int) {
	w, h := ctx.ScreenSize()
	return w/2 - petSlotMachineWidth/2, h/2 - petSlotMachineHeight/2
}

func petSlotMachineContains(x, y, rx, ry int) bool {
	return x >= rx && x < rx+petSlotMachineWidth && y >= ry && y < ry+petSlotMachineHeight
}
