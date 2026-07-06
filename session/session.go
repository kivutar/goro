package session

import "time"

type Session struct {
	AccountID   uint32
	CharID      uint32
	AuthCode    uint32
	UserLevel   uint32
	Sex         byte
	Playing     bool
	NoShift     bool
	AttackRange int
	CharServers []CharServer
	Characters  []Character
	Selected    Character
	Zone        ZoneServer
	PlayerX     int
	PlayerY     int
	PlayerDir   int
	Vitals      Vitals
	Progress    Progress
	Inventory   Inventory
	Storage     Storage
	Stats       Stats
	Skills      Skills
	Statuses    Statuses
}

func New() *Session {
	return &Session{}
}

type CharServer struct {
	Address   string
	Port      uint16
	Name      string
	UserCount uint16
	State     uint16
	Property  uint16
}

type Character struct {
	ID        uint32
	Money     int64
	Name      string
	Slot      uint8
	Level     int16
	JobLevel  int16
	Job       int16
	HP        int16
	MaxHP     int16
	SP        int16
	MaxSP     int16
	Str       uint8
	Agi       uint8
	Vit       uint8
	Int       uint8
	Dex       uint8
	Luk       uint8
	Hair      int16
	HairColor uint8
	HeadPal   int16
	BodyPal   int16
	Weapon    int16
	Shield    int16
	HeadTop   int16
	HeadMid   int16
	HeadLow   int16
}

type Vitals struct {
	HP    int
	MaxHP int
	SP    int
	MaxSP int
}

type Progress struct {
	BaseLevel   int
	JobLevel    int
	BaseExp     int64
	NextBaseExp int64
	JobExp      int64
	NextJobExp  int64
}

type Inventory struct {
	Zeny      int64
	Weight    int
	MaxWeight int
	Items     []InventoryItem
}

type Storage struct {
	Open      bool
	Amount    int
	MaxAmount int
	Items     []InventoryItem
}

type InventoryItem struct {
	Index      uint16
	ItemID     uint16
	Type       uint8
	Location   uint16
	Identified bool
	Amount     int
	Equip      bool
	Equipped   bool
	Damaged    bool
	Refine     uint8
}

type Stats struct {
	Points int
	Str    int
	Agi    int
	Vit    int
	Int    int
	Dex    int
	Luk    int

	StrBonus int
	AgiBonus int
	VitBonus int
	IntBonus int
	DexBonus int
	LukBonus int

	StrCost int
	AgiCost int
	VitCost int
	IntCost int
	DexCost int
	LukCost int

	Attack        int
	AttackBonus   int
	MatkMin       int
	MatkMax       int
	Defense       int
	DefenseBonus  int
	MDefense      int
	MDefenseBonus int
	Hit           int
	Flee          int
	FleeBonus     int
	Critical      int
	ASPD          int
	ASPDBonus     int
}

type Skills struct {
	Points int
	List   []Skill
}

type Skill struct {
	ID         uint16
	Type       uint32
	Level      int
	MaxLevel   int
	SPCost     int
	Range      int
	Name       string
	Upgradable bool
}

type Statuses struct {
	Active map[uint16]StatusEffect
}

type StatusEffect struct {
	ID          uint16
	Source      uint32
	StartedAt   time.Time
	ExpiresAt   time.Time
	HasDuration bool
}

type ZoneServer struct {
	Address string
	Port    uint16
	MapName string
}
