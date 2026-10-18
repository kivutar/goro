package ui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	friendsWindowWidth  = 286
	friendsWindowHeight = 256
	friendsRowHeight    = 24
	friendsTabWidth     = 72
	friendsTabHeight    = 24
	friendsListMax      = 40
)

type FriendsWindow struct {
	Window
	snapshot         string
	tab              friendsWindowTab
	action           FriendsWindowAction
	contextMenu      FriendContextMenu
	partyContextMenu PartyContextMenu
}

type friendsWindowTab int

const (
	friendsWindowTabFriends friendsWindowTab = iota
	friendsWindowTabParty
)

type FriendsWindowActionKind int

const (
	FriendsWindowActionNone FriendsWindowActionKind = iota
	FriendsWindowActionPartySettings
	FriendsWindowActionPartyLeave
	FriendsWindowActionFriendWhisper
	FriendsWindowActionFriendDelete
	FriendsWindowActionFriendSettings
	FriendsWindowActionFriendBlockWhisper
	FriendsWindowActionPartyCreate
	FriendsWindowActionPartyInvite
	FriendsWindowActionPartyMemberInfo
	FriendsWindowActionPartyMemberWhisper
	FriendsWindowActionPartyMemberExpel
)

type FriendsWindowAction struct {
	Kind        FriendsWindowActionKind
	Friend      session.Friend
	PartyMember session.PartyMember
}

func (w *FriendsWindow) Toggle(ctx Context) {
	if w.IsOpen() {
		w.Close()
		return
	}
	w.OpenWindow(ctx)
}

func (w *FriendsWindow) OpenWindow(ctx Context) {
	w.openTab(ctx, friendsWindowTabFriends)
}

func (w *FriendsWindow) ToggleFriends(ctx Context) {
	w.toggleTab(ctx, friendsWindowTabFriends)
}

func (w *FriendsWindow) ToggleParty(ctx Context) {
	w.toggleTab(ctx, friendsWindowTabParty)
}

func (w *FriendsWindow) toggleTab(ctx Context, tab friendsWindowTab) {
	if w.IsOpen() && w.tab == tab {
		w.Close()
		return
	}
	w.contextMenu.Close()
	w.partyContextMenu.Close()
	w.openTab(ctx, tab)
}

func (w *FriendsWindow) openTab(ctx Context, tab friendsWindowTab) {
	w.EnsureWindow(friendsWindowWidth, friendsWindowHeight)
	w.ctx = ctx
	w.snapshot = friendsWindowSnapshot(ctx.Session)
	w.tab = tab
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *FriendsWindow) Close() {
	w.contextMenu.Close()
	w.partyContextMenu.Close()
	w.Window.Close()
}

