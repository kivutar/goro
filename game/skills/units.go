package skills

import (
	"math"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type skillUnitModel struct {
	UnitID           uint16
	X                int
	Y                int
	ModelPath        string
	ModelFallbacks   []string
	TriggerEffectID  int
	Scale            float64
	FixedFrame       int
	HasFixedFrame    bool
	AnimationStarted time.Time
}

func skillUnitModelSpec(unitID uint16) (db.SkillUnitModelSpec, bool) {
	spec, ok := db.SkillUnitModels[unitID]
	return spec, ok
}

func skillUnitModelFromEntry(entry network.SkillUnitEntry, spec db.SkillUnitModelSpec) skillUnitModel {
	return skillUnitModel{
		UnitID:          entry.UnitID,
		X:               int(entry.X),
		Y:               int(entry.Y),
		ModelPath:       spec.ModelPath,
		ModelFallbacks:  append([]string(nil), spec.FallbackModelPaths...),
		TriggerEffectID: spec.TriggerEffectID,
		Scale:           spec.Scale,
		FixedFrame:      spec.FixedFrame,
		HasFixedFrame:   spec.HasFixedFrame,
	}
}

func (m *System) ApplySkillUnitModelEntry(entry network.SkillUnitEntry) bool {
	spec, ok := skillUnitModelSpec(entry.UnitID)
	if !ok {
		return false
	}
	unit := skillUnitModelFromEntry(entry, spec)
	if entry.Visible {
		if m.Units == nil {
			m.Units = make(map[uint32]skillUnitModel)
		}
		m.Units[entry.ID] = unit
		if m.HiddenUnits != nil {
			delete(m.HiddenUnits, entry.ID)
		}
		glog.Debugf("skill unit model unit=%d id=%d creator=%d cell=%d,%d model=%s", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, unit.ModelPath)
		return true
	}
	if m.HiddenUnits == nil {
		m.HiddenUnits = make(map[uint32]skillUnitModel)
	}
	m.HiddenUnits[entry.ID] = unit
	if m.Units != nil {
		delete(m.Units, entry.ID)
	}
	glog.Debugf("hidden skill unit model unit=%d id=%d creator=%d cell=%d,%d model=%s", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, unit.ModelPath)
	return true
}

func (m *System) ApplySkillUnitUpdate(update network.SkillUnitUpdate) {
	if update.ID == 0 || m.HiddenUnits == nil {
		return
	}
	unit, ok := m.HiddenUnits[update.ID]
	if !ok {
		return
	}
	delete(m.HiddenUnits, update.ID)
	if m.Units == nil {
		m.Units = make(map[uint32]skillUnitModel)
	}
	m.Units[update.ID] = unit
	glog.Debugf("hidden skill unit model revealed id=%d unit=%d cell=%d,%d model=%s", update.ID, unit.UnitID, unit.X, unit.Y, unit.ModelPath)
}

func (m *System) ApplyUsedTrapLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.Type != 0 || look.ID == 0 || (look.Value != uint32(db.SkillUnitUsedTraps) && look.Value != 141) {
		return false
	}
	if look.Value == 141 {
		unit, ok := m.Units[look.ID]
		if !ok {
			unit, ok = m.HiddenUnits[look.ID]
		}
		if !ok || unit.UnitID != 147 { // Land Mine becomes an active Fire Pillar.
			return false
		}
	}
	unit, ok := m.removeSkillUnitModel(look.ID)
	if !ok {
		return false
	}
	m.bindings.Effects.RemoveUnit(look.ID)
	if look.Value == 141 {
		unit.TriggerEffectID = effects.FirePillarBomb
	}
	if unit.TriggerEffectID > 0 {
		now := time.Now()
		if m.bindings.Effects.AddCellLifetime(ctx, unit.TriggerEffectID, 0, unit.X, unit.Y, now, 0, false) {
			glog.Debugf("skill unit trap triggered id=%d unit=%d cell=%d,%d effect=%d", look.ID, unit.UnitID, unit.X, unit.Y, unit.TriggerEffectID)
		}
	}
	return true
}

func (m *System) removeSkillUnitModel(id uint32) (skillUnitModel, bool) {
	if m.Units != nil {
		if unit, ok := m.Units[id]; ok {
			delete(m.Units, id)
			return unit, true
		}
	}
	if m.HiddenUnits != nil {
		if unit, ok := m.HiddenUnits[id]; ok {
			delete(m.HiddenUnits, id)
			return unit, true
		}
	}
	return skillUnitModel{}, false
}

func (m *System) RemoveSkillUnitModelOnly(id uint32) bool {
	removed := false
	if m.Units != nil {
		if _, ok := m.Units[id]; ok {
			delete(m.Units, id)
			removed = true
		}
	}
	if m.HiddenUnits != nil {
		if _, ok := m.HiddenUnits[id]; ok {
			delete(m.HiddenUnits, id)
			removed = true
		}
	}
	return removed
}

