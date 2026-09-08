package network

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

// These are the pre-RODEX packets used by the 2008 client. Legacy mail has
// one item-stack attachment and Zeny, and a single-byte body length.
const (
	PacketCZMailRefresh       uint16 = 0x023F
	PacketZCMailList          uint16 = 0x0240
	PacketCZMailRead          uint16 = 0x0241
	PacketZCMailRead          uint16 = 0x0242
	PacketCZMailDelete        uint16 = 0x0243
	PacketCZMailGetAttachment uint16 = 0x0244
	PacketZCMailGetAttachment uint16 = 0x0245
	PacketCZMailReset         uint16 = 0x0246
	PacketCZMailAddAttachment uint16 = 0x0247
	PacketCZMailSend          uint16 = 0x0248
	PacketZCMailSend          uint16 = 0x0249
	PacketZCMailNew           uint16 = 0x024A
	PacketZCMailAddAttachment uint16 = 0x0255
	PacketZCMailDelete        uint16 = 0x0257
	PacketZCMailWindow        uint16 = 0x0260
	PacketCZMailReturn        uint16 = 0x0273
	PacketZCMailReturn        uint16 = 0x0274

	MailInboxCapacity = 30
	MailRecipientMax  = 23
	MailTitleMax      = 39
	MailBodyMax       = 199
)

type MailResetType uint16

const (
	MailResetAll MailResetType = iota
	MailResetItem
	MailResetZeny
)

type MailEntry struct {
	ID        uint32
	Title     string
	Read      bool
	Sender    string
	Timestamp uint32
}

type MailAttachment struct {
	Amount     uint32
	ItemID     uint16
	Type       uint16
	Identified bool
	Damaged    bool
	Refine     uint8
	Cards      [4]uint16
}

type MailMessage struct {
	ID         uint32
	Title      string
	Sender     string
	Body       string
	Zeny       uint32
	Attachment MailAttachment
}

func (m MailMessage) HasAttachment() bool {
	return m.Zeny > 0 || m.Attachment.ItemID != 0 && m.Attachment.Amount > 0
}

// MailResult covers the legacy acknowledgements. Get-attachment and send
// acknowledgements carry no ID; the caller must correlate outstanding work.
type MailResult struct {
	ID     uint32
	Index  uint16
	Result uint16
}

func ParseMailWindow(packet Packet) (bool, bool, error) {
	if packet.ID != PacketZCMailWindow {
		return false, false, nil
	}
	if len(packet.Data) != 6 {
		return false, false, fmt.Errorf("ZC_MAIL_WINDOWS length %d, want 6", len(packet.Data))
	}
	kind := binary.LittleEndian.Uint32(packet.Data[2:6])
	if kind > 1 {
		return false, false, fmt.Errorf("unknown mail window type %d", kind)
	}
	return kind == 0, true, nil
}

func ParseMailList(packet Packet) ([]MailEntry, bool, error) {
	if packet.ID != PacketZCMailList {
		return nil, false, nil
	}
	p := packet.Data
	if len(p) < 8 || int(binary.LittleEndian.Uint16(p[2:4])) != len(p) {
		return nil, false, fmt.Errorf("invalid ZC_MAIL_REQ_GET_LIST length %d", len(p))
	}
	count := binary.LittleEndian.Uint32(p[4:8])
	if count > MailInboxCapacity || len(p) != 8+int(count)*73 {
		return nil, false, fmt.Errorf("invalid mail list count %d for length %d", count, len(p))
	}
	entries := make([]MailEntry, count)
	for i := range entries {
		o := 8 + i*73
		entries[i] = MailEntry{
			ID: binary.LittleEndian.Uint32(p[o : o+4]), Title: packetCString(p[o+4 : o+44]),
			Read: p[o+44] != 0, Sender: packetCString(p[o+45 : o+69]),
			Timestamp: binary.LittleEndian.Uint32(p[o+69 : o+73]),
		}
	}
	return entries, true, nil
}

func ParseMailRead(packet Packet) (MailMessage, bool, error) {
	if packet.ID != PacketZCMailRead {
		return MailMessage{}, false, nil
	}
	p := packet.Data
	if len(p) < 100 || int(binary.LittleEndian.Uint16(p[2:4])) != len(p) {
		return MailMessage{}, false, fmt.Errorf("invalid ZC_MAIL_REQ_OPEN length %d", len(p))
	}
	bodyLen := int(p[99])
	// rAthena appends a NUL after the advertised body. Also accept the exact
	// body-length form; never infer the length from an untrusted C string.
	if bodyLen > MailBodyMax || (len(p) != 100+bodyLen && len(p) != 101+bodyLen) {
		return MailMessage{}, false, fmt.Errorf("invalid mail body length %d for packet %d", bodyLen, len(p))
	}
	attachment := MailAttachment{
		Amount: binary.LittleEndian.Uint32(p[80:84]), ItemID: binary.LittleEndian.Uint16(p[84:86]),
		Type: binary.LittleEndian.Uint16(p[86:88]), Identified: p[88] != 0, Damaged: p[89] != 0, Refine: p[90],
	}
	for i := range attachment.Cards {
		attachment.Cards[i] = binary.LittleEndian.Uint16(p[91+i*2 : 93+i*2])
	}
	return MailMessage{
		ID: binary.LittleEndian.Uint32(p[4:8]), Title: packetCString(p[8:48]),
		Sender: packetCString(p[48:72]), Zeny: binary.LittleEndian.Uint32(p[76:80]),
		Body: strings.TrimRight(string(p[100:100+bodyLen]), "\x00"), Attachment: attachment,
	}, true, nil
}

