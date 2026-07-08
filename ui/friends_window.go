package ui

import (
	"fmt"
	"strings"

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
	window   WindowState
	ctx      Context
	snapshot string
}

func (w *FriendsWindow) Toggle(ctx Context) {
	if w.IsOpen() {
		w.Close()
		return
	}
	w.OpenWindow(ctx)
}

func (w *FriendsWindow) OpenWindow(ctx Context) {
	w.ensureWindow()
	w.ctx = ctx
	w.snapshot = friendsWindowSnapshot(ctx.Session)
	w.window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *FriendsWindow) Update(ctx Context) bool {
	w.ensureWindow()
	w.ctx = ctx
	if !w.window.IsOpen() {
		return false
	}
	nextSnapshot := friendsWindowSnapshot(ctx.Session)
	if nextSnapshot != w.snapshot {
		w.snapshot = nextSnapshot
		w.window.SetContent(w.widgetTree(ctx))
	}
	consumed := w.window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *FriendsWindow) IsOpen() bool {
	w.ensureWindow()
	return w.window.IsOpen()
}

func (w *FriendsWindow) Close() {
	w.ensureWindow()
	w.window.Close()
	w.Publish(w.ctx)
}

func (w *FriendsWindow) Publish(ctx Context) {
	w.ensureWindow()
	w.window.Publish(ctx)
}

func (w *FriendsWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(friendsWindowWidth, friendsWindowHeight)
	}
}

func (w *FriendsWindow) widgetTree(ctx Context) widget.Widget {
	friends := sessionFriends(ctx.Session)
	return Window(
		Title(fmt.Sprintf("Friends (%d/%d)", len(friends), friendsListMax)),
		CloseButton(true),
		OnClose(w.Close),
		Size(friendsWindowWidth, friendsWindowHeight),
		Content(
			primitives.Box(
				friendsTabs(),
				friendsList(friends),
			).
				Gap(-1),
		),
	)
}

func friendsTabs() widget.Widget {
	return primitives.HBox(
		newTabWidget(tabWidgetConfig{
			label:      "Friends",
			active:     true,
			width:      friendsTabWidth,
			height:     friendsTabHeight,
			blendEdge:  tabBlendBottom,
			blendInset: 1,
		}),
		newTabWidget(tabWidgetConfig{
			label:  "Party",
			width:  friendsTabWidth,
			height: friendsTabHeight,
		}),
		primitives.Expanded(primitives.Box()),
	).Gap(-1)
}

func friendsList(friends []session.Friend) widget.Widget {
	rows := make([]widget.Widget, 0, maxInt(1, len(friends)))
	if len(friends) == 0 {
		rows = append(rows,
			primitives.Box(
				rotheme.Text("No friends").
					Color(rotheme.Default.Colors.MutedText).
					Align(primitives.TextAlignCenter),
			).
				Height(friendsRowHeight).
				CrossAlign(primitives.CrossAxisStretch),
		)
	} else {
		for i, friend := range friends {
			rows = append(rows, friendRow(friend, i))
		}
	}
	return primitives.Box(rows...).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		CrossAlign(primitives.CrossAxisStretch)
}

func friendRow(friend session.Friend, index int) widget.Widget {
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
		name = fmt.Sprintf("%d", friend.CharID)
	}
	return primitives.HBox(
		rotheme.Text(trimRunes(name, 24)).
			Align(primitives.TextAlignStart),
		primitives.Expanded(primitives.Box()),
		rotheme.Text(state).
			Color(stateColor).
			Align(primitives.TextAlignEnd),
	).
		PaddingXY(8, 0).
		Height(friendsRowHeight).
		Background(bg).
		CrossAlign(primitives.CrossAxisCenter)
}

func sessionFriends(s *session.Session) []session.Friend {
	if s == nil {
		return nil
	}
	return s.Friends.List
}

func friendsWindowSnapshot(s *session.Session) string {
	friends := sessionFriends(s)
	var b strings.Builder
	fmt.Fprintf(&b, "%d", len(friends))
	for _, friend := range friends {
		fmt.Fprintf(&b, ";%d:%d:%s:%d", friend.AccountID, friend.CharID, friend.Name, friend.State)
	}
	return b.String()
}
