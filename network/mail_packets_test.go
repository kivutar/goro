package network

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func mailTestPacket(id uint16, size int) Packet {
	p := Packet{ID: id, Data: make([]byte, size)}
	binary.LittleEndian.PutUint16(p.Data, id)
	if PacketLengths2008()[id] == -1 {
		binary.LittleEndian.PutUint16(p.Data[2:], uint16(size))
	}
	return p
}

func TestMailRequests2008(t *testing.T) {
	id := uint32(0x12345678)
	for _, tc := range []struct {
		name      string
		got, want []byte
	}{
		{"refresh", BuildMailRefreshPacket(), []byte{0x3f, 2}},
		{"read", BuildMailReadPacket(id), []byte{0x41, 2, 0x78, 0x56, 0x34, 0x12}},
		{"delete", BuildMailDeletePacket(id), []byte{0x43, 2, 0x78, 0x56, 0x34, 0x12}},
		{"take", BuildMailGetAttachmentPacket(id), []byte{0x44, 2, 0x78, 0x56, 0x34, 0x12}},
		{"reset all", BuildMailResetPacket(MailResetAll), []byte{0x46, 2, 0, 0}},
		{"reset item", BuildMailResetPacket(MailResetItem), []byte{0x46, 2, 1, 0}},
		{"reset zeny", BuildMailResetPacket(MailResetZeny), []byte{0x46, 2, 2, 0}},
		{"item", BuildMailAddAttachmentPacket(5, 300), []byte{0x47, 2, 5, 0, 44, 1, 0, 0}},
		{"zeny", BuildMailAddAttachmentPacket(0, 100000), []byte{0x47, 2, 0, 0, 0xa0, 0x86, 1, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Equal(tc.got, tc.want) {
				t.Fatalf("got %x, want %x", tc.got, tc.want)
			}
		})
	}
	p := BuildMailReturnPacket(id, "Erika")
	if len(p) != 30 || ID(p) != PacketCZMailReturn || binary.LittleEndian.Uint32(p[2:]) != id || string(p[6:12]) != "Erika\x00" {
		t.Fatalf("return packet = %x", p)
	}
	for _, body := range []string{"", "Hello\nErika", strings.Repeat("x", MailBodyMax), strings.Repeat("é", 99)} {
		p, err := BuildMailSendPacket(" Erika ", " Test ", body)
		if err != nil {
			t.Fatal(err)
		}
		if ID(p) != PacketCZMailSend || len(p) != 69+len(body) || int(binary.LittleEndian.Uint16(p[2:])) != len(p) || int(p[68]) != len(body) || string(p[69:]) != body {
			t.Fatalf("send packet layout: %x", p)
		}
		if string(p[4:10]) != "Erika\x00" || string(p[28:33]) != "Test\x00" {
			t.Fatalf("fixed fields: %x", p)
		}
	}
}

func TestMailSendRejectsInvalidTextWithoutTruncating(t *testing.T) {
	for _, tc := range []struct{ recipient, title, body string }{
		{"", "Title", ""}, {"Erika", "  ", ""},
		{strings.Repeat("n", 24), "Title", ""}, {"Erika", strings.Repeat("t", 40), ""},
		{"Erika", "Title", strings.Repeat("b", 200)}, {"Erika", "Title", strings.Repeat("é", 100)},
		{"Erika\x00Other", "Title", ""}, {"Erika", "Title", "one\x00two"}, {"Erika", "Title", "\xff"},
	} {
		if p, err := BuildMailSendPacket(tc.recipient, tc.title, tc.body); err == nil || p != nil {
			t.Fatalf("accepted invalid mail %+v", tc)
		}
	}
}

func TestMailListEmptyAndFull(t *testing.T) {
	for _, count := range []int{0, 1, MailInboxCapacity} {
		p := mailTestPacket(PacketZCMailList, 8+73*count)
		binary.LittleEndian.PutUint32(p.Data[4:], uint32(count))
		for i := 0; i < count; i++ {
			o := 8 + i*73
			binary.LittleEndian.PutUint32(p.Data[o:], uint32(i+1))
			copy(p.Data[o+4:o+44], "Subject")
			p.Data[o+44] = byte(i % 2)
			copy(p.Data[o+45:o+69], "Sender")
			binary.LittleEndian.PutUint32(p.Data[o+69:], 1234567890)
		}
		entries, ok, err := ParseMailList(p)
		if !ok || err != nil || len(entries) != count {
			t.Fatalf("count=%d: %v %v %+v", count, ok, err, entries)
		}
		for i, e := range entries {
			if e.ID != uint32(i+1) || e.Title != "Subject" || e.Sender != "Sender" || e.Read != (i%2 != 0) || e.Timestamp != 1234567890 {
				t.Fatalf("entry %+v", e)
			}
		}
	}
}

