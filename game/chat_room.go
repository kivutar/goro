package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func (m *WorldMode) updateChatRoomWindows(ctx client.Context) bool {
	createConsumed := m.ui.chatRoomCreate.Update(ctx)
	if action := m.ui.chatRoomCreate.PopAction(); action.Title != "" {
		m.createChatRoomFromWindow(ctx, action)
		return true
	}
	if createConsumed {
		return true
	}
	roomConsumed := m.ui.chatRoom.Update(ctx)
	if action := m.ui.chatRoom.PopAction(); action.Leave || action.Message != "" {
		m.handleChatRoomAction(ctx, action)
		return true
	}
	return roomConsumed
}

func (m *WorldMode) createChatRoomFromWindow(ctx client.Context, action gameui.ChatRoomCreateWindowAction) {
	room := network.ChatRoomCreate{
		Title:    strings.TrimSpace(action.Title),
		Password: strings.TrimSpace(action.Password),
		Limit:    action.Limit,
		Public:   action.Public,
	}
	if room.Title == "" {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Create chat room failed: not connected.")
		return
	}
	if err := ctx.Network.SendCreateChatRoom(room); err != nil {
		m.ui.console.AddErrorMessage("Create chat room failed.")
		glog.Warnf("chat room create failed title=%q limit=%d public=%t: %v", room.Title, room.Limit, room.Public, err)
		return
	}
	m.pendingChatRoom = room
}

func (m *WorldMode) handleChatRoomCreateAck(ctx client.Context, ack network.ChatRoomCreateAck) {
	switch ack.Result {
	case 0:
		room := m.pendingChatRoom
		if room.Title == "" {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: 20, Public: true}
		}
		m.pendingChatRoom = network.ChatRoomCreate{}
		member := selectedCharacterName(ctx.Session)
		m.ui.chatRoom.Open(ctx, room.Title, room.Limit, room.Public, []string{member})
		m.ui.console.AddBlueMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	case 1, 2:
		m.pendingChatRoom = network.ChatRoomCreate{}
		m.ui.console.AddErrorMessage("%s", chatRoomCreateAckMessage(ctx, ack))
	default:
		m.pendingChatRoom = network.ChatRoomCreate{}
		m.ui.console.AddErrorMessage("Create chat room failed.")
	}
}

func (m *WorldMode) applyChatRoomBoard(ctx client.Context, board network.ChatRoomBoard) {
	applyChatRoomBoardToWorld(ctx, board)
	glog.Debugf("chat room board actor=%d room=%d title=%q count=%d limit=%d public=%t", board.OwnerID, board.RoomID, board.Title, board.Count, board.Limit, board.Public)
}

func applyChatRoomBoardToWorld(ctx client.Context, board network.ChatRoomBoard) {
	if ctx.World == nil {
		return
	}
	if isLocalActor(ctx, board.OwnerID) {
		ctx.World.Player.ChatRoom = true
		ctx.World.Player.ChatRoomID = board.RoomID
		ctx.World.Player.ChatRoomTitle = board.Title
		ctx.World.Player.ChatRoomCount = board.Count
		ctx.World.Player.ChatRoomLimit = board.Limit
		ctx.World.Player.ChatRoomPublic = board.Public
		return
	}
	actor, ok := ctx.World.Actors[board.OwnerID]
	if !ok {
		actor = worldstate.Actor{ID: board.OwnerID}
	}
	actor.ChatRoom = true
	actor.ChatRoomID = board.RoomID
	actor.ChatRoomTitle = board.Title
	actor.ChatRoomCount = board.Count
	actor.ChatRoomLimit = board.Limit
	actor.ChatRoomPublic = board.Public
	ctx.World.Actors[board.OwnerID] = actor
}

func (m *WorldMode) applyChatRoomDestroy(ctx client.Context, destroy network.ChatRoomDestroy) {
	applyChatRoomDestroyToWorld(ctx, destroy)
	glog.Debugf("chat room board removed room=%d", destroy.RoomID)
}

