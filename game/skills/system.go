package skills

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

type PendingTarget struct {
	Skill       session.Skill
	MaxLevel    int
	TargetID    uint32
	Ground      bool
	X, Y        int
	Text        string
	Expires     time.Time
	ReadyAt     time.Time
	Source      string
	Started     time.Time
	LastChaseAt time.Time
}

type PendingTextTarget struct {
	Skill  session.Skill
	X      int
	Y      int
	Source string
}

// Bindings connect skill control to movement, UI and scene rendering.
type Bindings struct {
	ChangeCart          *gameui.ChangeCartWindow
	TextPrompt          *gameui.TextPromptWindow
	Console             *gameui.ChatConsole
	TeleportModal       *gameui.TeleportModal
	StarPlaceConfirm    *gameui.ConfirmModal
	Error               func(string, ...any)
	RequestWalk         func(client.Context, int, int, string) bool
	WalkBlocked         func()
	BeginPlayerAction   func(client.Context)
	PlayerCanUseSkill   func(client.Context, time.Time) bool
	RememberSenseTarget func(uint16, uint32, time.Time)
	ClickActor          func(client.Context, scene.Projection, session.Skill, int, int, time.Time) (worldstate.Actor, bool)
	NumberBillboard     func(client.Context, string) (*scene.Billboard, bool)
	Effects             *effects.System
	Map                 *maps.System
}

type System struct {
	Graffiti        map[uint32]*groundGraffiti
	Pending         PendingTarget
	PendingText     PendingTextTarget
	Units           map[uint32]skillUnitModel
	HiddenUnits     map[uint32]skillUnitModel
	CursorLevelNums map[string]*scene.Billboard
	bindings        Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