func (m *System) SkillUnitModelCell(id uint32) (int, int, bool) {
	if m.Units != nil {
		if unit, ok := m.Units[id]; ok {
			return unit.X, unit.Y, true
		}
	}
	if m.HiddenUnits != nil {
		if unit, ok := m.HiddenUnits[id]; ok {
			return unit.X, unit.Y, true
		}
	}
	return 0, 0, false
}

func (m *System) ClickedSkillTarget(ctx client.Context, projection scene.Projection, skill session.Skill, mouseX, mouseY int, now time.Time) (worldstate.Actor, bool) {
	if ctx.World == nil {
		return worldstate.Actor{}, false
	}
	if skill.Type&SkillTargetTrap == 0 {
		if m.bindings.ClickActor == nil {
			return worldstate.Actor{}, false
		}
		return m.bindings.ClickActor(ctx, projection, skill, mouseX, mouseY, now)
	}
	var best worldstate.Actor
	bestDistance := math.Inf(1)
	for id, unit := range m.Units {
		x, y := float64(unit.X), float64(unit.Y)
		point, ok := projection.ProjectPoint(scene.CellCenter(x), scene.CellCenter(y), scene.TerrainHeightAt(ctx.World, x, y))
		if !ok {
			continue
		}
		dx, dy := float64(point.X)-float64(mouseX), float64(point.Y)-float64(mouseY)
		distance := dx*dx + dy*dy
		if distance <= 20*20 && (distance < bestDistance || distance == bestDistance && id < best.ID) {
			best, bestDistance = worldstate.Actor{ID: id, X: unit.X, Y: unit.Y}, distance
		}
	}
	return best, best.ID != 0
}

func (m *System) ApplyTrapModelLookChange(look network.ActorLookChange) bool {
	if look.Type != 0 || look.Value != 145 {
		return false
	}
	unit, ok := m.Units[look.ID]
	if !ok || unit.UnitID != 145 {
		return false
	}
	// ZC_SKILL_UPDATE reveals Ankle Snare; the following look change closes it.
	unit.AnimationStarted = time.Now()
	m.Units[look.ID] = unit
	return true
}

func (m *System) DrawSkillUnitRSMModels(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time) {
	if screen == nil || ctx.World == nil || ctx.World.GND == nil || ctx.Resources == nil || len(m.Units) == 0 {
		return
	}
	radius := maps.RSMRenderRadius() * 2
	for id, unit := range m.Units {
		worldX := scene.CellCenter(float64(unit.X))
		worldY := scene.CellCenter(float64(unit.Y))
		if math.Abs(worldX-projection.PlayerX) > radius || math.Abs(worldY-projection.PlayerY) > radius {
			continue
		}
		rsm := m.bindings.Map.RuntimeRSMModel(ctx.Resources, unit.ModelPath)
		if rsm == nil {
			rsm, unit.ModelPath = m.runtimeSkillUnitFallbackRSMModel(ctx.Resources, unit)
		}
		if rsm == nil {
			continue
		}
		placement := skillUnitRSMPlacement(ctx.World, id, unit)
		frame, _ := SkillUnitRSMFrame(rsm, placement.Model, unit, now)
		m.bindings.Map.DrawAnimatedRSMPlacement(screen, ctx.Resources, ctx.World.RSW, ctx.World.GND, rsm, placement, frame)
	}
}

func (m *System) runtimeSkillUnitFallbackRSMModel(manager *res.Manager, unit skillUnitModel) (*res.RSM, string) {
	for _, modelPath := range unit.ModelFallbacks {
		rsm := m.bindings.Map.RuntimeRSMModel(manager, modelPath)
		if rsm != nil {
			return rsm, modelPath
		}
	}
	return nil, ""
}

func skillUnitRSMPlacement(world *worldstate.World, id uint32, unit skillUnitModel) maps.ModelPlacement {
	baseX := scene.CellCenter(float64(unit.X))
	baseY := scene.CellCenter(float64(unit.Y))
	scale := unit.Scale
	if scale == 0 {
		scale = 1
	}
	return maps.ModelPlacement{
		Index: int(id),
		Model: res.RSWModel{
			Name:     "skill_unit",
			Filename: unit.ModelPath,
			Position: res.RSWVector3{
				Y: float32(scene.TerrainHeightAt(world, float64(unit.X), float64(unit.Y))),
			},
			Scale: res.RSWVector3{X: float32(scale), Y: float32(scale), Z: float32(scale)},
		},
		BaseX: baseX,
		BaseY: baseY,
	}
}

func SkillUnitRSMFrame(rsm *res.RSM, placement res.RSWModel, unit skillUnitModel, now time.Time) (int, bool) {
	if !unit.AnimationStarted.IsZero() {
		last := max(3, 3+(maps.RSMAnimationLength(rsm)-4)/100*100)
		frame := 3 + max(0, int(now.Sub(unit.AnimationStarted)/db.EffectFrameDuration))*100
		return min(frame, last), frame < last
	}
	if unit.HasFixedFrame {
		return unit.FixedFrame, false
	}
	return maps.RSMAnimationFrame(rsm, placement, now)
}
