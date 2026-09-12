package ui

import "strings"

// WhisperWindows keeps a conversation and draft per recipient for the session.
// Closing a window hides the conversation; opening it again restores it.
type WhisperWindows struct {
	conversations []*WhisperWindow
}

func (w *WhisperWindows) Find(target string) *WhisperWindow {
	for _, conversation := range w.conversations {
		if strings.EqualFold(conversation.target, strings.TrimSpace(target)) {
			return conversation
		}
	}
	return nil
}

func (w *WhisperWindows) open(ctx Context, target string) *WhisperWindow {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	conversation := w.Find(target)
	if conversation == nil {
		conversation = &WhisperWindow{}
		w.conversations = append(w.conversations, conversation)
	}
	if !conversation.IsOpen() {
		conversation.open(ctx, target)
	}
	return conversation
}

func (w *WhisperWindows) Open(ctx Context, target string) *WhisperWindow {
	conversation := w.open(ctx, target)
	if conversation != nil {
		conversation.Raise(ctx)
		conversation.focusInput()
	}
	return conversation
}

func (w *WhisperWindows) AddIncoming(ctx Context, sender, message string) {
	if conversation := w.open(ctx, sender); conversation != nil {
		conversation.AddIncoming(ctx, sender, message)
	}
}

func (w *WhisperWindows) IsOpen() bool {
	for _, conversation := range w.conversations {
		if conversation.IsOpen() {
			return true
		}
	}
	return false
}

func (w *WhisperWindows) Update(ctx Context) (bool, WhisperWindowAction) {
	// Widget callbacks run before world input. Drain sends independently of
	// which conversation happens to be under the pointer this frame.
	for _, conversation := range w.conversations {
		if conversation.IsOpen() {
			conversation.submitFromFocusedEnter(ctx)
		}
		if action := conversation.PopAction(); action.Target != "" {
			return true, action
		}
	}
	for _, conversation := range w.conversations {
		consumed := conversation.Update(ctx)
		if consumed {
			return true, WhisperWindowAction{}
		}
	}
	return false, WhisperWindowAction{}
}

func (w *WhisperWindows) Rebind(ctx Context) {
	for _, conversation := range w.conversations {
		conversation.Rebind(ctx)
	}
}