func (w *FriendsWindow) Update(ctx Context) bool {
	w.EnsureWindow(friendsWindowWidth, friendsWindowHeight)
	w.ctx = ctx
	if w.drainContextMenuAction() {
		return true
	}
	if w.contextMenu.Update(ctx) {
		w.drainContextMenuAction()
		return true
	}
	if w.partyContextMenu.Update(ctx) {
		w.drainContextMenuAction()
		return true
	}
	if w.drainContextMenuAction() {
		return true
	}
	if !w.IsOpen() {
		return false
	}
	nextSnapshot := friendsWindowSnapshot(ctx.Session)
	if nextSnapshot != w.snapshot {
		w.snapshot = nextSnapshot
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	if !w.Window.IsOpen() {
		w.contextMenu.Close()
		w.partyContextMenu.Close()
	}
	w.Publish(ctx)
	return consumed
}

func (w *FriendsWindow) drainContextMenuAction() bool {
	if action := w.contextMenu.PopAction(); action.Kind != FriendsWindowActionNone {
		w.action = action
		return true
	}
	if action := w.partyContextMenu.PopAction(); action.Kind != FriendsWindowActionNone {
		w.action = action
		return true
	}
	return false
}

func (w *FriendsWindow) Rebind(ctx Context) {
	w.EnsureWindow(friendsWindowWidth, friendsWindowHeight)
	w.contextMenu.Rebind(ctx)
	w.partyContextMenu.Rebind(ctx)
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.snapshot = friendsWindowSnapshot(ctx.Session)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *FriendsWindow) PopAction() FriendsWindowAction {
	action := w.action
	w.action = FriendsWindowAction{}
	return action
}

func (w *FriendsWindow) widgetTree(ctx Context) widget.Widget {
	friends := sessionFriends(ctx.Session)
	party := sessionParty(ctx.Session)
	title := fmt.Sprintf("Friends (%d/%d)", len(friends), friendsListMax)
	content := w.friendsList(friends)
	footer := w.friendsFooter()
	if w.tab == friendsWindowTabParty {
		title = partyWindowTitle(party)
		content = w.partyList(party)
		footer = w.partyFooter(party)
	}
	return Win(
		Title(title),
		CloseButton(true),
		OnClose(w.Close),
		Size(friendsWindowWidth, friendsWindowHeight),
		Content(
			primitives.Box(
				w.friendsTabs(),
				primitives.Expanded(content),
			).
				CrossAlign(primitives.CrossAxisStretch),
		),
		Footer(footer...),
	)
}

func (w *FriendsWindow) friendsTabs() widget.Widget {
	return primitives.HBox(
		newTabWidget(tabWidgetConfig{
			label:  "Friends",
			active: w.tab == friendsWindowTabFriends,
			width:  friendsTabWidth,
			height: friendsTabHeight,
			onClick: func() {
				w.tab = friendsWindowTabFriends
				w.refresh(w.ctx)
			},
		}),
		newTabWidget(tabWidgetConfig{
			label:  "Party",
			active: w.tab == friendsWindowTabParty,
			width:  friendsTabWidth,
			height: friendsTabHeight,
			onClick: func() {
				w.tab = friendsWindowTabParty
				w.refresh(w.ctx)
			},
		}),
		primitives.Expanded(primitives.Box()),
	).Gap(-1)
}

func (w *FriendsWindow) refresh(ctx Context) {
	w.snapshot = friendsWindowSnapshot(ctx.Session)
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *FriendsWindow) partyFooter(party session.Party) []widget.Widget {
	if !party.Active() {
		return []widget.Widget{
			primitives.Expanded(primitives.Box()),
			rotheme.Button("Create", func() {
				w.action = FriendsWindowAction{Kind: FriendsWindowActionPartyCreate}
			}),
		}
	}
	children := []widget.Widget{
		primitives.Expanded(primitives.Box()),
	}
	if partyCanManageSession(w.ctx.Session) {
		children = append(children, rotheme.Button("Invite", func() {
			w.action = FriendsWindowAction{Kind: FriendsWindowActionPartyInvite}
		}))
	}
	children = append(children,
		rotheme.ButtonDisabled("Settings", !party.Active(), func() {
			w.action = FriendsWindowAction{Kind: FriendsWindowActionPartySettings}
		}),
		rotheme.ButtonDisabled("Leave", !party.Active(), func() {
			w.action = FriendsWindowAction{Kind: FriendsWindowActionPartyLeave}
		}),
	)
	return children
}

func (w *FriendsWindow) friendsFooter() []widget.Widget {
	return []widget.Widget{
		primitives.Expanded(primitives.Box()),
		rotheme.Button("Setup", func() {
			w.action = FriendsWindowAction{Kind: FriendsWindowActionFriendSettings}
		}),
	}
}

func (w *FriendsWindow) friendsList(friends []session.Friend) widget.Widget {
	rows := make([]widget.Widget, 0, maxInt(1, len(friends)))
	if len(friends) == 0 {
		rows = append(rows, emptyFriendsList("No friends"))
	} else {
		for i, friend := range friends {
			rows = append(rows, w.friendRow(friend, i))
		}
	}
	return primitives.Box(rows...).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		CrossAlign(primitives.CrossAxisStretch)
}

func emptyFriendsList(label string) widget.Widget {
	return primitives.Expanded(
		primitives.Box(
			primitives.Expanded(primitives.Box()),
			rotheme.Text(label).
				Color(rotheme.Default.Colors.MutedText).
				Align(primitives.TextAlignCenter),
			primitives.Expanded(primitives.Box()),
		).
			CrossAlign(primitives.CrossAxisStretch),
	)
}

func (w *FriendsWindow) friendRow(friend session.Friend, index int) widget.Widget {
	state := "Offline"
	stateColor := rotheme.Default.Colors.MutedText
	if friend.Online() {
		state = "Online"
		stateColor = Color(GoodTextColor)
	}
	bg := rotheme.Default.Colors.WindowBody
	if index%2 == 0 {
		bg = Color(PanelAltColor)
	}
	name := strings.TrimSpace(friend.Name)
	if name == "" {
		name = "Unknown"
	}
	row := &friendRowWidget{
		friend:     friend,
		name:       name,
		state:      state,
		stateColor: stateColor,
		bg:         bg,
		onWhisper: func(friend session.Friend) {
			w.action = FriendsWindowAction{Kind: FriendsWindowActionFriendWhisper, Friend: friend}
		},
		onMenu: func(friend session.Friend) {
			x, y := w.x+friendsWindowWidth/2, w.y+ROWindowTitleHeight+friendsTabHeight+friendsRowHeight
			if w.ctx.Input != nil {
				x, y = w.ctx.Input.MouseX, w.ctx.Input.MouseY
			}
			w.contextMenu.Open(w.ctx, x, y, friend)
		},
	}
	row.SetVisible(true)
	row.SetEnabled(true)
	return row
}

type friendRowWidget struct {
	widget.WidgetBase
	friend     session.Friend
	name       string
	state      string
	stateColor widget.Color
	bg         widget.Color
	hovered    bool
	onWhisper  func(session.Friend)
	onMenu     func(session.Friend)
}

func (w *friendRowWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(friendsWindowWidth, friendsRowHeight))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *friendRowWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	bg := w.bg
	if w.hovered {
		bg = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(bounds, bg)
	textY := bounds.Min.Y + (bounds.Height()-14)/2
	rotheme.DrawText(canvas, trimRunes(w.name, 24), geometry.NewRect(bounds.Min.X+8, textY, bounds.Width()-88, 14), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, w.state, geometry.NewRect(bounds.Max.X-72, textY, 64, 14), rotheme.Default.Typography.TextSize, w.stateColor, false, widget.TextAlignRight)
}

func (w *friendRowWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		w.hovered = true
		ctx.SetCursor(widget.CursorPointer)
		return true
	case event.MouseLeave:
		w.hovered = false
		ctx.SetCursor(widget.CursorDefault)
		return true
	case event.MousePress:
		switch mouse.Button {
		case event.ButtonLeft:
			if w.onWhisper != nil {
				w.onWhisper(w.friend)
			}
			return true
		case event.ButtonRight:
			if w.onMenu != nil {
				w.onMenu(w.friend)
			}
			return true
		}
	}
	return true
}

