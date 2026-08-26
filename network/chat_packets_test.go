package network

import (
	"encoding/binary"
	"testing"
)

func TestBuildGlobalChatPacketFor2008ClientDate(t *testing.T) {
	packet := BuildGlobalChatPacketForClientDate("Kivutar", "hello", 20080910)
	if got := ID(packet); got != PacketCZRequestChat {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZRequestChat)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); int(got) != len(packet) {
		t.Fatalf("length = %d, want %d", got, len(packet))
	}
	if got := string(packet[4 : len(packet)-1]); got != "Kivutar : hello" {
		t.Fatalf("payload = %q", got)
	}
	if packet[len(packet)-1] != 0 {
		t.Fatalf("packet is not nul terminated")
	}
}

func TestBuildWhisperPacket(t *testing.T) {
	packet := BuildWhisperPacket("Rekka", "hello")
	if got := ID(packet); got != PacketCZWhisper {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZWhisper)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); int(got) != len(packet) {
		t.Fatalf("length = %d, want %d", got, len(packet))
	}
	if got := string(packet[4:9]); got != "Rekka" {
		t.Fatalf("receiver prefix = %q", got)
	}
	if packet[9] != 0 {
		t.Fatalf("receiver is not nul terminated")
	}
	if got := string(packet[28 : len(packet)-1]); got != "hello" {
		t.Fatalf("message = %q", got)
	}
	if packet[len(packet)-1] != 0 {
		t.Fatalf("packet is not nul terminated")
	}
}

func TestBuildWhisperIgnorePackets(t *testing.T) {
	packet := BuildWhisperIgnorePacket("Rekka", false)
	if got := ID(packet); got != PacketCZWhisperIgnore {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZWhisperIgnore)
	}
	if len(packet) != 27 {
		t.Fatalf("len = %d, want 27", len(packet))
	}
	if got := string(packet[2:7]); got != "Rekka" {
		t.Fatalf("name prefix = %q", got)
	}
	if packet[7] != 0 {
		t.Fatalf("name is not nul terminated")
	}
	if packet[26] != 0 {
		t.Fatalf("type = %d, want deny 0", packet[26])
	}

	packet = BuildWhisperIgnorePacket("123456789012345678901234", false)
	if got := string(packet[2:25]); got != "12345678901234567890123" || packet[25] != 0 {
		t.Fatalf("long name field = %x", packet[2:26])
	}

	packet = BuildWhisperIgnorePacket("Rekka", true)
	if packet[26] != 1 {
		t.Fatalf("type = %d, want allow 1", packet[26])
	}

	packet = BuildWhisperIgnoreAllPacket(false)
	if got := ID(packet); got != PacketCZWhisperIgnoreAll {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZWhisperIgnoreAll)
	}
	if len(packet) != 3 || packet[2] != 0 {
		t.Fatalf("ignore all packet = %x, want deny", packet)
	}

	packet = BuildWhisperIgnoreAllPacket(true)
	if len(packet) != 3 || packet[2] != 1 {
		t.Fatalf("ignore all packet = %x, want allow", packet)
	}
}

func TestBuildCreateChatRoomPacket(t *testing.T) {
	packet := BuildCreateChatRoomPacket(ChatRoomCreate{
		Title:    "Potion shop",
		Password: "secret",
		Limit:    12,
		Public:   true,
	})
	if got := ID(packet); got != PacketCZCreateChatRoom {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZCreateChatRoom)
	}
	if got := binary.LittleEndian.Uint16(packet[2:4]); int(got) != len(packet) {
		t.Fatalf("length = %d, want %d", got, len(packet))
	}
	if got := binary.LittleEndian.Uint16(packet[4:6]); got != 12 {
		t.Fatalf("limit = %d, want 12", got)
	}
	if packet[6] != 1 {
		t.Fatalf("type = %d, want public 1", packet[6])
	}
	if got := string(packet[7:13]); got != "secret" || packet[13] != 0 {
		t.Fatalf("password field = %x", packet[7:15])
	}
	if got := string(packet[15:]); got != "Potion shop" {
		t.Fatalf("title = %q", got)
	}
}

