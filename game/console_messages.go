package game

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

var (
	expNotifyColor      = color.RGBA{R: 255, G: 255, B: 99, A: 255}
	questExpNotifyColor = color.RGBA{R: 164, G: 66, B: 220, A: 255}
)

func formatConsoleMessage(manager *res.Manager, chat network.ChatMessage) string {
	if chat.Text != "" {
		return chat.Text
	}
	if chat.MessageID < 0 {
		return ""
	}
	text := ""
	if manager != nil {
		text, _ = manager.MsgString(chat.MessageID)
	}
	if text == "" {
		text = fmt.Sprintf("message #%d", chat.MessageID)
	}
	if chat.Value != 0 {
		if strings.Contains(text, "%") {
			text = fmt.Sprintf(text, chat.Value)
		} else {
			text = fmt.Sprintf("%s %d", text, chat.Value)
		}
	}
	if chat.SkillID != 0 {
		text = fmt.Sprintf("skill %d: %s", chat.SkillID, text)
	}
	return text
}

func addConsoleMessage(console *gameui.ChatConsole, manager *res.Manager, chat network.ChatMessage) {
	if console == nil {
		return
	}
	text := formatConsoleMessage(manager, chat)
	if text == "" {
		return
	}
	if chat.HasColor {
		console.AddColoredMessage(roClientColor(chat.Color), "%s", text)
		return
	}
	if chat.Text == "" || !strings.Contains(text, " : ") {
		console.AddSystemMessage("%s", text)
		return
	}
	console.AddMessage("%s", text)
}

func roClientColor(value uint32) color.RGBA {
	return color.RGBA{
		R: uint8(value),
		G: uint8(value >> 8),
		B: uint8(value >> 16),
		A: 255,
	}
}

func addExpNotifyMessage(console *gameui.ChatConsole, manager *res.Manager, notify network.ExpNotify) {
	if console == nil {
		return
	}
	text, messageColor := formatExpNotifyMessage(manager, notify)
	if text == "" {
		return
	}
	console.AddColoredMessage(messageColor, "%s", text)
}

func formatExpNotifyMessage(manager *res.Manager, notify network.ExpNotify) (string, color.RGBA) {
	switch notify.ExpType {
	case 0:
		switch notify.VarID {
		case network.StatusBaseExp:
			return formatExpNotifyTemplate(manager, 1613, "Gained %d base experience", notify.Amount), expNotifyColor
		case network.StatusJobExp:
			return formatExpNotifyTemplate(manager, 1614, "Gained %d job experience", notify.Amount), expNotifyColor
		default:
			return "", color.RGBA{}
		}
	case 1:
		switch notify.VarID {
		case network.StatusBaseExp:
			return fmt.Sprintf("Experience gained from Quest, Base:%d", notify.Amount), questExpNotifyColor
		case network.StatusJobExp:
			return fmt.Sprintf("Experience gained from Quest, Job:%d", notify.Amount), questExpNotifyColor
		default:
			return "", color.RGBA{}
		}
	default:
		return "", color.RGBA{}
	}
}

func formatExpNotifyTemplate(manager *res.Manager, messageID int, fallback string, amount int32) string {
	text := ""
	if manager != nil {
		text, _ = manager.MsgString(messageID)
	}
	if text == "" {
		return fmt.Sprintf(fallback, amount)
	}
	return strings.Replace(text, "%d", strconv.FormatInt(int64(amount), 10), 1)
}

func addWhisperMessage(console *gameui.ChatConsole, whisper network.WhisperMessage) {
	if console == nil {
		return
	}
	if whisper.Sender == "" || whisper.Message == "" {
		return
	}
	console.AddBlueMessage("[ From %s ] : %s", whisper.Sender, whisper.Message)
}

func addWhisperAck(console *gameui.ChatConsole, manager *res.Manager, ack network.WhisperAck) {
	if console == nil || ack.Result == 0 {
		return
	}
	console.AddErrorMessage("%s", whisperAckMessage(manager, ack))
}

func addWhisperIgnoreAck(console *gameui.ChatConsole, ack network.WhisperIgnoreAck) {
	if console == nil {
		return
	}
	if ack.Result != 0 {
		console.AddErrorMessage("%s", whisperIgnoreAckFailure(ack))
		return
	}
	console.AddSystemMessage("%s", whisperIgnoreAckSuccess(ack))
}

func whisperIgnoreAckSuccess(ack network.WhisperIgnoreAck) string {
	if ack.TargetAll {
		if ack.Allow {
			return "Whispers from everyone are allowed."
		}
		return "Whispers from everyone are blocked."
	}
	if ack.Allow {
		return "Whispers from that player are allowed."
	}
	return "Whispers from that player are blocked."
}

func whisperIgnoreAckFailure(ack network.WhisperIgnoreAck) string {
	if !ack.TargetAll && ack.Result == 2 {
		return "Whisper block list is full."
	}
	if ack.TargetAll {
		if ack.Allow {
			return "Allow all whispers failed."
		}
		return "Block all whispers failed."
	}
	if ack.Allow {
		return "Allow whisper failed."
	}
	return "Block whisper failed."
}

func whisperAckMessage(manager *res.Manager, ack network.WhisperAck) string {
	message := ""
	if manager != nil {
		message, _ = manager.MsgString(int(147 + ack.Result))
	}
	if message == "" {
		message = "Whisper failed."
	}
	return message
}

func pickupFailureMessage(manager *res.Manager, result uint8) string {
	// Match the original client's Zc_Item_Pickup_Ack message IDs. Unknown
	// results remain log-only in applyItemPickupAck.
	var messageID int
	var fallback string
	switch result {
	case 1, 6:
		messageID, fallback = 53, "You can't get this item."
	case 2:
		messageID, fallback = 52, "You can't carry any more weight."
	case 4:
		messageID, fallback = 220, "You can't carry any more items."
	case 5:
		messageID, fallback = 279, "You can't carry any more of this item."
	default:
		return ""
	}
	if manager != nil {
		if message, ok := manager.MsgString(messageID); ok {
			return message
		}
	}
	return fallback
}

func formatPickupConsoleMessage(manager *res.Manager, pickup network.ItemPickupAck) string {
	itemName := fmt.Sprintf("item %d", pickup.ItemID)
	if manager != nil {
		if name, ok := manager.ItemDisplayName(int(pickup.ItemID), pickup.Identified); ok && name != "" {
			itemName = name
		}
	}
	amount := int(pickup.Amount)
	if amount <= 0 {
		amount = 1
	}
	template := ""
	if manager != nil {
		template, _ = manager.MsgString(153)
	}
	if template == "" {
		template = "You got %s %d."
	}
	if strings.Contains(template, "%s") {
		template = strings.Replace(template, "%s", itemName, 1)
	} else {
		template = strings.TrimSpace(template + " " + itemName)
	}
	if strings.Contains(template, "%d") {
		template = strings.Replace(template, "%d", fmt.Sprintf("%d", amount), 1)
	} else if amount != 1 {
		template = strings.TrimSpace(fmt.Sprintf("%s %d", template, amount))
	}
	return template
}
