// Package actors manages actor appearance, animation, movement and rendering.
package actors

import (
	"time"

	"github.com/kivutar/goro/game/effects"
	maps "github.com/kivutar/goro/game/map"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/game/skills"
	"github.com/kivutar/goro/game/social"
	"github.com/kivutar/goro/game/sprites"
	"github.com/kivutar/goro/render"
)

type Bindings struct {
	Map                  *maps.System
	Effects              *effects.System
	Skills               *skills.System
	Social               *social.System
	SpriteTexture        func(*render.Image) *render.Image
	ScheduleSoundAtActor func(time.Time, uint32, ...string)
}

type System struct {
	PlayerView        *sprites.Humanoid
	ShadowView        *scene.SpriteView
	ShadowViewMiss    bool
	CartViews         map[int]*scene.SpriteView
	CartViewMiss      map[int]struct{}
	FalconViews       map[int]*scene.SpriteView
	FalconViewMiss    map[int]struct{}
	Falcons           map[uint32]*FalconRenderState
	ActorViews        map[sprites.Appearance]*sprites.Humanoid
	ActorViewMiss     map[sprites.Appearance]struct{}
	MercenaryViews    map[sprites.Appearance]*sprites.Humanoid
	MercenaryViewMiss map[sprites.Appearance]struct{}
	NonPCViews        map[int]*scene.SpriteView
	NonPCViewMiss     map[int]struct{}
	GR2Models         map[int]*GR2ModelView
	GR2ModelMiss      map[int]struct{}
	PetAccessoryIDs   map[uint32]uint32
	PetAccessoryViews map[PetAccessorySpriteKey]*scene.SpriteView
	PetAccessoryMiss  map[PetAccessorySpriteKey]struct{}
	ActorAnims        map[uint32]scene.ActorAnimation
	ComboDelays       map[uint32]time.Duration
	ActorDeaths       map[uint32]time.Time
	ActorVanishes     map[uint32]VanishFade
	ActorNameReqAt    map[uint32]time.Time
	SpeechBubbles     map[uint32]SpeechBubble
	Ailments          map[uint32]AilmentVisual
	bindings          Bindings
}

func (s *System) Bind(bindings Bindings) { s.bindings = bindings }