func TestBuildCreatePrivateChatRoomPacket(t *testing.T) {
	packet := BuildCreateChatRoomPacket(ChatRoomCreate{Title: "room", Password: "123456789", Limit: 2})
	if packet[6] != 0 {
		t.Fatalf("type = %d, want private 0", packet[6])
	}
	if got := string(packet[7:15]); got != "12345678" {
		t.Fatalf("password field = %q", got)
	}
}

func TestBuildExitChatRoomPacket(t *testing.T) {
	packet := BuildExitChatRoomPacket()
	if got := ID(packet); got != PacketCZExitChatRoom {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZExitChatRoom)
	}
	if len(packet) != 2 {
		t.Fatalf("len = %d, want 2", len(packet))
	}
}

func TestBuildEnterChatRoomPacket(t *testing.T) {
	packet := BuildEnterChatRoomPacket(0x11223344, "secret")
	if got := ID(packet); got != PacketCZEnterChatRoom {
		t.Fatalf("opcode = 0x%04X, want 0x%04X", got, PacketCZEnterChatRoom)
	}
	if len(packet) != 14 {
		t.Fatalf("len = %d, want 14", len(packet))
	}
	if got := binary.LittleEndian.Uint32(packet[2:6]); got != 0x11223344 {
		t.Fatalf("room id = 0x%08X", got)
	}
	if got := string(packet[6:12]); got != "secret" || packet[12] != 0 {
		t.Fatalf("password field = %x", packet[6:14])
	}
}

func TestParseNotifyChat(t *testing.T) {
	packet := Packet{ID: PacketZCNotifyChat, Data: []byte{0x8d, 0x00, 0x11, 0x00, 0x44, 0x33, 0x22, 0x11, 'h', 'i', 0}}
	chat, ok, err := ParseChatMessage(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("chat packet not recognized")
	}
	if chat.GID != 0x11223344 || chat.Text != "hi" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
}

func TestParseBroadcastAnnouncements(t *testing.T) {
	legacy := []byte{0x9a, 0x00, 0x0c, 0x00, 's', 's', 's', 's', 'W', 'o', 'E', 0}
	chat, ok, err := ParseChatMessage(Packet{ID: PacketZCBroadcast, Data: legacy})
	if !ok || err != nil || !chat.Announcement || !chat.HasColor || chat.Text != "WoE" || chat.Color != 0x0000FFFF {
		t.Fatalf("legacy broadcast ok=%t err=%v chat=%+v", ok, err, chat)
	}
	legacy = []byte{0x9a, 0x00, 0x0d, 0x00, 'b', 'l', 'u', 'e', 'I', 'n', 'f', 'o', 0}
	chat, ok, err = ParseChatMessage(Packet{ID: PacketZCBroadcast, Data: legacy})
	if !ok || err != nil || chat.Text != "Info" || chat.Color != 0x00FFFF00 {
		t.Fatalf("blue broadcast ok=%t err=%v chat=%+v", ok, err, chat)
	}

	formatted := make([]byte, 16+8)
	binary.LittleEndian.PutUint16(formatted[0:2], PacketZCBroadcast2)
	binary.LittleEndian.PutUint16(formatted[2:4], uint16(len(formatted)))
	binary.LittleEndian.PutUint32(formatted[4:8], 0x00112233)
	binary.LittleEndian.PutUint16(formatted[8:10], 1)
	binary.LittleEndian.PutUint16(formatted[10:12], 14)
	binary.LittleEndian.PutUint16(formatted[12:14], 2)
	binary.LittleEndian.PutUint16(formatted[14:16], 50)
	copy(formatted[16:], "Castle\x00")
	chat, ok, err = ParseChatMessage(Packet{ID: PacketZCBroadcast2, Data: formatted})
	if !ok || err != nil || !chat.Announcement || chat.Text != "Castle" || chat.Color != 0x00332211 || chat.FontType != 1 || chat.FontSize != 14 || chat.FontAlign != 2 || chat.FontY != 50 {
		t.Fatalf("formatted broadcast ok=%t err=%v chat=%+v", ok, err, chat)
	}
}

func TestParseWhisperMessage(t *testing.T) {
	data := make([]byte, 4+24+6)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCWhisper)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	copy(data[4:28], []byte("Rekka"))
	copy(data[28:], []byte("hello\x00"))
	whisper, ok, err := ParseWhisperMessage(Packet{ID: PacketZCWhisper, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("whisper packet not recognized")
	}
	if whisper.Sender != "Rekka" || whisper.Message != "hello" {
		t.Fatalf("whisper = %+v", whisper)
	}
}

