package ui

import (
	"fmt"
	"github.com/kivutar/goro/input"

	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	teleportSkillID      = 26
	warpPortalSkillID    = 27
	teleportRandomMap    = "Random"
	teleportSavePointMap = "SavePoint"
	warpPointCancelMap   = "cancel"

	teleportModalWidth   = 260
	teleportModalMinH    = 168
	teleportModalMaxRows = 6
	teleportModalPad     = 14
	teleportModalGap     = 8
	teleportModalRowH    = 20
)

const (
	TeleportSkillID      = teleportSkillID
	WarpPortalSkillID    = warpPortalSkillID
	TeleportRandomMap    = teleportRandomMap
	TeleportSavePointMap = teleportSavePointMap
	WarpPointCancelMap   = warpPointCancelMap
)

type TeleportModal struct {
	Window
	skill    session.Skill
	mapNames []string
	status   string
	scrollY  state.Signal[float32]
	row      int
	ctx      Context
}

type teleportDestination struct {
	label   string
	mapName string
	enabled bool
}

func (m *TeleportModal) OpenWarpPointList(list network.WarpPointList, skill session.Skill) {
	m.skill = skill
	m.mapNames = append(m.mapNames[:0], list.MapNames...)
	m.status = ""
	m.row = initialTeleportRow(m.destinations())
	if m.scrollY != nil {
		m.scrollY.Set(0)
	}
	m.ensureWindow()
	m.Window.open = true
}

func (m *TeleportModal) Reset() {
	m.closeWindow()
	*m = TeleportModal{}
}

func TeleportWarpListBypassesModal(skill session.Skill, list network.WarpPointList) bool {
	if list.SkillID != teleportSkillID {
		return false
	}
	if IsLevelOneTeleportSkill(skill) {
		return true
	}
	for _, name := range list.MapNames {
		if name != "" && name != teleportRandomMap {
			return false
		}
	}
	return true
}

func (m *TeleportModal) Update(ctx Context) bool {
	m.ctx = ctx
	if !m.IsOpen() {
		m.closeWindow()
		return false
	}
	if ctx.Input != nil && (ctx.Input.JustPressed(input.KeyEscape) || ctx.Input.MouseJustPressed(input.MouseButtonRight)) {
		m.cancel(ctx)
		return true
	}
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEnter) {
		m.selectCurrent(ctx)
		return true
	}
	m.openWindow(ctx)
	if m.Window.Update(ctx) {
		m.Publish(ctx)
		return true
	}
	m.Publish(ctx)
	return true
}

func (m *TeleportModal) cancel(ctx Context) {
	if m.skill.ID == warpPortalSkillID && ctx.Network != nil {
		if err := ctx.Network.SendSelectWarpPoint(uint16(warpPortalSkillID), warpPointCancelMap); err != nil {
			m.status = fmt.Sprintf("Cancel failed: %v", err)
			m.refresh(ctx)
			return
		}
	}
	m.closeWindow()
}

func (m *TeleportModal) selectWarpPoint(ctx Context, mapName string) {
	if ctx.Network == nil {
		m.status = "Teleport failed: not connected"
		m.refresh(ctx)
		return
	}
	skillID := uint16(teleportSkillID)
	if m.skill.ID != 0 {
		skillID = m.skill.ID
	}
	if err := ctx.Network.SendSelectWarpPoint(skillID, mapName); err != nil {
		m.status = fmt.Sprintf("Teleport failed: %v", err)
		m.refresh(ctx)
		return
	}
	m.closeWindow()
}

func (m *TeleportModal) selectCurrent(ctx Context) {
	destinations := m.destinations()
	if m.row < 0 || m.row >= len(destinations) || !destinations[m.row].enabled {
		return
	}
	m.selectWarpPoint(ctx, destinations[m.row].mapName)
}

func (m TeleportModal) savePointEnabled() bool {
	return m.savePointMapName() != ""
}

func (m TeleportModal) randomMapName() string {
	for _, name := range m.mapNames {
		if name == teleportRandomMap {
			return name
		}
	}
	return teleportRandomMap
}

func (m TeleportModal) savePointMapName() string {
	if m.skill.ID == teleportSkillID && m.skill.Level >= 2 && m.hasSavePointChoice() {
		return teleportSavePointMap
	}
	return ""
}

func (m TeleportModal) hasSavePointChoice() bool {
	if m.skill.ID != teleportSkillID || m.skill.Level < 2 {
		return false
	}
	if len(m.mapNames) == 0 {
		return true
	}
	for _, name := range m.mapNames {
		if name != "" && name != teleportRandomMap {
			return true
		}
	}
	return false
}

func IsLevelOneTeleportSkill(skill session.Skill) bool {
	return skill.ID == teleportSkillID && skill.Level <= 1
}

func warpPortalDestinationLabel(mapName string, index int) string {
	if index == 0 {
		return fmt.Sprintf("Save Point: %s", mapName)
	}
	return mapName
}