func applyChatRoomDestroyToWorld(ctx client.Context, destroy network.ChatRoomDestroy) {
	if ctx.World == nil {
		return
	}
	if ctx.World.Player.ChatRoomID == destroy.RoomID {
		clearActorChatRoom(&ctx.World.Player)
	}
	for id, actor := range ctx.World.Actors {
		if actor.ChatRoomID != destroy.RoomID {
			continue
		}
		clearActorChatRoom(&actor)
		ctx.World.Actors[id] = actor
		return
	}
}

func clearActorChatRoom(actor *worldstate.Actor) {
	actor.ChatRoom = false
	actor.ChatRoomID = 0
	actor.ChatRoomTitle = ""
	actor.ChatRoomCount = 0
	actor.ChatRoomLimit = 0
	actor.ChatRoomPublic = false
}

func actorHasChatRoom(actor worldstate.Actor) bool {
	return actor.ChatRoom && actor.ChatRoomID != 0 && actor.ChatRoomTitle != ""
}

func (m *WorldMode) handleChatRoomAction(ctx client.Context, action gameui.ChatRoomWindowAction) {
	if action.Leave {
		if ctx.Network == nil {
			m.ui.console.AddErrorMessage("Leave chat room failed: not connected.")
			return
		}
		if err := ctx.Network.SendExitChatRoom(); err != nil {
			m.ui.console.AddErrorMessage("Leave chat room failed.")
			glog.Warnf("chat room leave failed: %v", err)
		}
		return
	}
	message := strings.TrimSpace(action.Message)
	if message == "" {
		return
	}
	name := selectedCharacterName(ctx.Session)
	if name == "" {
		m.ui.chatRoom.AddError(ctx, "send failed: missing character name")
		m.ui.console.AddErrorMessage("Chat room send failed: missing character name.")
		return
	}
	if ctx.Network == nil {
		m.ui.chatRoom.AddError(ctx, "send failed: not connected")
		m.ui.console.AddErrorMessage("Chat room send failed: not connected.")
		return
	}
	if err := ctx.Network.SendGlobalChat(name, message); err != nil {
		m.ui.chatRoom.AddError(ctx, "send failed: "+err.Error())
		m.ui.console.AddErrorMessage("Chat room send failed.")
		glog.Warnf("chat room send failed: %v", err)
	}
}

func (m *WorldMode) requestChatRoomEnter(ctx client.Context, actor worldstate.Actor) {
	if actor.ChatRoomID == 0 {
		return
	}
	if !actor.ChatRoomPublic {
		m.ui.console.AddErrorMessage("Private chat rooms need a password.")
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Enter chat room failed: not connected.")
		return
	}
	if err := ctx.Network.SendEnterChatRoom(actor.ChatRoomID, ""); err != nil {
		m.ui.console.AddErrorMessage("Enter chat room failed.")
		glog.Warnf("chat room enter failed room=%d title=%q: %v", actor.ChatRoomID, actor.ChatRoomTitle, err)
		return
	}
	m.pendingChatRoom = network.ChatRoomCreate{
		Title:  actor.ChatRoomTitle,
		Limit:  actor.ChatRoomLimit,
		Public: actor.ChatRoomPublic,
	}
}

func (m *WorldMode) handleChatRoomEnter(ctx client.Context, enter network.ChatRoomEnter) {
	members := make([]string, 0, len(enter.Members))
	owner := ""
	for _, member := range enter.Members {
		name := strings.TrimSpace(member.Name)
		if name == "" {
			continue
		}
		members = append(members, name)
		if member.Owner {
			owner = name
		}
	}
	if !m.ui.chatRoom.IsOpen() {
		room := m.pendingChatRoom
		if room.Title == "" {
			room = network.ChatRoomCreate{Title: "Chat Room", Limit: uint16(len(members)), Public: true}
		}
		m.ui.chatRoom.Open(ctx, room.Title, room.Limit, room.Public, members)
	}
	m.ui.chatRoom.SetMembers(ctx, members, owner)
	m.pendingChatRoom = network.ChatRoomCreate{}
}

