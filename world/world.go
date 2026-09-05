package world

import (
	"time"

	"github.com/kivutar/goro/res"
)

type World struct {
	MapName     string
	MapProperty MapProperty
	Player      Actor
	Actors      map[uint32]Actor
	Items       map[uint32]FloorItem
	Camera      Camera
	Dir         int
	GAT         *res.GAT
	GND         *res.GND
	RSW         *res.RSW
	RSM         map[string]*res.RSM
	RSMFail     int
}

type MapProperty uint16

const (
	MapPropertyNothing MapProperty = iota
	MapPropertyFreePvPZone
	MapPropertyEventPvPZone
	MapPropertyAgitZone
	MapPropertyPKServerZone
	MapPropertyPvPServerZone
	MapPropertyDenySkillZone
)

func (p MapProperty) PlayerCombatEnabled() bool {
	return p.IsPvP() || p.IsGvG()
}

func (p MapProperty) IsPvP() bool {
	switch p {
	case MapPropertyFreePvPZone, MapPropertyEventPvPZone, MapPropertyPvPServerZone:
		return true
	default:
		return false
	}
}

// IsGvG reports the legacy 2008 WoE map property. PvP and GvG deliberately
// remain separate because only PvP maps have the client-side ranking display.
func (p MapProperty) IsGvG() bool {
	return p == MapPropertyAgitZone
}

func (p MapProperty) IsSiege() bool {
	return p.IsGvG()
}

// IsAutoReviveRestricted matches the classic client restrictions for using a
// Token of Siegfried.
func (p MapProperty) IsAutoReviveRestricted() bool {
	switch p {
	case MapPropertyEventPvPZone, MapPropertyAgitZone, MapPropertyPKServerZone:
		return true
	default:
		return false
	}
}

func (p MapProperty) PvPRankingEnabled() bool {
	return p == MapPropertyFreePvPZone || p == MapPropertyPvPServerZone
}

type FloorItem struct {
	ID         uint32
	ItemID     uint16
	Identified bool
	X          int
	Y          int
	SubX       uint8
	SubY       uint8
	Amount     uint16
	Falling    bool
	DroppedAt  time.Time
}

type Actor struct {
	ID               uint32
	Name             string
	PartyName        string
	GuildID          uint32
	EmblemVersion    uint32
	GuildName        string
	X                int
	Y                int
	Dir              int
	Job              int16
	Head             int16
	Weapon           int16
	Shield           int16
	HeadTop          int16
	HeadMid          int16
	HeadLow          int16
	HeadPal          int16
	BodyPal          int16
	Sex              byte
	HeadDir          uint8
	IsAdmin          bool
	Appearance       bool
	Moving           bool
	FromX            int
	FromY            int
	ToX              int
	ToY              int
	MoveStartTick    uint32
	HasMoveStartTick bool
	MoveStarted      time.Time
	MoveDuration     time.Duration
	MovePath         []WalkStep
	MoveStartX       float64
	MoveStartY       float64
	HasMoveStart     bool
	WalkDistance     float64
	ObjectType       uint8
	HasObjectType    bool
	Speed            int
	AIMotion         int
	HasAIMotion      bool
	AIMotionExpires  time.Time
	AITargetID       uint32
	AttackRange      int
	Sitting          bool
	BodyState        uint16
	HealthState      uint16
	EffectState      uint32
	Opt3State        uint32
	HasState         bool
	Level            int
	HasLevel         bool
	HasCart          bool
	CartNum          int
	HasCartState     bool
	Vending          bool
	VendingName      string
	ChatRoom         bool
	ChatRoomID       uint32
	ChatRoomTitle    string
	ChatRoomCount    uint16
	ChatRoomLimit    uint16
	ChatRoomPublic   bool
	PvPRank          int
	PvPTotal         int
	HasPvPRanking    bool
}

type WalkStep struct {
	X int
	Y int
}

type Camera struct {
	X float64
	Y float64
}

func New() *World {
	return &World{
		MapName: "prontera",
		Player:  Actor{Name: "Player", X: 50, Y: 50},
		Actors:  make(map[uint32]Actor),
		Items:   make(map[uint32]FloorItem),
	}
}

func (w *World) SetPlayerPosition(x, y, dir int) {
	w.Player.X = x
	w.Player.Y = y
	w.Player.Dir = dir
	w.Player.HeadDir = 0
	w.Dir = dir
	w.Player.Moving = false
	w.Player.FromX = x
	w.Player.FromY = y
	w.Player.ToX = x
	w.Player.ToY = y
	w.Player.MovePath = nil
	w.Player.HasMoveStart = false
}

