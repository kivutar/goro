package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
)

type Manager struct {
	app  client.UIApp
	root widget.Widget
}

var emptyRoot = primitives.Box()

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) SetUIApp(app client.UIApp) {
	if m == nil || m.app == app {
		return
	}
	m.app = app
	m.apply()
}

func (m *Manager) SetRoot(root widget.Widget) {
	if m == nil {
		return
	}
	if root == nil {
		root = emptyUIRoot()
	}
	if m.root == root {
		return
	}
	m.root = root
	m.apply()
}

func (m *Manager) Clear() {
	m.SetRoot(emptyUIRoot())
}

func (m *Manager) apply() {
	if m.app != nil && m.root != nil {
		m.app.SetRoot(m.root)
	}
}

func emptyUIRoot() widget.Widget {
	return emptyRoot
}
