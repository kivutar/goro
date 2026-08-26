package ui

import (
	"strings"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
)

const playerContextMenuWidth = 118

type PlayerContextActionKind uint8

const (
	PlayerContextActionNone PlayerContextActionKind = iota
	PlayerContextActionAddFriend
	PlayerContextActionInviteParty
	PlayerContextActionInviteGuild
	PlayerContextActionTrade
	PlayerContextActionSeeEquipment
	PlayerContextActionAdopt
	PlayerContextActionGuildAlliance
	PlayerContextActionGuildHostility
)

type PlayerContextAction struct {
	Kind    PlayerContextActionKind
	ActorID uint32
	Name    string
}

type PlayerContextMenu struct {
	Window
	actorID uint32
	name    string
	options PlayerContextOptions
	action  PlayerContextAction
}

type PlayerContextOptions struct {
	CanAddFriend       bool
	CanInviteParty     bool
	CanInviteGuild     bool
	CanAdopt           bool
	CanRequestAlliance bool
	CanDeclareHostile  bool
}

func (m *PlayerContextMenu) Open(ctx Context, x, y int, actorID uint32, name string, options PlayerContextOptions) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	m.EnsureWindow(playerContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	m.actorID = actorID
	m.name = name
	m.options = options
	screenW, screenH := ctx.ScreenSize()
	height := m.height()
	m.SetSize(playerContextMenuWidth, height)
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, screenW-playerContextMenuWidth-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	m.OpenAt(x, y, m.widgetTree(ctx))
	m.Publish(ctx)
}

func (m *PlayerContextMenu) Update(ctx Context) bool {
	m.EnsureWindow(playerContextMenuWidth, m.height())
	m.titleHeight = 0
	m.ctx = ctx
	if !m.IsOpen() {
		return false
	}
	if ctx.Input != nil {
		inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, m.x, m.y, playerContextMenuWidth, m.height())
		if ctx.Input.JustPressed(input.KeyEscape) || (!inside && (ctx.Input.MouseJustPressed(input.MouseButtonLeft) || ctx.Input.MouseJustPressed(input.MouseButtonRight))) {
			m.Close()
			return true
		}
	}
	consumed := m.Window.Update(ctx)
	m.Publish(ctx)
	return consumed
}

func (m *PlayerContextMenu) PopAction() PlayerContextAction {
	action := m.action
	m.action = PlayerContextAction{}
	return action
}

func (m *PlayerContextMenu) widgetTree(ctx Context) widget.Widget {
	rows := []widget.Widget{
		contextMenuItem("Trade", func() {
			m.action = PlayerContextAction{Kind: PlayerContextActionTrade, ActorID: m.actorID, Name: m.name}
			m.Close()
		}),
		contextMenuItem("See equipment", func() {
			m.action = PlayerContextAction{Kind: PlayerContextActionSeeEquipment, ActorID: m.actorID, Name: m.name}
			m.Close()
		}),
	}
	if m.options.CanAddFriend {
		rows = append(rows,
			contextMenuItem("Add Friend", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionAddFriend, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	if m.options.CanInviteParty {
		rows = append(rows,
			contextMenuItem("Invite", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionInviteParty, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	if m.options.CanInviteGuild {
		rows = append(rows,
			contextMenuItem("Invite Guild", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionInviteGuild, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	if m.options.CanAdopt {
		rows = append(rows,
			contextMenuItem("Adopt as Baby", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionAdopt, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	if m.options.CanRequestAlliance {
		rows = append(rows,
			contextMenuItem("Request Alliance", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionGuildAlliance, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	if m.options.CanDeclareHostile {
		rows = append(rows,
			contextMenuItem("Declare Hostility", func() {
				m.action = PlayerContextAction{Kind: PlayerContextActionGuildHostility, ActorID: m.actorID, Name: m.name}
				m.Close()
			}),
		)
	}
	return contextMenu(playerContextMenuWidth, rows...)
}

func (m *PlayerContextMenu) height() int {
	rows := 2
	if m.options.CanAddFriend {
		rows++
	}
	if m.options.CanInviteParty {
		rows++
	}
	if m.options.CanInviteGuild {
		rows++
	}
	if m.options.CanAdopt {
		rows++
	}
	if m.options.CanRequestAlliance {
		rows++
	}
	if m.options.CanDeclareHostile {
		rows++
	}
	return contextMenuHeight(rows)
}