func ParseMailNew(packet Packet) (MailEntry, bool, error) {
	if packet.ID != PacketZCMailNew {
		return MailEntry{}, false, nil
	}
	if len(packet.Data) != 70 {
		return MailEntry{}, false, fmt.Errorf("ZC_MAIL_RECEIVE length %d, want 70", len(packet.Data))
	}
	return MailEntry{
		ID:    binary.LittleEndian.Uint32(packet.Data[2:6]),
		Title: packetCString(packet.Data[6:46]), Sender: packetCString(packet.Data[46:70]),
	}, true, nil
}

func ParseMailResult(packet Packet) (MailResult, bool, error) {
	p := packet.Data
	want := 0
	switch packet.ID {
	case PacketZCMailGetAttachment, PacketZCMailSend:
		want = 3
	case PacketZCMailAddAttachment:
		want = 5
	case PacketZCMailDelete, PacketZCMailReturn:
		want = 8
	default:
		return MailResult{}, false, nil
	}
	if len(p) != want {
		return MailResult{}, false, fmt.Errorf("mail result 0x%04X length %d, want %d", packet.ID, len(p), want)
	}
	switch want {
	case 3:
		return MailResult{Result: uint16(p[2])}, true, nil
	case 5:
		return MailResult{Index: binary.LittleEndian.Uint16(p[2:4]), Result: uint16(p[4])}, true, nil
	default:
		return MailResult{ID: binary.LittleEndian.Uint32(p[2:6]), Result: binary.LittleEndian.Uint16(p[6:8])}, true, nil
	}
}

func BuildMailRefreshPacket() []byte {
	return []byte{0x3f, 0x02}
}

func buildMailIDPacket(opcode uint16, id uint32) []byte {
	p := make([]byte, 6)
	binary.LittleEndian.PutUint16(p, opcode)
	binary.LittleEndian.PutUint32(p[2:], id)
	return p
}

func BuildMailReadPacket(id uint32) []byte   { return buildMailIDPacket(PacketCZMailRead, id) }
func BuildMailDeletePacket(id uint32) []byte { return buildMailIDPacket(PacketCZMailDelete, id) }
func BuildMailGetAttachmentPacket(id uint32) []byte {
	return buildMailIDPacket(PacketCZMailGetAttachment, id)
}

func BuildMailResetPacket(kind MailResetType) []byte {
	p := make([]byte, 4)
	binary.LittleEndian.PutUint16(p, PacketCZMailReset)
	binary.LittleEndian.PutUint16(p[2:], uint16(kind))
	return p
}

func BuildMailAddAttachmentPacket(index uint16, amount uint32) []byte {
	p := make([]byte, 8)
	binary.LittleEndian.PutUint16(p, PacketCZMailAddAttachment)
	binary.LittleEndian.PutUint16(p[2:], index)
	binary.LittleEndian.PutUint32(p[4:], amount)
	return p
}

func BuildMailReturnPacket(id uint32, sender string) []byte {
	p := make([]byte, 30)
	binary.LittleEndian.PutUint16(p, PacketCZMailReturn)
	binary.LittleEndian.PutUint32(p[2:], id)
	writeFixedName(p[6:], sender)
	return p
}

func BuildMailSendPacket(recipient, title, body string) ([]byte, error) {
	recipient, title = strings.TrimSpace(recipient), strings.TrimSpace(title)
	for _, field := range []struct {
		name, value string
		max         int
		required    bool
	}{{"recipient", recipient, MailRecipientMax, true}, {"subject", title, MailTitleMax, true}, {"message", body, MailBodyMax, false}} {
		if field.required && field.value == "" {
			return nil, fmt.Errorf("%s is required", field.name)
		}
		if len(field.value) > field.max {
			return nil, fmt.Errorf("%s is limited to %d bytes", field.name, field.max)
		}
		if strings.ContainsRune(field.value, 0) || !utf8.ValidString(field.value) {
			return nil, fmt.Errorf("%s contains invalid text", field.name)
		}
	}
	p := make([]byte, 69+len(body))
	binary.LittleEndian.PutUint16(p, PacketCZMailSend)
	binary.LittleEndian.PutUint16(p[2:], uint16(len(p)))
	writeFixedCString(p[4:28], recipient)
	writeFixedCString(p[28:68], title)
	p[68] = byte(len(body))
	copy(p[69:], body)
	return p, nil
}

func (c *Client) SendMailRefresh() error         { return c.Send(BuildMailRefreshPacket()) }
func (c *Client) SendMailRead(id uint32) error   { return c.Send(BuildMailReadPacket(id)) }
func (c *Client) SendMailDelete(id uint32) error { return c.Send(BuildMailDeletePacket(id)) }
func (c *Client) SendMailGetAttachment(id uint32) error {
	return c.Send(BuildMailGetAttachmentPacket(id))
}
func (c *Client) SendMailReset(kind MailResetType) error { return c.Send(BuildMailResetPacket(kind)) }
func (c *Client) SendMailAddAttachment(index uint16, amount uint32) error {
	return c.Send(BuildMailAddAttachmentPacket(index, amount))
}
func (c *Client) SendMailReturn(id uint32, sender string) error {
	return c.Send(BuildMailReturnPacket(id, sender))
}
func (c *Client) SendMail(recipient, title, body string) error {
	p, err := BuildMailSendPacket(recipient, title, body)
	if err != nil {
		return err
	}
	return c.Send(p)
}
