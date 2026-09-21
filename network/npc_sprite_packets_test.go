package network

import (
	"encoding/binary"
	"reflect"
	"testing"
)

func npcSpriteChangeTestPacket(id, job uint32, packetType byte) Packet {
	data := make([]byte, 11)
	binary.LittleEndian.PutUint16(data, 0x01B0)
	binary.LittleEndian.PutUint32(data[2:], id)
	data[6] = packetType
	binary.LittleEndian.PutUint32(data[7:], job)
	return Packet{ID: 0x01B0, Data: data}
}

func TestParseNPCSpriteChange(t *testing.T) {
	for _, packetType := range []byte{0, 1, 2, 255} {
		packet := npcSpriteChangeTestPacket(110000001, 0x12345678, packetType)
		change, ok, err := ParseNPCSpriteChange(packet)
		if err != nil || !ok || change.ID != 110000001 || change.Job != 0x12345678 {
			t.Fatalf("type %d: change=%+v ok=%t err=%v", packetType, change, ok, err)
		}
		for length := 0; length < 11; length++ {
			if _, ok, err := ParseNPCSpriteChange(Packet{ID: packet.ID, Data: packet.Data[:length]}); err == nil || ok {
				t.Fatalf("accepted truncated packet of length %d", length)
			}
		}
	}
	if _, ok, err := ParseNPCSpriteChange(npcSpriteChangeTestPacket(1, 1018, 0)); err != nil || !ok {
		t.Fatalf("Creamy transformation: ok=%t err=%v", ok, err)
	}
	if _, ok, err := ParseNPCSpriteChange(Packet{ID: 0x01D7}); err != nil || ok {
		t.Fatal("NPC sprite parser consumed an ordinary appearance change")
	}
}

func TestFrameNPCTransformationsAcrossReadBoundaries(t *testing.T) {
	var want []Packet
	var stream []byte
	for i := uint32(0); i < 64; i++ {
		// The ID deliberately contains another valid opcode (0x0095).
		id := uint32(0x00009500) + i
		transform := npcSpriteChangeTestPacket(id, 1018, byte(i%2))
		name := make([]byte, 30)
		binary.LittleEndian.PutUint16(name, 0x0095)
		binary.LittleEndian.PutUint32(name[2:], id)
		copy(name[6:], "Creamy")
		want = append(want, transform, Packet{ID: 0x0095, Data: name})
		stream = append(stream, transform.Data...)
		stream = append(stream, name...)
	}
	for _, chunkSize := range []int{1, 5, 11, 40, len(stream)} {
		framer := NewFramer(PacketLengths2008())
		var got []Packet
		for offset := 0; offset < len(stream); offset += chunkSize {
			packets, err := framer.Push(stream[offset:min(offset+chunkSize, len(stream))])
			if err != nil {
				t.Fatalf("chunk=%d offset=%d: %v", chunkSize, offset, err)
			}
			got = append(got, packets...)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("chunk=%d: packet stream changed; received %d packets, want %d", chunkSize, len(got), len(want))
		}
	}
}
