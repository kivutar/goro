package session

import "time"

type Session struct {
	AccountID            uint32
	CharID               uint32
	AuthCode             uint32
	UserLevel            uint32
	Sex                  byte
	Playing              bool
	NoShift              bool
	NoCtrl               bool
	LessEffects          bool
	HomunculusCustomAI   bool
	HomunculusAggressive bool
	MercenaryCustomAI    bool
	MercenaryAggressive  bool
	ShowEquip            bool
	GuildID              uint32
	EmblemVersion        uint32
	GuildName            string
	Guild                Guild
	PendingGuildName     string
	SnapTargets          bool
	SnapItems            bool
	AttackRange          int
	CharServers          []CharServer
	Characters           []Character
	Selected             Character
	Zone                 ZoneServer
	ServerTick           uint32
	ServerTickAt         time.Time
	PlayerX              int
	PlayerY              int
	PlayerDir            int
	Vitals               Vitals
	Progress             Progress
	Inventory            Inventory
	Storage              Storage
	Cart                 Cart
	Stats                Stats
	Skills               Skills
	Hotkeys              Hotkeys
	Statuses             Statuses
	Friends              Friends
	Whisper              WhisperSettings
	Party                Party
	Movement             Movement
	Homunculus           Companion
	Mercenary            Companion
}

func New() *Session {
	return &Session{Whisper: DefaultWhisperSettings()}
}

func (s *Session) SelectCharacter(character Character) {
	if s == nil {
		return
	}
	s.CharID = character.ID
	s.Selected = character
	s.AttackRange = 0
	s.ShowEquip = false
	s.GuildID = 0
	s.EmblemVersion = 0
	s.GuildName = ""
	s.Guild = Guild{}
	s.PendingGuildName = ""
	s.Zone = ZoneServer{}
	s.ServerTick = 0
	s.ServerTickAt = time.Time{}
	s.PlayerX = 0
	s.PlayerY = 0
	s.PlayerDir = 0
	s.Vitals = VitalsFromCharacter(character)
	s.Progress = ProgressFromCharacter(character)
	s.Inventory = Inventory{Zeny: character.Money}
	s.Storage = Storage{}
	s.Cart = Cart{}
	s.Stats = StatsFromCharacter(character)
	s.Skills = Skills{}
	s.Hotkeys = Hotkeys{}
	s.Statuses = Statuses{}
	s.Friends = Friends{}
	s.Party = Party{}
	s.Movement = Movement{}
	s.Homunculus = Companion{}
	s.Mercenary = Companion{}
}

func (s *Session) SelectedCharacter() Character {
	if s == nil {
		return Character{Name: "Player"}
	}
	if s.Selected.ID != 0 {
		return s.Selected
	}
	for _, character := range s.Characters {
		if character.ID == s.CharID {
			return character
		}
	}
	if len(s.Characters) > 0 {
		return s.Characters[0]
	}
	return Character{ID: s.CharID, Name: "Player", Job: 0}
}

func VitalsFromCharacter(character Character) Vitals {
	return Vitals{
		HP:    int(character.HP),
		MaxHP: int(character.MaxHP),
		SP:    int(character.SP),
		MaxSP: int(character.MaxSP),
	}
}

func ProgressFromCharacter(character Character) Progress {
	return Progress{
		BaseLevel: int(character.Level),
		JobLevel:  int(character.JobLevel),
	}
}

func StatsFromCharacter(character Character) Stats {
	return Stats{
		Str: int(character.Str),
		Agi: int(character.Agi),
		Vit: int(character.Vit),
		Int: int(character.Int),
		Dex: int(character.Dex),
		Luk: int(character.Luk),
	}
}

func (s *Session) SyncServerTick(tick uint32, at time.Time) {
	if s == nil {
		return
	}
	s.ServerTick = tick
	s.ServerTickAt = at
}

