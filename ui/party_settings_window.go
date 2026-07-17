package ui

import (
	"strconv"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/radio"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	partySettingsW       = 286
	partySettingsContent = 132
	partySettingsFooterH = 42
)

type PartySettingsWindow struct {
	Window
	expShare      uint32
	refuseInvites bool
}

func (w *PartySettingsWindow) Open(ctx Context) {
	w.EnsureWindow(partySettingsW, ROWindowTitleHeight+partySettingsContent+partySettingsFooterH)
	w.ctx = ctx
	party := sessionParty(ctx.Session)
	w.expShare = party.ExpShare
	w.refuseInvites = party.RefuseInvites
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *PartySettingsWindow) Update(ctx Context) bool {
	w.EnsureWindow(partySettingsW, ROWindowTitleHeight+partySettingsContent+partySettingsFooterH)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *PartySettingsWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *PartySettingsWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title("Party Settings"),
		CloseButton(true),
		OnClose(w.Close),
		Size(partySettingsW, ROWindowTitleHeight+partySettingsContent+partySettingsFooterH),
		Content(
			primitives.Box(
				rotheme.SectionLabel("EXP"),
				rotheme.Radio(
					radio.Items(
						radio.ItemDef{Value: "0", Label: "Each Take"},
						radio.ItemDef{Value: "1", Label: "Even Share"},
					),
					radio.Selected(strconv.Itoa(int(w.expShare))),
					radio.OnChange(func(value string) {
						w.expShare = parsePartySettingUint32(value)
					}),
				),
				rotheme.Checkbox(
					checkbox.Checked(w.refuseInvites),
					checkbox.LabelOpt("Refuse party invites"),
					checkbox.OnToggle(func(enabled bool) {
						w.refuseInvites = enabled
					}),
				),
			).
				Padding(14).
				Gap(8),
		),
		FooterHeight(partySettingsFooterH),
		Footer(
			primitives.HBox(
				primitives.Expanded(primitives.Box()),
				rotheme.Button("OK", func() {
					w.apply(ctx)
				}).Width(float32(ButtonLabelWidth("OK"))),
				rotheme.Button("Cancel", w.Close).Width(float32(ButtonLabelWidth("Cancel"))),
			).
				Gap(8),
		),
	)
}

func (w *PartySettingsWindow) apply(ctx Context) {
	if ctx.Session != nil {
		ctx.Session.Party.ExpShare = w.expShare
		ctx.Session.Party.RefuseInvites = w.refuseInvites
	}
	if ctx.Network != nil {
		if err := ctx.Network.SendPartyOption(w.expShare); err != nil {
			glog.Warnf("party settings failed: %v", err)
		}
		if err := ctx.Network.SendPartyInviteConfig(w.refuseInvites); err != nil {
			glog.Warnf("party invite settings failed: %v", err)
		}
	}
	w.Window.Close()
}

func parsePartySettingUint32(value string) uint32 {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0
	}
	return uint32(n)
}

func parsePartySettingUint8(value string) uint8 {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0
	}
	return 1
}
