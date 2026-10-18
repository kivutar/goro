package network

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	PacketCAPlainLogin     uint16 = 0x0064
	PacketCAEnter          uint16 = 0x0065
	PacketCZSelectChar     uint16 = 0x0066
	PacketCHMakeChar       uint16 = 0x0067
	PacketCHDeleteCharOld  uint16 = 0x0068
	PacketCHDeleteChar     uint16 = 0x01FB
	PacketCZLoadEndAck     uint16 = 0x007D
	PacketCZTickSend       uint16 = 0x0089
	PacketCZTickSendRE     uint16 = 0x0360
	PacketCZReqNameRE      uint16 = 0x0368
	PacketCZWalkToXY       uint16 = 0x00A7
	PacketCZWalkToXYRE     uint16 = 0x035F
	PacketCZChangeDir      uint16 = 0x009B
	PacketCZChangeDirRE    uint16 = 0x0361
	PacketCZRestart        uint16 = 0x00B2
	PacketPing             uint16 = 0x0187
	PacketCZQuitGame       uint16 = 0x018A
	PacketCZRequestAct     uint16 = 0x0190
	PacketCZRequestAct2    uint16 = 0x0437
	PacketCZRequestAct2012 uint16 = 0x0369
	PacketCZEnter2         uint16 = 0x0436
)

const PacketCAConnectInfoChanged uint16 = 0x0200

type nameRequestPacketLayout struct {
	date   int
	opcode uint16
	length int
	offset int
}

type changeDirectionPacketLayout struct {
	date          int
	opcode        uint16
	length        int
	headDirOffset int
	dirOffset     int
}

var nameRequestPacketLayouts = []nameRequestPacketLayout{
	// Keep this table aligned with rAthena's active packetdb branch. Our
	// default 20080910 pre-renewal server uses PACKETVER_MAIN_NUM, not the
	// 20080827 RagexeRE block from reference client's cumulative table.
	{date: 20101124, opcode: PacketCZReqNameRE, length: 6, offset: 2},
	{date: 20070212, opcode: 0x008C, length: 11, offset: 7},
}

var changeDirectionPacketLayouts = []changeDirectionPacketLayout{
	// For the default 20080910 pre-renewal profile, the last active main-client
	// remap is the 20070212 shuffled 0x0085 packet. Sending the unshuffled
	// 0x009B packet disconnects because that opcode has been remapped.
	{date: 20101124, opcode: PacketCZChangeDirRE, length: 5, headDirOffset: 2, dirOffset: 4},
	{date: 20070212, opcode: 0x0085, length: 11, headDirOffset: 7, dirOffset: 10},
	{date: 20070108, opcode: 0x0085, length: 14, headDirOffset: 10, dirOffset: 13},
	{date: 20050110, opcode: 0x0085, length: 23, headDirOffset: 12, dirOffset: 22},
	{date: 20041129, opcode: 0x00F3, length: 8, headDirOffset: 3, dirOffset: 7},
}

const (
	ActionSitDown uint8 = 2
	ActionStandUp uint8 = 3
	ActionAttack  uint8 = 7
)

const (
	RestartTypeRespawn    uint8 = 0
	RestartTypeCharSelect uint8 = 1
)

type AccountLogin struct {
	Version    uint32
	Username   string
	Password   string
	ClientType uint8
}

func BuildAccountLoginPacket(login AccountLogin) []byte {
	var w Writer
	w.Uint16(PacketCAPlainLogin)
	w.Uint32(login.Version)
	w.CString(login.Username, 24)
	w.CString(login.Password, 24)
	w.Uint8(login.ClientType)
	return w.Bytes()
}

type CharServerEnter struct {
	AccountID uint32
	AuthCode  uint32
	UserLevel uint32
	Sex       uint8
}

func BuildCharServerEnterPacket(enter CharServerEnter) []byte {
	var w Writer
	w.Uint16(PacketCAEnter)
	w.Uint32(enter.AccountID)
	w.Uint32(enter.AuthCode)
	w.Uint32(enter.UserLevel)
	w.Uint16(0)
	w.Uint8(enter.Sex)
	return w.Bytes()
}

func BuildLoginServerKeepalivePacket(username string) []byte {
	var w Writer
	w.Uint16(PacketCAConnectInfoChanged)
	w.CString(username, 24)
	return w.Bytes()
}

func BuildSelectCharacterPacket(slot uint8) []byte {
	var w Writer
	w.Uint16(PacketCZSelectChar)
	w.Uint8(slot)
	return w.Bytes()
}

type MakeCharacter struct {
	Name      string
	Str       uint8
	Agi       uint8
	Vit       uint8
	Int       uint8
	Dex       uint8
	Luk       uint8
	Slot      uint8
	HairColor uint16
	HairStyle uint16
}

func BuildMakeCharacterPacket(character MakeCharacter) []byte {
	var w Writer
	w.Uint16(PacketCHMakeChar)
	w.CString(character.Name, 24)
	w.Uint8(character.Str)
	w.Uint8(character.Agi)
	w.Uint8(character.Vit)
	w.Uint8(character.Int)
	w.Uint8(character.Dex)
	w.Uint8(character.Luk)
	w.Uint8(character.Slot)
	w.Uint16(character.HairColor)
	w.Uint16(character.HairStyle)
	return w.Bytes()
}

func BuildDeleteCharacterPacket(clientDate int, charID uint32, key string) []byte {
	var w Writer
	if clientDate >= 20040419 {
		w.Uint16(PacketCHDeleteChar)
		w.Uint32(charID)
		w.CString(key, 50)
		return w.Bytes()
	}
	w.Uint16(PacketCHDeleteCharOld)
	w.Uint32(charID)
	w.CString(key, 40)
	return w.Bytes()
}