func (m TeleportModal) Title() string {
	if m.skill.ID == warpPortalSkillID {
		return "Warp Portal"
	}
	return "Teleport"
}

func (m *TeleportModal) ensureWindow() {
	height := m.windowHeight()
	if m.width == 0 {
		m.Window = NewWindow(teleportModalWidth, height)
		return
	}
	m.SetSize(teleportModalWidth, height)
}

func (m *TeleportModal) openWindow(ctx Context) {
	m.ensureWindow()
	if m.content == nil {
		m.Open(ctx, m.widgetTree())
	}
	m.Publish(ctx)
}

func (m *TeleportModal) refresh(ctx Context) {
	m.ensureWindow()
	if !m.IsOpen() {
		return
	}
	m.Window.SetContent(m.widgetTree())
	m.Publish(ctx)
}

func (m *TeleportModal) closeWindow() {
	if m.IsOpen() {
		m.Close()
		m.Publish(m.ctx)
	}
}

func (m *TeleportModal) widgetTree() widget.Widget {
	return Win(
		Title(m.Title()),
		CloseButton(false),
		Size(teleportModalWidth, float32(m.windowHeight())),
		Content(
			primitives.Box(
				rotheme.Text("Choose destination."),
				m.destinationList(),
				m.statusText(),
			).
				Padding(teleportModalPad).
				Gap(teleportModalGap),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabledFn("OK", func() bool {
				destinations := m.destinations()
				return m.row < 0 || m.row >= len(destinations) || !destinations[m.row].enabled
			}, func() {
				m.selectCurrent(m.ctx)
			}),
			rotheme.Button("Cancel", func() {
				m.cancel(m.ctx)
			}),
		),
	)
}

func (m *TeleportModal) destinationList() widget.Widget {
	destinations := m.destinations()
	if m.row >= len(destinations) {
		m.row = initialTeleportRow(destinations)
	}
	lv := listview.New(
		listview.ItemCount(len(destinations)),
		listview.FixedItemHeight(teleportModalRowH),
		listview.ScrollYSignal(m.ensureScrollSignal()),
		listview.SelectionModeOpt(listview.SelectionSingle),
		listview.SelectedIndex(m.row),
		listview.OnSelectionChange(func(index int) {
			m.row = index
		}),
		listview.PainterOpt(rotheme.SelectListPainter{EmptyText: "No destinations."}),
		listview.BuildItem(func(item listview.ItemContext) widget.Widget {
			if item.Index < 0 || item.Index >= len(destinations) {
				return rotheme.SelectListRow("", false, teleportModalRowH)
			}
			destination := destinations[item.Index]
			return rotheme.SelectListRow(trimRunes(destination.label, 32), destination.enabled, teleportModalRowH)
		}),
	)
	lv.SetFocused(true)
	return primitives.Box(
		lv,
	).
		Height(float32(m.destinationListHeight())).
		CrossAlign(primitives.CrossAxisStretch)
}

func (m *TeleportModal) destinations() []teleportDestination {
	if m.skill.ID == warpPortalSkillID {
		destinations := make([]teleportDestination, 0, len(m.mapNames))
		for i, name := range m.mapNames {
			if name == "" {
				continue
			}
			destinations = append(destinations, teleportDestination{
				label:   warpPortalDestinationLabel(name, i),
				mapName: name,
				enabled: true,
			})
		}
		return destinations
	}
	return []teleportDestination{
		{label: "Random", mapName: m.randomMapName(), enabled: true},
		{label: "Save Point", mapName: m.savePointMapName(), enabled: m.savePointEnabled()},
	}
}

func (m *TeleportModal) statusText() widget.Widget {
	if m.status == "" {
		return primitives.Box().Height(0)
	}
	return rotheme.Text(trimRunes(m.status, 34)).Color(widget.RGBA8(204, 48, 48, 255))
}

func (m *TeleportModal) destinationListHeight() int {
	count := len(m.destinations())
	if count < 1 {
		count = 1
	}
	if count > teleportModalMaxRows {
		count = teleportModalMaxRows
	}
	return count * teleportModalRowH
}

func (m *TeleportModal) windowHeight() int {
	height := ROWindowTitleHeight + teleportModalPad*2 + int(rotheme.Default.Typography.TextSize) + teleportModalGap + m.destinationListHeight() + ROWindowFooterHeight
	if m.status != "" {
		height += teleportModalGap + int(rotheme.Default.Typography.TextSize)
	}
	return maxInt(teleportModalMinH, height)
}

func (m *TeleportModal) ensureScrollSignal() state.Signal[float32] {
	if m.scrollY == nil {
		m.scrollY = state.NewSignal[float32](0)
	}
	return m.scrollY
}

func initialTeleportRow(destinations []teleportDestination) int {
	for i, destination := range destinations {
		if destination.enabled {
			return i
		}
	}
	return -1
}