func TestParseWhisperAck(t *testing.T) {
	ack, ok, err := ParseWhisperAck(Packet{ID: PacketZCAckWhisper, Data: []byte{0x98, 0x00, 0x01}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("whisper ack not recognized")
	}
	if ack.Result != 1 {
		t.Fatalf("result = %d, want 1", ack.Result)
	}
}

func TestParseWhisperIgnoreAck(t *testing.T) {
	ack, ok, err := ParseWhisperIgnoreAck(Packet{ID: PacketZCWhisperIgnoreAck, Data: []byte{0xd1, 0x00, 0x00, 0x02}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("whisper ignore ack not recognized")
	}
	if ack.TargetAll || ack.Allow || ack.Result != 2 {
		t.Fatalf("ack = %+v", ack)
	}

	ack, ok, err = ParseWhisperIgnoreAck(Packet{ID: PacketZCWhisperAllAck, Data: []byte{0xd2, 0x00, 0x01, 0x00}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("whisper all ack not recognized")
	}
	if !ack.TargetAll || !ack.Allow || ack.Result != 0 {
		t.Fatalf("ack = %+v", ack)
	}
}

func TestParseChatRoomCreateAck(t *testing.T) {
	ack, ok, err := ParseChatRoomCreateAck(Packet{ID: PacketZCAckCreateChatRoom, Data: []byte{0xd6, 0x00, 0x02}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("chat room ack not recognized")
	}
	if ack.Result != 2 {
		t.Fatalf("result = %d, want 2", ack.Result)
	}
}

func TestParseChatRoomBoardPackets(t *testing.T) {
	boardData := make([]byte, 17+10)
	binary.LittleEndian.PutUint16(boardData[0:2], PacketZCChatRoomBoard)
	binary.LittleEndian.PutUint16(boardData[2:4], uint16(len(boardData)))
	binary.LittleEndian.PutUint32(boardData[4:8], 100)
	binary.LittleEndian.PutUint32(boardData[8:12], 200)
	binary.LittleEndian.PutUint16(boardData[12:14], 12)
	binary.LittleEndian.PutUint16(boardData[14:16], 4)
	boardData[16] = 1
	copy(boardData[17:], []byte("Room title"))
	board, ok, err := ParseChatRoomBoard(Packet{ID: PacketZCChatRoomBoard, Data: boardData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("chat room board not recognized")
	}
	if board.OwnerID != 100 || board.RoomID != 200 || board.Limit != 12 || board.Count != 4 || !board.Public || board.Title != "Room title" {
		t.Fatalf("board = %+v", board)
	}

	destroy, ok, err := ParseChatRoomDestroy(Packet{ID: PacketZCDestroyChatRoom, Data: []byte{0xd8, 0x00, 0xc8, 0x00, 0x00, 0x00}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || destroy.RoomID != 200 {
		t.Fatalf("destroy = %+v ok=%t", destroy, ok)
	}
}

func TestParseChatRoomEnter(t *testing.T) {
	data := make([]byte, 8+28*2)
	binary.LittleEndian.PutUint16(data[0:2], PacketZCChatRoomEnter)
	binary.LittleEndian.PutUint16(data[2:4], uint16(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], 0x10203040)
	binary.LittleEndian.PutUint32(data[8:12], 0)
	copy(data[12:36], []byte("Owner"))
	binary.LittleEndian.PutUint32(data[36:40], 1)
	copy(data[40:64], []byte("Guest"))
	enter, ok, err := ParseChatRoomEnter(Packet{ID: PacketZCChatRoomEnter, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("chat room enter not recognized")
	}
	if enter.RoomID != 0x10203040 || len(enter.Members) != 2 {
		t.Fatalf("enter = %+v", enter)
	}
	if !enter.Members[0].Owner || enter.Members[0].Name != "Owner" || enter.Members[1].Owner || enter.Members[1].Name != "Guest" {
		t.Fatalf("members = %+v", enter.Members)
	}
}

func TestParseChatRoomMemberUpdates(t *testing.T) {
	joinData := make([]byte, 28)
	binary.LittleEndian.PutUint16(joinData[0:2], PacketZCChatRoomJoin)
	binary.LittleEndian.PutUint16(joinData[2:4], 3)
	copy(joinData[4:28], []byte("Alice"))
	join, ok, err := ParseChatRoomMemberJoin(Packet{ID: PacketZCChatRoomJoin, Data: joinData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || join.Count != 3 || join.Name != "Alice" {
		t.Fatalf("join = %+v ok=%t", join, ok)
	}

	leaveData := make([]byte, 29)
	binary.LittleEndian.PutUint16(leaveData[0:2], PacketZCChatRoomLeave)
	binary.LittleEndian.PutUint16(leaveData[2:4], 2)
	copy(leaveData[4:28], []byte("Alice"))
	leaveData[28] = 1
	leave, ok, err := ParseChatRoomMemberLeave(Packet{ID: PacketZCChatRoomLeave, Data: leaveData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || leave.Count != 2 || leave.Name != "Alice" || !leave.Kicked {
		t.Fatalf("leave = %+v ok=%t", leave, ok)
	}
}

func TestParseChatRoomChangeAndRole(t *testing.T) {
	changeData := make([]byte, 17+4)
	binary.LittleEndian.PutUint16(changeData[0:2], PacketZCChatRoomChanged)
	binary.LittleEndian.PutUint16(changeData[2:4], uint16(len(changeData)))
	binary.LittleEndian.PutUint32(changeData[4:8], 100)
	binary.LittleEndian.PutUint32(changeData[8:12], 200)
	binary.LittleEndian.PutUint16(changeData[12:14], 12)
	binary.LittleEndian.PutUint16(changeData[14:16], 4)
	changeData[16] = 1
	copy(changeData[17:], []byte("Room"))
	change, ok, err := ParseChatRoomChange(Packet{ID: PacketZCChatRoomChanged, Data: changeData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || change.OwnerID != 100 || change.RoomID != 200 || change.Limit != 12 || change.Count != 4 || !change.Public || change.Title != "Room" {
		t.Fatalf("change = %+v ok=%t", change, ok)
	}

	roleData := make([]byte, 30)
	binary.LittleEndian.PutUint16(roleData[0:2], PacketZCChatRoomRole)
	binary.LittleEndian.PutUint32(roleData[2:6], 0)
	copy(roleData[6:30], []byte("Owner"))
	role, ok, err := ParseChatRoomRoleChange(Packet{ID: PacketZCChatRoomRole, Data: roleData})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !role.Owner || role.Name != "Owner" {
		t.Fatalf("role = %+v ok=%t", role, ok)
	}
}

func TestParseBroadcastChat(t *testing.T) {
	packet := Packet{ID: PacketZCBroadcast, Data: []byte{0x9a, 0x00, 0x0c, 0x00, 's', 'e', 'r', 'v', 'e', 'r', 0}}
	chat, ok, err := ParseChatMessage(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("broadcast packet not recognized")
	}
	if chat.Text != "server" {
		t.Fatalf("text = %q", chat.Text)
	}
}

func TestParseNPCColorChat(t *testing.T) {
	packet := Packet{ID: PacketZCNPCChat, Data: []byte{
		0xc1, 0x02, 0x1e, 0x00,
		0x44, 0x33, 0x22, 0x11,
		0xb5, 0xff, 0xb5, 0x00,
		'E', 'x', 'p', 'e', 'r', 'i', 'e', 'n', 'c', 'e', ' ', 'G', 'a', 'i', 'n', 'e', 'd', 0,
	}}
	chat, ok, err := ParseChatMessage(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("npc color chat packet not recognized")
	}
	if chat.GID != 0x11223344 || chat.Text != "Experience Gained" || !chat.HasColor || chat.Color != 0x00B5FFB5 {
		t.Fatalf("unexpected npc color chat: %+v", chat)
	}
}

func TestParseMsgStringID(t *testing.T) {
	packet := Packet{ID: PacketZCMsg, Data: []byte{0x91, 0x02, 0x2a, 0x00}}
	chat, ok, err := ParseChatMessage(packet)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("message packet not recognized")
	}
	if chat.MessageID != 42 || chat.Text != "" {
		t.Fatalf("chat = %+v", chat)
	}
}