func BuildLoadEndAckPacket() []byte {
	var w Writer
	w.Uint16(PacketCZLoadEndAck)
	return w.Bytes()
}

func BuildRestartPacket(restartType uint8) []byte {
	var w Writer
	w.Uint16(PacketCZRestart)
	w.Uint8(restartType)
	return w.Bytes()
}

func BuildQuitGamePacket() []byte {
	var w Writer
	w.Uint16(PacketCZQuitGame)
	w.Uint16(0)
	return w.Bytes()
}

func BuildPingPacket(accountID uint32) []byte {
	var w Writer
	w.Uint16(PacketPing)
	w.Uint32(accountID)
	return w.Bytes()
}

func BuildTickSendPacketForClientDate(clientTick uint32, clientDate int) []byte {
	var w Writer
	if clientDate > 20180307 {
		w.Uint16(PacketCZTickSendRE)
		w.Uint32(clientTick)
	} else {
		w.Uint16(PacketCZTickSend)
		w.Uint8(0)
		w.Uint8(0)
		w.Uint32(clientTick)
	}
	return w.Bytes()
}

func BuildNameRequestPacketForClientDate(gid uint32, clientDate int) ([]byte, bool) {
	for _, layout := range nameRequestPacketLayouts {
		if clientDate >= layout.date {
			packet := make([]byte, layout.length)
			binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
			binary.LittleEndian.PutUint32(packet[layout.offset:layout.offset+4], gid)
			return packet, true
		}
	}
	return nil, false
}

type MapServerEnter struct {
	AccountID  uint32
	CharID     uint32
	AuthCode   uint32
	ClientTick uint32
	Sex        uint8
}

func BuildMapServerEnterPacketForClientDate(enter MapServerEnter, clientDate int) []byte {
	var w Writer
	w.Uint16(PacketCZEnter2)
	w.Uint32(enter.AccountID)
	w.Uint32(enter.CharID)
	w.Uint32(enter.AuthCode)
	w.Uint32(enter.ClientTick)
	if clientDate >= 20211103 {
		w.Uint32(0)
	}
	w.Uint8(enter.Sex)
	return w.Bytes()
}

func BuildWalkToXYPacketForClientDate(x, y, clientDate int) ([]byte, bool) {
	dest, ok := EncodeMoveDestination(x, y)
	if !ok {
		return nil, false
	}

	var w Writer
	if clientDate >= 20211103 {
		w.Uint16(PacketCZWalkToXYRE)
	} else {
		w.Uint16(PacketCZWalkToXY)
		w.Uint8(0)
		w.Uint8(0)
		w.Uint8(0)
	}
	_, _ = w.Write(dest[:])
	return w.Bytes(), true
}

func BuildChangeDirectionPacketForClientDate(headDir, dir uint8, clientDate int) []byte {
	for _, layout := range changeDirectionPacketLayouts {
		if clientDate >= layout.date {
			packet := make([]byte, layout.length)
			binary.LittleEndian.PutUint16(packet[0:2], layout.opcode)
			if layout.headDirOffset > 2 {
				fillLegacyPacketPadding(packet[2:layout.headDirOffset])
			}
			binary.LittleEndian.PutUint16(packet[layout.headDirOffset:layout.headDirOffset+2], uint16(headDir))
			if layout.dirOffset > layout.headDirOffset+2 {
				fillLegacyPacketPadding(packet[layout.headDirOffset+2 : layout.dirOffset])
			}
			packet[layout.dirOffset] = dir & 7
			return packet
		}
	}
	var w Writer
	w.Uint16(PacketCZChangeDir)
	w.Uint16(uint16(headDir))
	w.Uint8(dir & 7)
	return w.Bytes()
}

func BuildActionRequestPacketForClientDate(targetGID uint32, action uint8, clientDate int) []byte {
	switch {
	case clientDate >= 20211103:
		return buildCompactActionRequestPacket(PacketCZRequestAct2, targetGID, action)
	case clientDate >= 20120410:
		return buildCompactActionRequestPacket(PacketCZRequestAct2012, targetGID, action)
	default:
		return buildLegacyActionRequestPacket(targetGID, action)
	}
}

func buildCompactActionRequestPacket(opcode uint16, targetGID uint32, action uint8) []byte {
	var w Writer
	w.Uint16(opcode)
	w.Uint32(targetGID)
	w.Uint8(action)
	return w.Bytes()
}

func buildLegacyActionRequestPacket(targetGID uint32, action uint8) []byte {
	packet := make([]byte, 19)
	binary.LittleEndian.PutUint16(packet[0:2], PacketCZRequestAct)
	fillLegacyPacketPadding(packet[2:5])
	binary.LittleEndian.PutUint32(packet[5:9], targetGID)
	fillLegacyPacketPadding(packet[9:18])
	packet[18] = action
	return packet
}

func fillLegacyPacketPadding(pad []byte) {
	if len(pad) == 0 {
		return
	}
	now := uint32(time.Now().UnixMilli())
	text := fmt.Sprintf("%x", now^(now>>uint(len(pad))))
	if text == "" {
		text = "0"
	}
	src := len(text) - 1
	fallback := text[len(text)-1]
	for i := range pad {
		if src >= 0 {
			pad[i] = text[src]
			src--
		} else {
			pad[i] = fallback
		}
	}
	pad[len(pad)-1] = 0
}

func EncodeMoveDestination(x, y int) ([3]byte, bool) {
	if x < 0 || x > 1023 || y < 0 || y > 1023 {
		return [3]byte{}, false
	}
	return [3]byte{
		byte((x >> 2) & 0xff),
		byte(((x & 0x03) << 6) | ((y >> 4) & 0x3f)),
		byte((y & 0x0f) << 4),
	}, true
}
