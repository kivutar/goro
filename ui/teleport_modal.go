package ui

import (
	"fmt"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
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
	teleportModalListPad = 1
	teleportModalFooterH = 42
	teleportModalPad     = 14
	teleportModalGap     = 8
)

const (
	TeleportSkillID      = teleportSkillID
	WarpPortalSkillID    = warpPortalSkillID
	TeleportRandomMap    = teleportRandomMap
	TeleportSavePointMap = teleportSavePointMap
	WarpPointCancelMap   = warpPointCancelMap
)

type TeleportModal struct {
	open     bool
	skill    session.Skill
	mapNames []string
	status   string
	window   WindowState
	scrollY  state.Signal[float32]
	ctx      Context
}

func (m *TeleportModal) OpenWarpPointList(list network.WarpPointList, skill session.Skill) {
	m.open = true
	m.skill = skill
	m.mapNames = append(m.mapNames[:0], list.MapNames...)
	m.status = ""
	m.closeWindow()
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

func (m *TeleportModal) Update(ctx Context, actions GameActions) bool {
	m.ctx = ctx
	if !m.open {
		m.closeWindow()
		return false
	}
	if ctx.Input != nil && (ctx.Input.JustPressed(render.KeyEscape) || ctx.Input.MouseJustPressed(render.MouseButtonRight)) {
		m.cancel(ctx)
		return true
	}
	m.openWindow(ctx, actions)
	if m.window.Update(ctx) {
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
	m.open = false
	m.closeWindow()
}

func (m *TeleportModal) selectWarpPoint(ctx Context, actions GameActions, mapName string) {
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
	if actions != nil && skillID == teleportSkillID {
		actions.AddTeleportEffect(ctx)
	}
	m.open = false
	m.closeWindow()
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

func (m TeleportModal) IsOpen() bool {
	return m.open
}

func (m *TeleportModal) Publish(ctx Context) {
	if ctx.UIManager == nil {
		return
	}
	m.window.Publish(ctx)
}

func (m *TeleportModal) ensureWindow() {
	height := m.windowHeight()
	if m.window.width == 0 {
		m.window = NewWindowState(teleportModalWidth, height)
		m.window.SetCloseOnEscape(false)
		return
	}
	m.window.SetSize(teleportModalWidth, height)
}

func (m *TeleportModal) openWindow(ctx Context, actions GameActions) {
	m.ensureWindow()
	if !m.window.IsOpen() {
		m.window.Open(ctx, m.widgetTree(ctx, actions))
	}
}

func (m *TeleportModal) refresh(ctx Context) {
	m.ensureWindow()
	if !m.window.IsOpen() {
		return
	}
	m.window.SetContent(m.widgetTree(ctx, nil))
	m.Publish(ctx)
}

func (m *TeleportModal) closeWindow() {
	if m.window.IsOpen() {
		m.window.Close()
		m.Publish(m.ctx)
	}
}

func (m *TeleportModal) widgetTree(ctx Context, actions GameActions) widget.Widget {
	return Window(
		Title(m.Title()),
		CloseButton(false),
		Size(teleportModalWidth, float32(m.windowHeight())),
		FooterHeight(teleportModalFooterH),
		Content(
			primitives.Box(
				rotheme.Text("Choose destination."),
				m.destinationList(ctx, actions),
				m.statusText(),
			).
				Padding(teleportModalPad).
				Gap(teleportModalGap),
		),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Cancel", func() {
					m.cancel(m.ctx)
				}).Width(float32(ButtonLabelWidth("Cancel"))),
			),
		),
	)
}

func (m *TeleportModal) destinationList(ctx Context, actions GameActions) widget.Widget {
	rows := m.destinationRows(ctx, actions)
	if len(rows) == 0 {
		return primitives.Box().Height(float32(teleportModalRowHeight()))
	}
	rowH := teleportModalRowHeight()
	return primitives.Box(
		scrollview.New(
			primitives.Box(rows...).
				Gap(teleportModalGap).
				Padding(teleportModalListPad).
				CrossAlign(primitives.CrossAxisStretch),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollYSignal(m.ensureScrollSignal()),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(float32(rowH)),
		),
	).
		Height(float32(m.destinationListHeight())).
		CrossAlign(primitives.CrossAxisStretch)
}

func (m *TeleportModal) destinationRows(ctx Context, actions GameActions) []widget.Widget {
	if m.skill.ID == warpPortalSkillID {
		rows := make([]widget.Widget, 0, len(m.mapNames))
		for i, name := range m.mapNames {
			if name == "" {
				continue
			}
			rows = append(rows, m.destinationButton(ctx, actions, warpPortalDestinationLabel(name, i), name, true))
		}
		return rows
	}
	return []widget.Widget{
		m.destinationButton(ctx, actions, "Random", m.randomMapName(), true),
		m.destinationButton(ctx, actions, "Save Point", m.savePointMapName(), m.savePointEnabled()),
	}
}

func (m *TeleportModal) destinationButton(ctx Context, actions GameActions, label, mapName string, enabled bool) widget.Widget {
	return rotheme.LargeButtonDisabled(label, !enabled, func() {
		m.selectWarpPoint(ctx, actions, mapName)
	})
}

func (m *TeleportModal) statusText() widget.Widget {
	if m.status == "" {
		return primitives.Box().Height(0)
	}
	return rotheme.Text(trimRunes(m.status, 34)).Color(widget.RGBA8(204, 48, 48, 255))
}

func (m *TeleportModal) destinationListHeight() int {
	count := 2
	if m.skill.ID == warpPortalSkillID {
		count = 0
		for _, name := range m.mapNames {
			if name != "" {
				count++
			}
		}
	}
	if count < 1 {
		count = 1
	}
	if count > teleportModalMaxRows {
		count = teleportModalMaxRows
	}
	return count*teleportModalRowHeight() + (count-1)*teleportModalGap + teleportModalListPad*2
}

func teleportModalRowHeight() int {
	return int(rotheme.Default.Typography.TextSize + rotheme.LargeButtonPaddingY*2)
}

func (m *TeleportModal) windowHeight() int {
	height := ROWindowTitleHeight + teleportModalPad*2 + int(rotheme.Default.Typography.TextSize) + teleportModalGap + m.destinationListHeight() + teleportModalFooterH
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
