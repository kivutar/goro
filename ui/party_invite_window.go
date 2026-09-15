package ui

const partyInviteW = 286

type PartyInviteWindow struct {
	TextPromptWindow
}

func (w *PartyInviteWindow) Open(ctx Context) {
	w.EnsureWindow(partyInviteW, ROWindowTitleHeight+textPromptContentH+ROWindowFooterHeight)
	w.TextPromptWindow.Open(ctx, "Party Invitation", "Player Name", "Player Name", 23)
}

func (w *PartyInviteWindow) PopAction() string {
	return w.TextPromptWindow.PopAction().Text
}
