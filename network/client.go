package network

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/kivutar/goro/glog"
	"io"
	"net"
	"sync"
	"time"
)

var ErrDisconnected = errors.New("disconnected from server")

type FrameError struct {
	Err error
}

func (e FrameError) Error() string {
	if e.Err == nil {
		return "frame error"
	}
	return e.Err.Error()
}

func (e FrameError) Unwrap() error {
	return e.Err
}

type Client struct {
	clientDate int
	trace      bool
	framer     *Framer

	mu      sync.Mutex
	conn    net.Conn
	sendCh  chan outboundPacket
	packets []Packet
	status  string
	errs    []error
}

const sendQueueSize = 256

type outboundPacket struct {
	data     []byte
	enqueued time.Time
}

func NewClient(clientDate int, trace bool) *Client {
	return &Client{
		clientDate: clientDate,
		trace:      trace,
		framer:     NewFramer(PacketLengths2008()),
		status:     "offline",
	}
}

func (c *Client) Connect(ctx context.Context, address string, port int) error {
	c.Close()

	dialer := net.Dialer{Timeout: 5 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		c.setStatus("connect failed")
		return err
	}

	sendCh := make(chan outboundPacket, sendQueueSize)

	c.mu.Lock()
	c.conn = conn
	c.sendCh = sendCh
	c.status = fmt.Sprintf("connected to %s:%d", address, port)
	c.mu.Unlock()

	go c.readLoop(conn)
	go c.writeLoop(conn, sendCh)
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	conn := c.conn
	sendCh := c.sendCh
	c.conn = nil
	c.sendCh = nil
	c.status = "offline"
	if sendCh != nil {
		close(sendCh)
	}
	c.mu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (c *Client) Send(data []byte) error {
	packet := append([]byte(nil), data...)
	start := time.Now()
	c.mu.Lock()
	sendCh := c.sendCh
	if c.conn == nil || sendCh == nil {
		c.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	select {
	case sendCh <- outboundPacket{data: packet, enqueued: start}:
	default:
		c.mu.Unlock()
		return fmt.Errorf("send queue full")
	}
	c.mu.Unlock()
	elapsed := time.Since(start)

	if c.trace || elapsed > 2*time.Millisecond {
		headLen := min(len(packet), 32)
		glog.Debugf("network enqueue opcode=0x%04X n=%d elapsed=%s head=%s", ID(packet), len(packet), elapsed, hex.EncodeToString(packet[:headLen]))
	}
	return nil
}

func (c *Client) SendAccountLogin(username, password string, version uint32, clientType uint8) error {
	packet := BuildAccountLoginPacket(AccountLogin{
		Version:    version,
		Username:   username,
		Password:   password,
		ClientType: clientType,
	})
	return c.Send(packet)
}

func (c *Client) SendCharServerEnter(accountID, authCode, userLevel uint32, sex uint8) error {
	packet := BuildCharServerEnterPacket(CharServerEnter{
		AccountID: accountID,
		AuthCode:  authCode,
		UserLevel: userLevel,
		Sex:       sex,
	})
	return c.Send(packet)
}

func (c *Client) SendSelectCharacter(slot uint8) error {
	return c.Send(BuildSelectCharacterPacket(slot))
}

func (c *Client) SendMakeCharacter(character MakeCharacter) error {
	packet := BuildMakeCharacterPacket(character)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CH_MAKE_CHAR opcode=0x%04X name=%q slot=%d stats=%d/%d/%d/%d/%d/%d hair_color=%d hair_style=%d client_date=%d",
			ID(packet), character.Name, character.Slot, character.Str, character.Agi, character.Vit, character.Int, character.Dex, character.Luk, character.HairColor, character.HairStyle, c.clientDate)
	} else {
		glog.Warnf("send CH_MAKE_CHAR failed opcode=0x%04X len=%d name=%q slot=%d client_date=%d: %v",
			ID(packet), len(packet), character.Name, character.Slot, c.clientDate, err)
	}
	return err
}