func (m *WorldMode) handleChatRoomMemberJoin(ctx client.Context, join network.ChatRoomMemberJoin) {
	if !m.ui.chatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(join.Name)
	m.ui.chatRoom.AddMember(ctx, name, join.Count)
	m.ui.chatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, 179, "%s entered the chat room.", name))
}

func (m *WorldMode) handleChatRoomMemberLeave(ctx client.Context, leave network.ChatRoomMemberLeave) {
	if !m.ui.chatRoom.IsOpen() {
		return
	}
	name := strings.TrimSpace(leave.Name)
	m.ui.chatRoom.RemoveMember(ctx, name, leave.Count)
	messageID := 180
	fallback := "%s left the chat room."
	if leave.Kicked {
		messageID = 181
		fallback = "%s has been kicked out of the chat room."
	}
	m.ui.chatRoom.AddSystem(ctx, chatRoomMemberMessage(ctx, messageID, fallback, name))
}

func (m *WorldMode) handleChatRoomChange(ctx client.Context, change network.ChatRoomChange) {
	if !m.ui.chatRoom.IsOpen() {
		return
	}
	m.ui.chatRoom.UpdateRoom(ctx, change.Title, change.Limit, change.Count, change.Public)
}

func (m *WorldMode) handleChatRoomRoleChange(ctx client.Context, role network.ChatRoomRoleChange) {
	if !m.ui.chatRoom.IsOpen() || !role.Owner {
		return
	}
	m.ui.chatRoom.SetOwner(ctx, role.Name)
}

func (m *WorldMode) addChatRoomMessage(ctx client.Context, chat network.ChatMessage) {
	if !m.ui.chatRoom.IsOpen() {
		return
	}
	text := formatConsoleMessage(ctx.Resources, chat)
	if text == "" {
		return
	}
	m.ui.chatRoom.AddMessage(ctx, text)
}

func (m *WorldMode) handleChatMessage(ctx client.Context, chat network.ChatMessage, now time.Time) {
	if chat.Announcement {
		text := formatConsoleMessage(ctx.Resources, chat)
		if text == "" {
			return
		}
		m.ui.announcement.Show(text, roClientColor(chat.Color), gameui.AnnouncementStyle{
			Y:        int(chat.FontY),
			FontSize: int(chat.FontSize),
			Bold:     chat.FontType >= 700,
		}, now)
		addConsoleMessage(&m.ui.console, ctx.Resources, chat)
		return
	}
	if m.ui.chatRoom.IsOpen() && !chat.HasColor {
		m.addChatRoomMessage(ctx, chat)
		return
	}
	m.applySpeechBubble(ctx, chat, now)
	addConsoleMessage(&m.ui.console, ctx.Resources, chat)
}

func chatRoomMemberMessage(ctx client.Context, messageID int, fallback string, name string) string {
	name = strings.TrimSpace(name)
	message := ""
	if ctx.Resources != nil {
		message, _ = ctx.Resources.MsgString(messageID)
	}
	if message == "" {
		message = fallback
	}
	if strings.Contains(message, "%s") {
		return strings.Replace(message, "%s", name, 1)
	}
	if name != "" {
		return strings.TrimSpace(message + " " + name)
	}
	return message
}

func chatRoomCreateAckMessage(ctx client.Context, ack network.ChatRoomCreateAck) string {
	if ctx.Resources != nil {
		if text, ok := ctx.Resources.MsgString(64 + int(ack.Result)); ok && strings.TrimSpace(text) != "" {
			return text
		}
	}
	switch ack.Result {
	case 0:
		return "Chat room has been created."
	case 1:
		return "Chat room limit exceeded."
	case 2:
		return "Same chat room already exists."
	default:
		return fmt.Sprintf("Create chat room failed (%d).", ack.Result)
	}
}