func (w *friendRowWidget) Children() []widget.Widget {
	return nil
}

func partyWindowTitle(party session.Party) string {
	if !party.Active() {
		return "Party"
	}
	name := strings.TrimSpace(party.Name)
	if name == "" {
		name = "Party"
	}
	return name
}

func (w *FriendsWindow) partyList(party session.Party) widget.Widget {
	members := party.Members
	rows := make([]widget.Widget, 0, maxInt(1, len(members)))
	if len(members) == 0 {
		rows = append(rows, emptyFriendsList("No party"))
	} else {
		for i, member := range members {
			rows = append(rows, w.partyRow(member, i))
		}
	}
	return primitives.Box(rows...).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *FriendsWindow) partyRow(member session.PartyMember, index int) widget.Widget {
	bg := rotheme.Default.Colors.WindowBody
	if index%2 == 0 {
		bg = Color(PanelAltColor)
	}
	name := strings.TrimSpace(member.Name)
	if name == "" {
		name = "Player"
	}
	if member.Leader() {
		name += " *"
	}
	state := "Offline"
	stateColor := rotheme.Default.Colors.MutedText
	if member.Online() {
		state = trimRunes(member.MapName, 12)
		if state == "" {
			state = "Online"
		}
		stateColor = Color(GoodTextColor)
	}
	row := &partyRowWidget{
		member:     member,
		name:       name,
		state:      state,
		stateColor: stateColor,
		bg:         bg,
		hp:         member.HP,
		maxHP:      member.MaxHP,
		onMenu: func(member session.PartyMember) {
			x, y := w.x+friendsWindowWidth/2, w.y+ROWindowTitleHeight+friendsTabHeight+friendsRowHeight
			if w.ctx.Input != nil {
				x, y = w.ctx.Input.MouseX, w.ctx.Input.MouseY
			}
			w.partyContextMenu.Open(w.ctx, x, y, member, partyCanManageSession(w.ctx.Session), w.ctx.Session != nil && member.AccountID == w.ctx.Session.AccountID)
		},
	}
	row.SetVisible(true)
	row.SetEnabled(true)
	return row
}

