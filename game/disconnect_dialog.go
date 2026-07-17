package game

import (
	"errors"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	gameui "github.com/kivutar/goro/ui"
)

type disconnectMessage struct {
	msgID    int
	fallback string
}

var disconnectBanMessages = map[uint8]disconnectMessage{
	0:   {3, "Disconnected from Server!"},
	1:   {4, "Server Closed."},
	2:   {5, "Someone has Logged in with this ID."},
	3:   {241, "You've been disconnected due to a time gap between you and the server."},
	4:   {264, "Server is jammed due to over population. Please try again shortly."},
	5:   {305, "You are underaged and cannot join this server."},
	6:   {764, "Trial players can't connect Pay to Play Server."},
	8:   {440, "Server still recognizes your last log-in. Please try again after about 30 seconds."},
	9:   {529, "IP capacity of this Internet Cafe is full. Would you like to pay the personal base?"},
	10:  {530, "You are out of available paid playing time. Game will be shut down automatically."},
	11:  {575, "Your account is suspended."},
	12:  {576, "Your connection is terminated due to change in the billing policy. Please connect again."},
	13:  {577, "Your connection is terminated because your IP doesn't match the authorized IP from the account server."},
	14:  {578, "Your connection is terminated to prevent charging from your account's play time."},
	15:  {3, "Disconnected from Server!"}, // rAthena also uses code 15 for non-GM forced disconnects, including some shutdown/session-change paths.
	16:  {606, "Disconnected from Server!"},
	17:  {607, "Disconnected from Server!"},
	18:  {678, "Your account is already connected to account server."},
	100: {1123, "Please check the connection, more than 2 accounts are connected with Internet Cafe Time Plan."},
	101: {1178, "More than 30 players sharing the same IP have logged into the game for an hour. Please check this matter."},
	102: {1179, "More than 10 connections sharing the same IP have logged into the game for an hour. Please check this matter."},
	103: {1309, "You need to accept the Privacy Policy from Gravity in order to use the service."},
	104: {1310, "You need to accept the User Agreement in order to use the service."},
	105: {1311, "Incorrect or nonexistent ID."},
	106: {1373, "The number of accounts connected to this IP has exceeded the limit."},
	107: {1429, "You have too many characters in your account. Server does not allow you to make more."},
	108: {1582, "It is impossible to connect using this IP in Ragnarok Online. Please contact the customer support center or home."},
	109: {1583, "You have entered a wrong password for more than six times, please check your personal information again."},
	110: {1589, "Sorry the character you are trying to use is banned for testing connection."},
}

func disconnectMessageForBanCode(resources *res.Manager, code uint8) string {
	message, ok := disconnectBanMessages[code]
	if !ok {
		message = disconnectBanMessages[0]
	}
	return disconnectMessageText(resources, message)
}

func disconnectMessageText(resources *res.Manager, message disconnectMessage) string {
	if resources != nil {
		if text, ok := resources.MsgString(message.msgID); ok {
			return text
		}
	}
	return message.fallback
}

func openConnectionFailedDialog(ctx client.Context, modal *gameui.ConfirmModal) {
	message := disconnectMessageText(ctx.Resources, disconnectMessage{1, "Failed to Connect to Server."})
	openDisconnectDialog(ctx, modal, message)
}

func openDisconnectDialog(ctx client.Context, modal *gameui.ConfirmModal, message string) {
	if modal == nil || modal.IsOpen() {
		return
	}
	if ctx.Network != nil {
		ctx.Network.Close()
	}
	modal.OpenAlert(ctx, "Disconnected", message, func() {
		if ctx.RequestQuit != nil {
			ctx.RequestQuit()
		}
	})
}

func handleDisconnectPacket(ctx client.Context, modal *gameui.ConfirmModal, pkt network.Packet) bool {
	if ban, ok, err := network.ParseNotifyBan(pkt); err != nil {
		glog.Errorf("parse SC_NOTIFY_BAN 0x%04X: %v", pkt.ID, err)
		message := disconnectMessageText(ctx.Resources, disconnectMessage{3, "Disconnected from Server!"})
		openDisconnectDialog(ctx, modal, message)
		return true
	} else if ok {
		openDisconnectDialog(ctx, modal, disconnectMessageForBanCode(ctx.Resources, ban.ErrorCode))
		return true
	}
	if network.IsInterServerDisconnect(pkt) {
		message := disconnectMessageText(ctx.Resources, disconnectMessage{3, "Disconnected from Server!"})
		openDisconnectDialog(ctx, modal, message)
		return true
	}
	return false
}

func handleNetworkDisconnectErrors(ctx client.Context, modal *gameui.ConfirmModal, errs []error) bool {
	if len(errs) == 0 {
		return false
	}
	var disconnectErr error
	for _, err := range errs {
		var frameErr network.FrameError
		if errors.As(err, &frameErr) {
			continue
		}
		glog.Errorf("network error: %v", err)
		if disconnectErr == nil {
			disconnectErr = err
		}
	}
	if disconnectErr == nil {
		return false
	}
	message := disconnectMessageText(ctx.Resources, disconnectMessage{2, "Disconnected from Server."})
	openDisconnectDialog(ctx, modal, message)
	return true
}