func (c *Client) SendDeleteCharacter(charID uint32, key string) error {
	packet := BuildDeleteCharacterPacket(c.clientDate, charID, key)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CH_DELETE_CHAR opcode=0x%04X char_id=%d client_date=%d", ID(packet), charID, c.clientDate)
	} else {
		glog.Warnf("send CH_DELETE_CHAR failed opcode=0x%04X len=%d char_id=%d client_date=%d: %v",
			ID(packet), len(packet), charID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendLoadEndAck() error {
	return c.Send(BuildLoadEndAckPacket())
}

func (c *Client) SendRestart(restartType uint8) error {
	packet := BuildRestartPacket(restartType)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_RESTART opcode=0x%04X type=%d client_date=%d", ID(packet), restartType, c.clientDate)
	} else {
		glog.Warnf("send CZ_RESTART failed opcode=0x%04X len=%d type=%d client_date=%d: %v", ID(packet), len(packet), restartType, c.clientDate, err)
	}
	return err
}

func (c *Client) SendQuitGame() error {
	packet := BuildQuitGamePacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_DISCONNECT opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_DISCONNECT failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendPing(accountID uint32) error {
	packet := BuildPingPacket(accountID)
	err := c.Send(packet)
	if err == nil && c.trace {
		glog.Debugf("sent PING opcode=0x%04X account_id=%d client_date=%d", ID(packet), accountID, c.clientDate)
	} else if err != nil {
		glog.Warnf("send PING failed opcode=0x%04X len=%d account_id=%d client_date=%d: %v", ID(packet), len(packet), accountID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendTick(clientTick uint32) error {
	packet := BuildTickSendPacketForClientDate(clientTick, c.clientDate)
	err := c.Send(packet)
	if err == nil && c.trace {
		glog.Debugf("sent CZ_REQUEST_TIME opcode=0x%04X tick=%d client_date=%d", ID(packet), clientTick, c.clientDate)
	} else if err != nil {
		glog.Warnf("send CZ_REQUEST_TIME failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendNameRequest(gid uint32) error {
	packet, ok := BuildNameRequestPacketForClientDate(gid, c.clientDate)
	if !ok {
		glog.Warnf("skip CZ_REQNAME target=%d client_date=%d: unsupported packet profile", gid, c.clientDate)
		return nil
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQNAME opcode=0x%04X len=%d target=%d client_date=%d", ID(packet), len(packet), gid, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQNAME failed opcode=0x%04X len=%d target=%d client_date=%d: %v", ID(packet), len(packet), gid, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMapServerEnter(accountID, charID, authCode, clientTick uint32, sex uint8) error {
	packet := BuildMapServerEnterPacketForClientDate(MapServerEnter{
		AccountID:  accountID,
		CharID:     charID,
		AuthCode:   authCode,
		ClientTick: clientTick,
		Sex:        sex,
	}, c.clientDate)
	if c.trace {
		glog.Debugf("sent CZ_ENTER2 opcode=0x%04X len=%d client_date=%d sex_offset=%d", ID(packet), len(packet), c.clientDate, len(packet)-1)
	}
	return c.Send(packet)
}

func (c *Client) SendWalkToXY(x, y int) error {
	packet, ok := BuildWalkToXYPacketForClientDate(x, y, c.clientDate)
	if !ok {
		return fmt.Errorf("invalid walk destination %d,%d", x, y)
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_MOVE opcode=0x%04X dst=%d,%d client_date=%d", ID(packet), x, y, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_MOVE failed opcode=0x%04X len=%d dst=%d,%d client_date=%d: %v", ID(packet), len(packet), x, y, c.clientDate, err)
	}
	return err
}

func (c *Client) SendChangeDirection(headDir, dir uint8) error {
	packet := BuildChangeDirectionPacketForClientDate(headDir, dir, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CHANGE_DIRECTION opcode=0x%04X head_dir=%d dir=%d client_date=%d", ID(packet), headDir, dir&7, c.clientDate)
	} else {
		glog.Warnf("send CZ_CHANGE_DIRECTION failed opcode=0x%04X len=%d head_dir=%d dir=%d client_date=%d: %v", ID(packet), len(packet), headDir, dir&7, c.clientDate, err)
	}
	return err
}

func (c *Client) SendActionRequest(targetGID uint32, action uint8) error {
	packet := BuildActionRequestPacketForClientDate(targetGID, action, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_ACT opcode=0x%04X target=%d action=%d client_date=%d", ID(packet), targetGID, action, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_ACT failed opcode=0x%04X len=%d target=%d action=%d client_date=%d: %v", ID(packet), len(packet), targetGID, action, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCContact(npcID uint32) error {
	packet := BuildNPCContactPacket(npcID, 0)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CONTACTNPC opcode=0x%04X npc=%d client_date=%d", ID(packet), npcID, c.clientDate)
	} else {
		glog.Warnf("send CZ_CONTACTNPC failed opcode=0x%04X npc=%d client_date=%d: %v", ID(packet), npcID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCNext(npcID uint32) error {
	packet := BuildNPCNextPacket(npcID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_NEXT_SCRIPT opcode=0x%04X npc=%d client_date=%d", ID(packet), npcID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_NEXT_SCRIPT failed opcode=0x%04X npc=%d client_date=%d: %v", ID(packet), npcID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCClose(npcID uint32) error {
	packet := BuildNPCClosePacket(npcID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CLOSE_DIALOG opcode=0x%04X npc=%d client_date=%d", ID(packet), npcID, c.clientDate)
	} else {
		glog.Warnf("send CZ_CLOSE_DIALOG failed opcode=0x%04X npc=%d client_date=%d: %v", ID(packet), npcID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCMenuChoice(npcID uint32, choice uint8) error {
	packet := BuildNPCMenuChoicePacket(npcID, choice)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_CHOOSE_MENU opcode=0x%04X npc=%d choice=%d client_date=%d", ID(packet), npcID, choice, c.clientDate)
	} else {
		glog.Warnf("send CZ_CHOOSE_MENU failed opcode=0x%04X npc=%d choice=%d client_date=%d: %v", ID(packet), npcID, choice, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCNumberInput(npcID uint32, value int32) error {
	packet := BuildNPCNumberInputPacket(npcID, value)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_INPUT_EDITDLG opcode=0x%04X npc=%d value=%d client_date=%d", ID(packet), npcID, value, c.clientDate)
	} else {
		glog.Warnf("send CZ_INPUT_EDITDLG failed opcode=0x%04X npc=%d value=%d client_date=%d: %v", ID(packet), npcID, value, c.clientDate, err)
	}
	return err
}

func (c *Client) SendNPCStringInput(npcID uint32, value string) error {
	packet := BuildNPCStringInputPacket(npcID, value)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_INPUT_EDITDLGSTR opcode=0x%04X npc=%d len=%d client_date=%d", ID(packet), npcID, len(value), c.clientDate)
	} else {
		glog.Warnf("send CZ_INPUT_EDITDLGSTR failed opcode=0x%04X npc=%d len=%d client_date=%d: %v", ID(packet), npcID, len(value), c.clientDate, err)
	}
	return err
}

func (c *Client) SendStatusIncrease(statusID uint16) error {
	packet := BuildStatusIncreasePacket(statusID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_STATUS_CHANGE opcode=0x%04X status=%d amount=1 client_date=%d", ID(packet), statusID, c.clientDate)
	} else {
		glog.Warnf("send CZ_STATUS_CHANGE failed opcode=0x%04X len=%d status=%d client_date=%d: %v", ID(packet), len(packet), statusID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendSkillLevelUp(skillID uint16) error {
	packet := BuildSkillLevelUpPacket(skillID)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_UPGRADE_SKILLLEVEL opcode=0x%04X skill=%d client_date=%d", ID(packet), skillID, c.clientDate)
	} else {
		glog.Warnf("send CZ_UPGRADE_SKILLLEVEL failed opcode=0x%04X len=%d skill=%d client_date=%d: %v", ID(packet), len(packet), skillID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendRememberWarpPoint() error {
	packet := BuildRememberWarpPointPacket()
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REMEMBER_WARPPOINT opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_REMEMBER_WARPPOINT failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendUseSkillToID(skillID, level uint16, targetID uint32) error {
	packet := BuildUseSkillToIDPacketForClientDate(skillID, level, targetID, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_USE_SKILL opcode=0x%04X skill=%d level=%d target=%d client_date=%d", ID(packet), skillID, level, targetID, c.clientDate)
	} else {
		glog.Warnf("send CZ_USE_SKILL failed opcode=0x%04X len=%d skill=%d level=%d target=%d client_date=%d: %v", ID(packet), len(packet), skillID, level, targetID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendUseSkillToGround(skillID, level uint16, x, y int) error {
	packet := BuildUseSkillToGroundPacketForClientDate(skillID, level, x, y, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_USE_SKILL_TOGROUND opcode=0x%04X skill=%d level=%d dst=%d,%d client_date=%d", ID(packet), skillID, level, x, y, c.clientDate)
	} else {
		glog.Warnf("send CZ_USE_SKILL_TOGROUND failed opcode=0x%04X len=%d skill=%d level=%d dst=%d,%d client_date=%d: %v", ID(packet), len(packet), skillID, level, x, y, c.clientDate, err)
	}
	return err
}

func (c *Client) SendUseSkillToGroundWithText(skillID, level uint16, x, y int, text string) error {
	packet := BuildUseSkillToGroundWithTextPacketForClientDate(skillID, level, x, y, text, c.clientDate)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_USE_SKILL_TOGROUND_WITHTALKBOX opcode=0x%04X skill=%d level=%d dst=%d,%d text_len=%d client_date=%d", ID(packet), skillID, level, x, y, len([]byte(text)), c.clientDate)
	} else {
		glog.Warnf("send CZ_USE_SKILL_TOGROUND_WITHTALKBOX failed opcode=0x%04X len=%d skill=%d level=%d dst=%d,%d text_len=%d client_date=%d: %v", ID(packet), len(packet), skillID, level, x, y, len([]byte(text)), c.clientDate, err)
	}
	return err
}

func (c *Client) SendHomunculusCommand(command uint8) error {
	packet := BuildHomunculusCommandPacket(command)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_COMMAND_MER opcode=0x%04X command=%d client_date=%d", ID(packet), command, c.clientDate)
	} else {
		glog.Warnf("send CZ_COMMAND_MER failed opcode=0x%04X len=%d command=%d client_date=%d: %v", ID(packet), len(packet), command, c.clientDate, err)
	}
	return err
}

func (c *Client) SendHomunculusFeed() error {
	packet := BuildHomunculusCommandPacketWithType(PacketZCFeedMercenary, HomunculusCommandFeed)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_COMMAND_MER feed opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_COMMAND_MER feed failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendHomunculusDelete() error {
	packet := BuildHomunculusCommandPacketWithType(PacketZCFeedMercenary, HomunculusCommandDelete)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_COMMAND_MER delete opcode=0x%04X client_date=%d", ID(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_COMMAND_MER delete failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendHomunculusRename(name string) error {
	packet := BuildHomunculusRenamePacket(name)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_RENAME_MER opcode=0x%04X len=%d client_date=%d", ID(packet), len(packet), c.clientDate)
	} else {
		glog.Warnf("send CZ_RENAME_MER failed opcode=0x%04X len=%d client_date=%d: %v", ID(packet), len(packet), c.clientDate, err)
	}
	return err
}

func (c *Client) SendCompanionMove(gid uint32, x, y int) error {
	packet, ok := BuildCompanionMovePacket(gid, x, y)
	if !ok {
		return fmt.Errorf("invalid companion destination %d,%d", x, y)
	}
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_MOVENPC opcode=0x%04X gid=%d dst=%d,%d client_date=%d", ID(packet), gid, x, y, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_MOVENPC failed opcode=0x%04X len=%d gid=%d dst=%d,%d client_date=%d: %v", ID(packet), len(packet), gid, x, y, c.clientDate, err)
	}
	return err
}

func (c *Client) SendCompanionAttack(gid, targetGID uint32) error {
	packet := BuildCompanionAttackPacket(gid, targetGID, 0)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_ACTNPC opcode=0x%04X gid=%d target=%d client_date=%d", ID(packet), gid, targetGID, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_ACTNPC failed opcode=0x%04X len=%d gid=%d target=%d client_date=%d: %v", ID(packet), len(packet), gid, targetGID, c.clientDate, err)
	}
	return err
}

func (c *Client) SendCompanionMoveToOwner(gid uint32) error {
	packet := BuildCompanionMoveToOwnerPacket(gid)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQUEST_MOVETOOWNER opcode=0x%04X gid=%d client_date=%d", ID(packet), gid, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQUEST_MOVETOOWNER failed opcode=0x%04X len=%d gid=%d client_date=%d: %v", ID(packet), len(packet), gid, c.clientDate, err)
	}
	return err
}

func (c *Client) SendMercenaryCommand(command uint8) error {
	packet := BuildMercenaryCommandPacket(command)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_MER_COMMAND opcode=0x%04X command=%d client_date=%d", ID(packet), command, c.clientDate)
	} else {
		glog.Warnf("send CZ_MER_COMMAND failed opcode=0x%04X len=%d command=%d client_date=%d: %v", ID(packet), len(packet), command, c.clientDate, err)
	}
	return err
}

func (c *Client) SendChangeCart(cartNum uint16) error {
	packet := BuildChangeCartPacket(cartNum)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_REQ_CHANGECART opcode=0x%04X cart=%d client_date=%d", ID(packet), cartNum, c.clientDate)
	} else {
		glog.Warnf("send CZ_REQ_CHANGECART failed opcode=0x%04X len=%d cart=%d client_date=%d: %v", ID(packet), len(packet), cartNum, c.clientDate, err)
	}
	return err
}

func (c *Client) SendSelectWarpPoint(skillID uint16, mapName string) error {
	packet := BuildSelectWarpPointPacket(skillID, mapName)
	err := c.Send(packet)
	if err == nil {
		glog.Debugf("sent CZ_SELECT_WARPPOINT opcode=0x%04X skill=%d map=%q client_date=%d", ID(packet), skillID, mapName, c.clientDate)
	} else {
		glog.Warnf("send CZ_SELECT_WARPPOINT failed opcode=0x%04X len=%d skill=%d map=%q client_date=%d: %v", ID(packet), len(packet), skillID, mapName, c.clientDate, err)
	}
	return err
}

func (c *Client) Pump() {
}

func (c *Client) Status() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.status
}

func (c *Client) DrainPackets() []Packet {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]Packet, len(c.packets))
	copy(out, c.packets)
	c.packets = c.packets[:0]
	return out
}

func (c *Client) DrainErrors() []error {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]error, len(c.errs))
	copy(out, c.errs)
	c.errs = c.errs[:0]
	return out
}

func (c *Client) readLoop(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if n > 0 {
			if c.trace {
				headLen := min(n, 32)
				glog.Debugf("network read n=%d head=%s", n, hex.EncodeToString(buf[:headLen]))
			}
			packets, frameErr := c.framer.Push(buf[:n])
			c.mu.Lock()
			c.packets = append(c.packets, packets...)
			if frameErr != nil {
				c.errs = append(c.errs, FrameError{Err: frameErr})
			}
			c.mu.Unlock()
		}
		if err != nil {
			if c.isCurrentConn(conn) {
				if err == io.EOF {
					c.addError(ErrDisconnected)
				} else {
					c.addError(err)
				}
			}
			c.clearConn(conn)
			return
		}
	}
}

func (c *Client) writeLoop(conn net.Conn, sendCh <-chan outboundPacket) {
	for packet := range sendCh {
		queued := time.Since(packet.enqueued)
		start := time.Now()
		if _, err := conn.Write(packet.data); err != nil {
			if c.isCurrentConn(conn) {
				c.addError(err)
			}
			c.clearConn(conn)
			return
		}
		elapsed := time.Since(start)
		if c.trace || queued > 2*time.Millisecond || elapsed > 2*time.Millisecond {
			headLen := min(len(packet.data), 32)
			glog.Debugf("network write opcode=0x%04X n=%d queued=%s elapsed=%s head=%s", ID(packet.data), len(packet.data), queued, elapsed, hex.EncodeToString(packet.data[:headLen]))
		}
	}
}

func (c *Client) isCurrentConn(conn net.Conn) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn == conn
}

func (c *Client) clearConn(conn net.Conn) {
	c.mu.Lock()
	if c.conn == conn {
		sendCh := c.sendCh
		c.conn = nil
		c.sendCh = nil
		c.status = "offline"
		if sendCh != nil {
			close(sendCh)
		}
	}
	c.mu.Unlock()
	_ = conn.Close()
}

func (c *Client) setStatus(status string) {
	c.mu.Lock()
	c.status = status
	c.mu.Unlock()
}

func (c *Client) addError(err error) {
	c.mu.Lock()
	c.errs = append(c.errs, err)
	c.mu.Unlock()
}