func (s *Session) EstimatedServerTick(at time.Time) (uint32, bool) {
	if s == nil || s.ServerTickAt.IsZero() {
		return 0, false
	}
	elapsed := at.Sub(s.ServerTickAt)
	if elapsed < 0 {
		elapsed = 0
	}
	return s.ServerTick + uint32(elapsed/time.Millisecond), true
}

func (s *Session) ElapsedSinceServerTick(start uint32, at time.Time) (time.Duration, bool) {
	current, ok := s.EstimatedServerTick(at)
	if !ok {
		return 0, false
	}
	elapsedMS := current - start
	return time.Duration(elapsedMS) * time.Millisecond, true
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
	Option    uint32
}

type Guild struct {
	ID               uint32
	IsMaster         bool
	Level            uint32
	UserNum          uint32
	MaxUserNum       uint32
	UserAverageLevel uint32
	Exp              uint32
	MaxExp           uint32
	Point            uint32
	Honor            uint32
	Virtue           uint32
	EmblemVersion    uint32
	Name             string
	MasterName       string
	ManageLand       string
	Zeny             uint32
	Members          []GuildMember
	Positions        []GuildPosition
	SkillPoints      int
	Skills           []Skill
	ExpelHistory     []GuildExpelHistory
	NoticeSubject    string
	Notice           string
}

type GuildMember struct {
	AccountID    uint32
	CharID       uint32
	HeadType     uint16
	HeadPalette  uint16
	Sex          uint16
	Job          uint16
	Level        uint16
	MemberExp    uint32
	CurrentState uint32
	PositionID   uint32
	Memo         string
	CharName     string
}

type GuildPosition struct {
	PositionID uint32
	Right      uint32
	Ranking    uint32
	PayRate    uint32
	PosName    string
}

type GuildExpelHistory struct {
	CharName string
	Account  string
	Reason   string
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

type Cart struct {
	Open      bool
	Amount    int
	MaxAmount int
	Weight    int
	MaxWeight int
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
	Cards      [4]uint16
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

type Companion struct {
	ID           uint32
	Active       bool
	Name         string
	Flags        uint8
	Level        int
	Hunger       int
	Intimacy     int
	ItemID       uint32
	Faith        int
	SummonCount  int
	Calls        uint32
	Kills        uint32
	ExpireTick   uint32
	Attack       int
	MagicAttack  int
	Hit          int
	Critical     int
	Defense      int
	MagicDefense int
	Flee         int
	ASPD         int
	HP           int
	MaxHP        int
	SP           int
	MaxSP        int
	Exp          uint64
	MaxExp       uint64
	AttackRange  int
	Job          int16
	Skills       Skills
}

type HotkeySlot struct {
	Type  uint8
	ID    uint32
	Level uint16
}

type Hotkeys struct {
	Loaded  bool
	Version int
	Slots   []HotkeySlot
}

type Movement struct {
	ServerSpeed    int
	HasServerSpeed bool
}

type Statuses struct {
	Active map[uint16]StatusEffect
}

type Friends struct {
	List []Friend
}

type WhisperSettings struct {
	OpenStrangers bool
	OpenFriends   bool
	Alert         bool
	Configured    bool
}

func DefaultWhisperSettings() WhisperSettings {
	return WhisperSettings{
		OpenStrangers: true,
		OpenFriends:   true,
		Alert:         true,
		Configured:    true,
	}
}

type Friend struct {
	AccountID uint32
	CharID    uint32
	Name      string
	State     uint8
}

func (f Friend) Online() bool {
	return f.State == 0
}

type Party struct {
	Name             string
	Members          []PartyMember
	ExpShare         uint32
	ItemPickupRule   uint8
	ItemDivisionRule uint8
	RefuseInvites    bool
}

func (p Party) Active() bool {
	return p.Name != "" || len(p.Members) > 0
}

type PartyMember struct {
	AccountID uint32
	Name      string
	MapName   string
	Role      uint32
	State     uint8
	X         int
	Y         int
	HP        int
	MaxHP     int
	Dead      bool
}

func (m PartyMember) Online() bool {
	return m.State == 0
}

func (m PartyMember) Leader() bool {
	return m.Role == 0
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
