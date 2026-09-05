package game

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

var taekwonAnnouncementColor = color.RGBA{R: 248, G: 248, B: 255, A: 255}

const (
	starPlaceConfirmMessageID = 1028
	starPlaceConfirmFallback  = "You cannot change a map's designation once it is designated. Are you sure that you want to designate this map?"
)

func (m *WorldMode) applyStarPlaceRequest(ctx client.Context, place network.StarPlace) {
	message := starPlaceConfirmFallback
	if ctx.Resources != nil {
		message = taekwonMessageString(ctx, starPlaceConfirmMessageID, message)
	}
	m.ui.starPlaceConfirm.Open(ctx, "Feeling the Sun, Moon and Stars", message, func() {
		if ctx.Network == nil {
			m.ui.console.AddErrorMessage("Place registration failed: not connected.")
			return
		}
		if err := ctx.Network.SendAgreeStarPlace(place.Place); err != nil {
			m.ui.console.AddErrorMessage("Place registration failed.")
			glog.Warnf("star gladiator place confirmation failed place=%d: %v", place.Place, err)
			return
		}
		glog.Debugf("star gladiator place confirmed place=%d", place.Place)
	}, func() {
		glog.Debugf("star gladiator place canceled place=%d", place.Place)
	})
	glog.Debugf("star gladiator place confirmation opened place=%d", place.Place)
}

func (m *WorldMode) applyTaekwonMission(ctx client.Context, mission network.TaekwonMission) {
	progress := int(mission.Progress)
	template := "Taekwon mission: %s (%d%)"
	if ctx.Resources != nil {
		if message, ok := ctx.Resources.MsgString(927); ok && strings.TrimSpace(message) != "" {
			template = message
		}
	}
	message := strings.Replace(template, "%s", mission.MonsterName, 1)
	message = strings.Replace(message, "%d%", strconv.Itoa(progress)+"%", 1)
	m.ui.console.AddColoredMessage(taekwonAnnouncementColor, "%s", message)
	glog.Debugf("taekwon mission monster=%q id=%d progress=%d result=%d", mission.MonsterName, mission.MonsterID, mission.Progress, mission.Result)
}

func taekwonMessageString(ctx client.Context, id int, fallback string) string {
	message, ok := ctx.Resources.MsgString(id)
	if !ok || strings.TrimSpace(message) == "" {
		return fallback
	}
	return message
}
