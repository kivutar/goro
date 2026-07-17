package game

import (
	"fmt"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/input"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func (m *WorldMode) openFriendRequest(ctx client.Context, request network.FriendRequest) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		name = "Someone"
	}
	glog.Debugf("friend request aid=%d gid=%d name=%q", request.AccountID, request.CharID, request.Name)
	m.ui.friendRequest.Open(ctx, "Friend Request", fmt.Sprintf("%s wants to be friends with you.", name), func() {
		if ctx.Network == nil {
			glog.Warnf("friend request accept failed: not connected")
			return
		}
		if err := ctx.Network.SendFriendRequestAck(request.AccountID, request.CharID, true); err != nil {
			glog.Warnf("friend request accept failed aid=%d gid=%d: %v", request.AccountID, request.CharID, err)
		}
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("friend request reject failed: not connected")
			return
		}
		if err := ctx.Network.SendFriendRequestAck(request.AccountID, request.CharID, false); err != nil {
			glog.Warnf("friend request reject failed aid=%d gid=%d: %v", request.AccountID, request.CharID, err)
		}
	})
}

func (m *WorldMode) addFriendResultMessage(result network.FriendAddResult) {
	name := strings.TrimSpace(result.Name)
	if name == "" {
		name = "Player"
	}
	switch result.Result {
	case 0:
		m.ui.console.AddBlueMessage("You have become friends with %s.", name)
	case 1:
		m.ui.console.AddErrorMessage("%s does not want to be friends with you.", name)
	case 2:
		m.ui.console.AddErrorMessage("Your Friend List is full.")
	case 3:
		m.ui.console.AddErrorMessage("%s's Friend List is full.", name)
	default:
		m.ui.console.AddErrorMessage("Friend request failed.")
	}
}

func (m *WorldMode) openPlayerContextFromInput(ctx client.Context, now time.Time) bool {
	if ctx.Input == nil || !ctx.Input.MouseJustPressed(input.MouseButtonRight) || uiPointerBlocked(ctx) {
		return false
	}
	screenW, screenH := ctx.ScreenSize()
	projection := m.sceneProjection(ctx, screenW, screenH, now)
	actor, ok := clickedPlayerTarget(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now, m.actorDeaths)
	if !ok {
		return false
	}
	m.ui.playerContext.Open(ctx, ctx.Input.MouseX, ctx.Input.MouseY, actor.ID, actor.Name, !friendNameInSession(ctx.Session, actor.Name), partyCanInvite(ctx.Session), true)
	return true
}

func (m *WorldMode) sendAddFriend(ctx client.Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("add friend failed name=%q: not connected", name)
		return
	}
	if err := ctx.Network.SendAddFriend(name); err != nil {
		glog.Warnf("add friend failed name=%q: %v", name, err)
	}
}

func (m *WorldMode) openDeleteFriendConfirm(ctx client.Context, friend session.Friend) {
	name := strings.TrimSpace(friend.Name)
	if name == "" {
		name = "this friend"
	}
	m.ui.friendConfirm.Open(ctx, "Delete Friend", fmt.Sprintf("Delete %s from your friend list?", name), func() {
		if ctx.Network == nil {
			m.ui.console.AddErrorMessage("Delete friend failed: not connected.")
			return
		}
		if err := ctx.Network.SendDeleteFriend(friend.AccountID, friend.CharID); err != nil {
			m.ui.console.AddErrorMessage("Delete friend failed.")
			glog.Warnf("delete friend failed aid=%d gid=%d name=%q: %v", friend.AccountID, friend.CharID, friend.Name, err)
			return
		}
		m.ui.console.AddSystemMessage("Delete friend request sent for %s.", name)
	}, nil)
}

func partyCanInvite(s *session.Session) bool {
	return partyCanManage(s)
}

func friendNameInSession(s *session.Session, name string) bool {
	name = strings.TrimSpace(name)
	if s == nil || name == "" {
		return false
	}
	for _, friend := range s.Friends.List {
		if friend.Name == name {
			return true
		}
	}
	return false
}

func applyFriendsList(ctx client.Context, friends []network.Friend) {
	if ctx.Session == nil {
		return
	}
	known := make(map[[2]uint32]string, len(ctx.Session.Friends.List))
	for _, friend := range ctx.Session.Friends.List {
		if friend.Name != "" {
			known[[2]uint32{friend.AccountID, friend.CharID}] = friend.Name
		}
	}
	ctx.Session.Friends.List = ctx.Session.Friends.List[:0]
	for _, friend := range friends {
		next := sessionFriendFromNetwork(friend)
		if next.Name == "" {
			next.Name = known[[2]uint32{next.AccountID, next.CharID}]
		}
		ctx.Session.Friends.List = append(ctx.Session.Friends.List, next)
	}
	glog.Debugf("friend list received count=%d", len(ctx.Session.Friends.List))
}

func applyFriendState(ctx client.Context, state network.FriendState) {
	if ctx.Session == nil {
		return
	}
	for i := range ctx.Session.Friends.List {
		friend := &ctx.Session.Friends.List[i]
		if friend.AccountID != state.AccountID || friend.CharID != state.CharID {
			continue
		}
		friend.State = state.State
		if state.Name != "" {
			friend.Name = state.Name
		}
		glog.Debugf("friend state aid=%d gid=%d name=%q state=%d online=%t", friend.AccountID, friend.CharID, friend.Name, friend.State, friend.Online())
		return
	}
	ctx.Session.Friends.List = append(ctx.Session.Friends.List, session.Friend{
		AccountID: state.AccountID,
		CharID:    state.CharID,
		Name:      state.Name,
		State:     state.State,
	})
	glog.Debugf("friend state aid=%d gid=%d name=%q state=%d online=%t", state.AccountID, state.CharID, state.Name, state.State, state.State == 0)
}

func applyFriendAddResult(ctx client.Context, result network.FriendAddResult) {
	if ctx.Session == nil || result.Result != 0 {
		return
	}
	upsertSessionFriend(ctx.Session, session.Friend{
		AccountID: result.AccountID,
		CharID:    result.CharID,
		Name:      result.Name,
		State:     0,
	})
	glog.Debugf("friend added aid=%d gid=%d name=%q", result.AccountID, result.CharID, result.Name)
}

func applyFriendDelete(ctx client.Context, deleted network.FriendDelete) (session.Friend, bool) {
	if ctx.Session == nil {
		return session.Friend{}, false
	}
	for i := range ctx.Session.Friends.List {
		friend := ctx.Session.Friends.List[i]
		if friend.AccountID != deleted.AccountID || friend.CharID != deleted.CharID {
			continue
		}
		ctx.Session.Friends.List = append(ctx.Session.Friends.List[:i], ctx.Session.Friends.List[i+1:]...)
		glog.Debugf("friend deleted aid=%d gid=%d name=%q", friend.AccountID, friend.CharID, friend.Name)
		return friend, true
	}
	glog.Warnf("friend delete not found aid=%d gid=%d", deleted.AccountID, deleted.CharID)
	return session.Friend{}, false
}

func sessionFriendFromNetwork(friend network.Friend) session.Friend {
	return session.Friend{
		AccountID: friend.AccountID,
		CharID:    friend.CharID,
		Name:      friend.Name,
		State:     friend.State,
	}
}

func upsertSessionFriend(s *session.Session, friend session.Friend) {
	for i := range s.Friends.List {
		if s.Friends.List[i].AccountID != friend.AccountID || s.Friends.List[i].CharID != friend.CharID {
			continue
		}
		s.Friends.List[i] = friend
		return
	}
	s.Friends.List = append(s.Friends.List, friend)
}
