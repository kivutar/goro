package ui

import (
	"math"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func TestPartyFooterAlignsButtonsRight(t *testing.T) {
	var window FriendsWindow
	footer := primitives.HBox(window.partyFooter(session.Party{Name: "Party"})...).
		Gap(ROWindowFooterGap).
		CrossAlign(primitives.CrossAxisCenter)

	footer.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(180, 24)))

	children := footer.Children()
	if len(children) != 3 {
		t.Fatalf("party footer children = %d, want spacer and two buttons", len(children))
	}
	leave := children[2].(interface{ Bounds() geometry.Rect }).Bounds()
	if leave.Max.X != 180 {
		t.Fatalf("leave button right edge = %.1f, want 180.0", leave.Max.X)
	}
}

func TestPartyFooterShowsCreateWhenNoParty(t *testing.T) {
	var window FriendsWindow
	footer := window.partyFooter(session.Party{})

	if len(footer) != 2 {
		t.Fatalf("empty party footer children = %d, want spacer and create button", len(footer))
	}
}

func TestPartyFooterShowsInviteForLeader(t *testing.T) {
	var window FriendsWindow
	window.ctx = Context{Session: &session.Session{
		AccountID: 10,
		Party: session.Party{
			Name:    "Goro",
			Members: []session.PartyMember{{AccountID: 10, Role: 0}},
		},
	}}
	footer := window.partyFooter(window.ctx.Session.Party)

	if len(footer) != 4 {
		t.Fatalf("leader party footer children = %d, want spacer, invite, settings, leave", len(footer))
	}
}

func TestEmptyFriendsListExpandsAndCentersText(t *testing.T) {
	assertEmptyListExpandsAndCenters(t, friendsList(nil), "No friends")
}

func TestEmptyPartyListExpandsAndCentersText(t *testing.T) {
	assertEmptyListExpandsAndCenters(t, partyList(session.Party{}), "No party")
}

func TestFriendsWindowListStartsBelowTabs(t *testing.T) {
	var window FriendsWindow
	root := window.widgetTree(Context{Session: &session.Session{}})

	root.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(friendsWindowWidth, friendsWindowHeight)))
	rootChildren := root.Children()
	if len(rootChildren) < 2 {
		t.Fatalf("window children = %d, want title and content", len(rootChildren))
	}
	contentChildren := rootChildren[1].Children()
	if len(contentChildren) != 1 {
		t.Fatalf("content wrapper children = %d, want content box", len(contentChildren))
	}
	stackChildren := contentChildren[0].Children()
	if len(stackChildren) != 2 {
		t.Fatalf("content stack children = %d, want tabs and list", len(stackChildren))
	}
	tabsBounds := stackChildren[0].(interface{ Bounds() geometry.Rect }).Bounds()
	listBounds := stackChildren[1].(interface{ Bounds() geometry.Rect }).Bounds()
	if listBounds.Min.Y != tabsBounds.Height() {
		t.Fatalf("list top = %.1f, want tabs height %.1f", listBounds.Min.Y, tabsBounds.Height())
	}
}

func TestFriendRowLeftClickStartsWhisper(t *testing.T) {
	var window FriendsWindow
	friend := session.Friend{AccountID: 10, CharID: 20, Name: "Alice Smith"}
	row := window.friendRow(friend, 0)

	row.Event(widget.NewContext(), event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		event.ButtonStateLeft,
		geometry.Pt(4, 4),
		geometry.Pt(40, 40),
		0,
	))

	action := window.PopAction()
	if action.Kind != FriendsWindowActionFriendWhisper {
		t.Fatalf("action kind = %d, want whisper", action.Kind)
	}
	if action.Friend.Name != friend.Name || action.Friend.AccountID != friend.AccountID || action.Friend.CharID != friend.CharID {
		t.Fatalf("action friend = %+v, want %+v", action.Friend, friend)
	}
}

func TestPartyRowRightClickOpensContextMenu(t *testing.T) {
	var window FriendsWindow
	window.ctx = Context{
		Session: &session.Session{
			AccountID: 10,
			Party: session.Party{
				Name:    "Goro",
				Members: []session.PartyMember{{AccountID: 10, Role: 0}},
			},
		},
		ScreenW: 800,
		ScreenH: 600,
	}
	member := session.PartyMember{AccountID: 11, Name: "Alice"}
	row := window.partyRow(window.ctx, member, 0)

	row.Event(widget.NewContext(), event.NewMouseEvent(
		event.MousePress,
		event.ButtonRight,
		event.ButtonStateRight,
		geometry.Pt(4, 4),
		geometry.Pt(40, 40),
		0,
	))

	if !window.partyContextMenu.IsOpen() {
		t.Fatal("party row right click did not open context menu")
	}
}

