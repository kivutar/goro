package game

import (
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
)

func sendLessEffectPreference(ctx client.Context) {
	if ctx.Session == nil || ctx.Network == nil || !ctx.Session.LessEffects {
		return
	}
	if err := ctx.Network.SendLessEffect(true); err != nil {
		glog.Warnf("send less effect preference failed: %v", err)
	}
}