func TestMailReadAttachmentsAndBody(t *testing.T) {
	body := " Hello\nworld! "
	for _, terminator := range []int{0, 1} {
		p := mailTestPacket(PacketZCMailRead, 100+len(body)+terminator)
		binary.LittleEndian.PutUint32(p.Data[4:], 42)
		copy(p.Data[8:48], "Subject")
		copy(p.Data[48:72], "Sender")
		binary.LittleEndian.PutUint32(p.Data[76:], 123456)
		binary.LittleEndian.PutUint32(p.Data[80:], 15)
		binary.LittleEndian.PutUint16(p.Data[84:], 501)
		binary.LittleEndian.PutUint16(p.Data[86:], 4)
		p.Data[88], p.Data[89], p.Data[90] = 1, 1, 7
		for i := 0; i < 4; i++ {
			binary.LittleEndian.PutUint16(p.Data[91+i*2:], uint16(4000+i))
		}
		p.Data[99] = byte(len(body))
		copy(p.Data[100:], body)
		m, ok, err := ParseMailRead(p)
		if !ok || err != nil {
			t.Fatalf("read: %v %v", ok, err)
		}
		if m.ID != 42 || m.Title != "Subject" || m.Sender != "Sender" || m.Body != body || m.Zeny != 123456 || m.Attachment != (MailAttachment{ItemID: 501, Amount: 15, Type: 4, Identified: true, Damaged: true, Refine: 7, Cards: [4]uint16{4000, 4001, 4002, 4003}}) {
			t.Fatalf("message %+v", m)
		}
	}
}

func TestMailNotificationsAndResults(t *testing.T) {
	for kind := uint32(0); kind < 2; kind++ {
		p := mailTestPacket(PacketZCMailWindow, 6)
		binary.LittleEndian.PutUint32(p.Data[2:], kind)
		open, ok, err := ParseMailWindow(p)
		if !ok || err != nil || open != (kind == 0) {
			t.Fatalf("window=%d: %v %v %v", kind, open, ok, err)
		}
	}
	for _, id := range []uint16{PacketZCMailGetAttachment, PacketZCMailSend, PacketZCMailAddAttachment, PacketZCMailDelete, PacketZCMailReturn} {
		for result := uint16(0); result < 3; result++ {
			p := mailTestPacket(id, PacketLengths2008()[id])
			want := MailResult{Result: result}
			switch len(p.Data) {
			case 3:
				p.Data[2] = byte(result)
			case 5:
				want.Index = 7
				binary.LittleEndian.PutUint16(p.Data[2:], 7)
				p.Data[4] = byte(result)
			case 8:
				want.ID = 42
				binary.LittleEndian.PutUint32(p.Data[2:], 42)
				binary.LittleEndian.PutUint16(p.Data[6:], result)
			}
			got, ok, err := ParseMailResult(p)
			if !ok || err != nil || got != want {
				t.Fatalf("ack %04x: %+v %v %v", id, got, ok, err)
			}
		}
	}
	p := mailTestPacket(PacketZCMailNew, 70)
	binary.LittleEndian.PutUint32(p.Data[2:], 42)
	copy(p.Data[6:46], "Subject")
	copy(p.Data[46:70], "Sender")
	e, ok, err := ParseMailNew(p)
	if !ok || err != nil || e.ID != 42 || e.Title != "Subject" || e.Sender != "Sender" {
		t.Fatalf("new: %+v %v %v", e, ok, err)
	}
}

func TestMailFramingAndMalformedPackets(t *testing.T) {
	parsers := map[uint16]func(Packet) (bool, error){
		PacketZCMailList:   func(p Packet) (bool, error) { _, ok, err := ParseMailList(p); return ok, err },
		PacketZCMailRead:   func(p Packet) (bool, error) { _, ok, err := ParseMailRead(p); return ok, err },
		PacketZCMailWindow: func(p Packet) (bool, error) { _, ok, err := ParseMailWindow(p); return ok, err },
		PacketZCMailNew:    func(p Packet) (bool, error) { _, ok, err := ParseMailNew(p); return ok, err },
	}
	for _, id := range []uint16{PacketZCMailGetAttachment, PacketZCMailSend, PacketZCMailAddAttachment, PacketZCMailDelete, PacketZCMailReturn} {
		parsers[id] = func(p Packet) (bool, error) { _, ok, err := ParseMailResult(p); return ok, err }
	}
	for id, parse := range parsers {
		size := PacketLengths2008()[id]
		if id == PacketZCMailList {
			size = 8
		}
		if id == PacketZCMailRead {
			size = 101
		}
		p := mailTestPacket(id, size)
		for n := 0; n < size-1; n++ {
			if ok, err := parse(Packet{ID: id, Data: p.Data[:n]}); ok || err == nil {
				t.Fatalf("accepted truncated %04x at %d", id, n)
			}
		}
		if ok, err := parse(Packet{ID: 0xffff}); ok || err != nil {
			t.Fatalf("unrelated packet: %v %v", ok, err)
		}
		framer := NewFramer(PacketLengths2008())
		for i, b := range p.Data {
			got, err := framer.Push([]byte{b})
			if err != nil {
				t.Fatal(err)
			}
			if i < size-1 && len(got) != 0 {
				t.Fatal("framed partial mail")
			}
			if i == size-1 && (len(got) != 1 || !bytes.Equal(got[0].Data, p.Data)) {
				t.Fatal("mail not framed")
			}
		}
	}
	p := mailTestPacket(PacketZCMailList, 8)
	binary.LittleEndian.PutUint32(p.Data[4:], 0xffffffff)
	if _, ok, err := ParseMailList(p); ok || err == nil {
		t.Fatal("accepted overflowing list count")
	}
	p = mailTestPacket(PacketZCMailRead, 101)
	p.Data[99] = 199
	if _, ok, err := ParseMailRead(p); ok || err == nil {
		t.Fatal("accepted truncated body")
	}
}
