package game

import (
	"image/color"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func (m *WorldMode) startMapFadeOut(change network.MapChange, now time.Time) {
	m.ui.npcCutin.Clear()
	// A queued cell (or message) belongs to the map where it was selected,
	// even when the server warps us within the same loaded map.
	m.pendingSkill = pendingSkillTarget{}
	m.pendingSkillText = pendingSkillTextTarget{}
	if m.ui.skillTextPrompt.IsOpen() {
		m.ui.skillTextPrompt.Close()
	}
	if m.mapFade.phase == mapFadeHold || m.mapFade.phase == mapFadePrewarm {
		m.mapFade = mapFadeState{
			phase:     mapFadeHold,
			change:    change,
			hasChange: true,
		}
		return
	}
	m.mapFade = mapFadeState{
		phase:     mapFadeOut,
		started:   now,
		change:    change,
		hasChange: true,
	}
}

func (m *WorldMode) startCharacterSelectFadeOut(now time.Time) {
	m.ui.npcCutin.Clear()
	m.mapFade = mapFadeState{
		phase:           mapFadeOut,
		started:         now,
		characterSelect: true,
	}
}

func (m *WorldMode) startMapFadeIn(now time.Time) {
	m.mapFade = mapFadeState{phase: mapFadeIn, started: now}
}

func (m *WorldMode) startMapPrewarm() {
	m.mapFade = mapFadeState{phase: mapFadePrewarm}
}

func (m *WorldMode) recordCoveredMapFrame() {
	switch m.mapFade.phase {
	case mapFadeHold:
		if m.mapFade.coveredFrames < mapFadeHandoffFrames {
			m.mapFade.coveredFrames++
		}
	case mapFadePrewarm:
		if m.mapFade.coveredFrames < mapFadePrewarmFrames {
			m.mapFade.coveredFrames++
		}
	}
}

func (m *WorldMode) advanceMapPrewarm(now time.Time) {
	if m.mapFade.phase == mapFadePrewarm && m.mapFade.coveredFrames >= mapFadePrewarmFrames {
		m.startMapFadeIn(now)
	}
}

func (m *WorldMode) mapFadeElapsed(now time.Time) bool {
	switch m.mapFade.phase {
	case mapFadeOut:
		return now.Sub(m.mapFade.started) >= mapFadeOutDuration
	case mapFadeIn:
		return now.Sub(m.mapFade.started) >= mapFadeInDuration
	default:
		return false
	}
}

func (m *WorldMode) mapFadeAlpha(now time.Time) uint8 {
	switch m.mapFade.phase {
	case mapFadeHold, mapFadePrewarm:
		return 255
	case mapFadeOut:
		if m.mapFade.started.IsZero() {
			return 0
		}
		return clampColor(255 * clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeOutDuration)))
	case mapFadeIn:
		if m.mapFade.started.IsZero() {
			return 0
		}
		return clampColor(255 * (1 - clampUnit(float64(now.Sub(m.mapFade.started))/float64(mapFadeInDuration))))
	default:
		return 0
	}
}

func (m *WorldMode) drawMapFade(screen *render.Frame, now time.Time) {
	alpha := m.mapFadeAlpha(now)
	if alpha == 0 {
		return
	}
	bounds := screen.Bounds()
	render.DrawRect(screen, 0, 0, float64(bounds.Dx()), float64(bounds.Dy()), color.RGBA{A: alpha})
}