func TestFriendsWindowDrainsClosedContextMenuAction(t *testing.T) {
	var window FriendsWindow
	friend := session.Friend{AccountID: 10, CharID: 20, Name: "Alice Smith"}
	window.contextMenu.action = FriendsWindowAction{Kind: FriendsWindowActionFriendWhisper, Friend: friend}

	if !window.Update(Context{}) {
		t.Fatal("friends window did not consume pending context menu action")
	}

	action := window.PopAction()
	if action.Kind != FriendsWindowActionFriendWhisper || action.Friend.Name != friend.Name {
		t.Fatalf("action = %+v, want whisper for %+v", action, friend)
	}
}

func TestPartyCreateWindowSubmitAction(t *testing.T) {
	var window PartyCreateWindow
	window.Open(Context{})
	window.name = "Goro"
	window.itemPickup = 1
	window.itemDivision = 1

	window.submit(Context{})

	action := window.PopAction()
	if action.Name != "Goro" || action.ItemPickup != 1 || action.ItemDivision != 1 {
		t.Fatalf("party create action = %+v", action)
	}
}

func TestPartyCreateWindowFocusedEnterSubmitsImmediately(t *testing.T) {
	inputState := input.NewState()
	ctx := Context{Input: inputState}
	var window PartyCreateWindow
	window.Open(ctx)
	window.name = "Goro"
	window.nameInput(ctx).SetFocused(true)
	inputState.SetKey(input.KeyEnter, true)

	if !window.Update(ctx) {
		t.Fatal("focused enter did not consume party create update")
	}
	if action := window.PopAction(); action.Name != "Goro" {
		t.Fatalf("party create action = %+v, want Goro", action)
	}
}

func TestPartyInviteWindowSubmitAction(t *testing.T) {
	var window PartyInviteWindow
	window.Open(Context{})
	window.value = "Alice"

	window.submit(Context{})

	if action := window.PopAction(); action != "Alice" {
		t.Fatalf("party invite action = %q, want Alice", action)
	}
}

func TestPartyInviteWindowFocusedEnterSubmitsImmediately(t *testing.T) {
	inputState := input.NewState()
	ctx := Context{Input: inputState}
	var window PartyInviteWindow
	window.Open(ctx)
	window.value = "Alice"
	window.input(ctx).SetFocused(true)
	inputState.SetKey(input.KeyEnter, true)

	if !window.Update(ctx) {
		t.Fatal("focused enter did not consume party invite update")
	}
	if action := window.PopAction(); action != "Alice" {
		t.Fatalf("party invite action = %q, want Alice", action)
	}
}

func TestFriendSettingsDefaultEnabled(t *testing.T) {
	settings := friendSettings(&session.Session{})
	if !settings.OpenStrangers || !settings.OpenFriends || !settings.Alert {
		t.Fatalf("default friend settings = %+v, want all enabled", settings)
	}
}

func assertEmptyListExpandsAndCenters(t *testing.T, list widget.Widget, label string) {
	t.Helper()

	list.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(220, 120)))
	listBounds := list.(interface{ Bounds() geometry.Rect }).Bounds()
	if listBounds.Width() != 220 || listBounds.Height() != 120 {
		t.Fatalf("%s list bounds = %.1fx%.1f, want 220.0x120.0", label, listBounds.Width(), listBounds.Height())
	}

	children := list.Children()
	if len(children) != 1 {
		t.Fatalf("%s list children = %d, want empty state only", label, len(children))
	}
	emptyState, ok := children[0].(*primitives.ExpandedWidget)
	if !ok {
		t.Fatalf("%s empty state = %T, want ExpandedWidget", label, children[0])
	}
	emptyBounds := emptyState.Bounds()
	if emptyBounds.Width() != 220 || emptyBounds.Height() != 120 {
		t.Fatalf("%s empty bounds = %.1fx%.1f, want 220.0x120.0", label, emptyBounds.Width(), emptyBounds.Height())
	}

	inner := emptyState.Children()[0]
	innerChildren := inner.Children()
	if len(innerChildren) != 3 {
		t.Fatalf("%s empty children = %d, want top spacer, text, bottom spacer", label, len(innerChildren))
	}
	textBounds := innerChildren[1].(interface{ Bounds() geometry.Rect }).Bounds()
	if textBounds.Width() != 220 {
		t.Fatalf("%s text width = %.1f, want 220.0", label, textBounds.Width())
	}
	textCenterY := textBounds.Min.Y + textBounds.Height()/2
	if math.Abs(float64(textCenterY-60)) > 0.5 {
		t.Fatalf("%s text center y = %.1f, want 60.0", label, textCenterY)
	}
}