func (w *World) UpsertActor(actor Actor) {
	if actor.ID == 0 {
		return
	}
	if existing, ok := w.Actors[actor.ID]; ok {
		if actor.Name == "" {
			actor.Name = existing.Name
		}
		if actor.PartyName == "" {
			actor.PartyName = existing.PartyName
		}
		if actor.GuildID == 0 {
			actor.GuildID = existing.GuildID
		}
		if actor.EmblemVersion == 0 {
			actor.EmblemVersion = existing.EmblemVersion
		}
		if actor.GuildName == "" {
			actor.GuildName = existing.GuildName
		}
		if !actor.Appearance {
			actor.Job = existing.Job
			actor.Head = existing.Head
			actor.Weapon = existing.Weapon
			actor.Shield = existing.Shield
			actor.HeadTop = existing.HeadTop
			actor.HeadMid = existing.HeadMid
			actor.HeadLow = existing.HeadLow
			actor.HeadPal = existing.HeadPal
			actor.BodyPal = existing.BodyPal
			actor.Sex = existing.Sex
			actor.HeadDir = existing.HeadDir
			actor.Appearance = existing.Appearance
		}
		if !actor.IsAdmin && existing.IsAdmin {
			actor.IsAdmin = true
		}
		if !actor.HasObjectType {
			actor.ObjectType = existing.ObjectType
			actor.HasObjectType = existing.HasObjectType
		}
		if actor.Speed <= 0 {
			actor.Speed = existing.Speed
		}
		if !actor.HasAIMotion {
			actor.AIMotion = existing.AIMotion
			actor.HasAIMotion = existing.HasAIMotion
			actor.AIMotionExpires = existing.AIMotionExpires
		}
		if actor.AITargetID == 0 {
			actor.AITargetID = existing.AITargetID
		}
		if actor.AttackRange <= 0 {
			actor.AttackRange = existing.AttackRange
		}
		if !actor.HasState {
			actor.BodyState = existing.BodyState
			actor.HealthState = existing.HealthState
			actor.EffectState = existing.EffectState
			actor.Opt3State = existing.Opt3State
			actor.HasState = existing.HasState
		}
		if !actor.HasLevel {
			actor.Level = existing.Level
			actor.HasLevel = existing.HasLevel
		}
		if !actor.HasCartState {
			actor.HasCart = existing.HasCart
			actor.CartNum = existing.CartNum
			actor.HasCartState = existing.HasCartState
		}
		if !actor.Vending && existing.Vending {
			actor.Vending = existing.Vending
			actor.VendingName = existing.VendingName
		}
		if !actor.ChatRoom && existing.ChatRoom {
			actor.ChatRoom = existing.ChatRoom
			actor.ChatRoomID = existing.ChatRoomID
			actor.ChatRoomTitle = existing.ChatRoomTitle
			actor.ChatRoomCount = existing.ChatRoomCount
			actor.ChatRoomLimit = existing.ChatRoomLimit
			actor.ChatRoomPublic = existing.ChatRoomPublic
		}
		if !actor.HasPvPRanking && existing.HasPvPRanking {
			actor.PvPRank = existing.PvPRank
			actor.PvPTotal = existing.PvPTotal
			actor.HasPvPRanking = true
		}
		if existing.Sitting && !actor.Moving {
			actor.Sitting = true
		}
		if existing.HeadDir != 0 && actor.HeadDir == 0 && !actor.Moving {
			actor.HeadDir = existing.HeadDir
		}
	}
	if actor.Moving {
		actor.Sitting = false
		actor.HeadDir = 0
		if !actor.HasAIMotion {
			actor.AIMotion = 1
			actor.HasAIMotion = true
		}
	} else {
		actor.FromX = actor.X
		actor.FromY = actor.Y
		actor.ToX = actor.X
		actor.ToY = actor.Y
		actor.MovePath = nil
		actor.HasMoveStart = false
		actor.WalkDistance = 0
	}
	w.Actors[actor.ID] = actor
}

func (w *World) ResetMapProperty() {
	if w == nil {
		return
	}
	w.MapProperty = MapPropertyNothing
	w.ClearPvPRankings()
}

func (w *World) ClearPvPRankings() {
	if w == nil {
		return
	}
	w.Player.PvPRank = 0
	w.Player.PvPTotal = 0
	w.Player.HasPvPRanking = false
	for id, actor := range w.Actors {
		actor.PvPRank = 0
		actor.PvPTotal = 0
		actor.HasPvPRanking = false
		w.Actors[id] = actor
	}
}

func (w *World) RemoveActor(id uint32) {
	delete(w.Actors, id)
}

func (w *World) UpsertItem(item FloorItem) {
	if item.ID == 0 {
		return
	}
	if item.DroppedAt.IsZero() {
		item.DroppedAt = time.Now()
	}
	if w.Items == nil {
		w.Items = make(map[uint32]FloorItem)
	}
	w.Items[item.ID] = item
}

func (w *World) RemoveItem(id uint32) {
	delete(w.Items, id)
}