type partyRowWidget struct {
	widget.WidgetBase
	member     session.PartyMember
	name       string
	state      string
	stateColor widget.Color
	bg         widget.Color
	hp         int
	maxHP      int
	hovered    bool
	onMenu     func(session.PartyMember)
}

func (w *partyRowWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(friendsWindowWidth, friendsRowHeight))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *partyRowWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	bounds := w.Bounds()
	bg := w.bg
	if w.hovered {
		bg = rotheme.Default.Colors.ButtonHover
	}
	canvas.DrawRect(bounds, bg)
	textY := bounds.Min.Y + 3
	rotheme.DrawText(canvas, trimRunes(w.name, 20), geometry.NewRect(bounds.Min.X+8, textY, bounds.Width()-88, 13), rotheme.Default.Typography.TextSize, rotheme.Default.Colors.Text, false, widget.TextAlignLeft)
	rotheme.DrawText(canvas, w.state, geometry.NewRect(bounds.Max.X-72, textY, 64, 13), rotheme.Default.Typography.TextSize, w.stateColor, false, widget.TextAlignRight)
	bar := geometry.NewRect(bounds.Min.X+8, bounds.Max.Y-8, bounds.Width()-16, 5)
	canvas.DrawRect(bar, widget.RGBA8(224, 232, 242, 255))
	fillW := float32(0)
	if w.maxHP > 0 && w.hp > 0 {
		fillW = bar.Width() * float32(w.hp) / float32(w.maxHP)
		if fillW > bar.Width() {
			fillW = bar.Width()
		}
	}
	if fillW > 0 {
		canvas.DrawRect(geometry.NewRect(bar.Min.X, bar.Min.Y, fillW, bar.Height()), Color(PlayerHPBarColor))
	}
	canvas.StrokeRect(bar, rotheme.Default.Colors.WindowBorder, 1)
}

func (w *partyRowWidget) Event(ctx widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		w.hovered = true
		ctx.SetCursor(widget.CursorPointer)
		return true
	case event.MouseLeave:
		w.hovered = false
		ctx.SetCursor(widget.CursorDefault)
		return true
	case event.MousePress:
		if mouse.Button == event.ButtonRight && w.onMenu != nil {
			w.onMenu(w.member)
			return true
		}
	}
	return false
}

func (w *partyRowWidget) Children() []widget.Widget {
	return nil
}

func sessionFriends(s *session.Session) []session.Friend {
	if s == nil {
		return nil
	}
	return s.Friends.List
}

func sessionParty(s *session.Session) session.Party {
	if s == nil {
		return session.Party{}
	}
	return s.Party
}

func partyCanManageSession(s *session.Session) bool {
	if s == nil || !s.Party.Active() {
		return false
	}
	if len(s.Party.Members) == 0 {
		return true
	}
	for _, member := range s.Party.Members {
		if member.AccountID == s.AccountID {
			return member.Leader()
		}
	}
	return false
}

func friendsWindowSnapshot(s *session.Session) string {
	friends := sessionFriends(s)
	party := sessionParty(s)
	var b strings.Builder
	fmt.Fprintf(&b, "friends=%d", len(friends))
	for _, friend := range friends {
		fmt.Fprintf(&b, ";%d:%d:%s:%d", friend.AccountID, friend.CharID, friend.Name, friend.State)
	}
	fmt.Fprintf(&b, ";party=%s:%d:%t:%d", party.Name, party.ExpShare, party.RefuseInvites, len(party.Members))
	for _, member := range party.Members {
		fmt.Fprintf(&b, ";%d:%s:%s:%d:%d:%d:%d:%d:%t", member.AccountID, member.Name, member.MapName, member.Role, member.State, member.HP, member.MaxHP, member.X, member.Dead)
	}
	return b.String()
}
