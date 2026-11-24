// Package battle manages combat status, damage numbers and combat overlays.
package battle

import (
	"image"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/npc"
	"github.com/kivutar/goro/game/scene"
	gameui "github.com/kivutar/goro/ui"
)

type Windows struct {
	AutoSpell   gameui.AutoSpellWindow
	MonsterInfo gameui.MonsterInfoWindow
	StatusIcons gameui.StatusIcons
}

type Bindings struct {
	Windows              *Windows
	Console              *gameui.ChatConsole
	Minimap              *gameui.Minimap
	Map                  *maps.System
	Actors               *actors.System
	Effects              *effects.System
	Progress             *npc.ProgressState
	SetActorAction       func(client.Context, uint32, scene.ActorAnimation)
	ScheduleSoundAtActor func(time.Time, uint32, ...string)
	MonsterPreview       func(client.Context, uint16, int, int) image.Image
}

type System struct {
	DamageNumberView *scene.SpriteView
	DamageNumberMiss bool
	DamageNumbers    map[string]*scene.Billboard
	DamageMsgView    *scene.SpriteView
	DamageMsgMiss    bool
	BlueMsgView      *scene.SpriteView
	BlueMsgMiss      bool
	DamageFloaters   []Floater
	ActorLife        map[uint32]Life
	ActorCastBars    map[uint32]CastBar
	senseRequest     senseRequest
	bindings         Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
