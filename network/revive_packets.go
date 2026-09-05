package network

import "encoding/binary"

const PacketCZAutoRevive uint16 = 0x0292

func BuildAutoRevivePacket() []byte {
	packet := make([]byte, 2)
	binary.LittleEndian.PutUint16(packet, PacketCZAutoRevive)
	return packet
}

func (c *Client) SendAutoRevive() error {
	return c.Send(BuildAutoRevivePacket())
}
