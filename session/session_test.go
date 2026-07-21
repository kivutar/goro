package session

import (
	"testing"
	"time"
)

func TestSelectCharacterClearsCharacterScopedState(t *testing.T) {
	s := New()
	s.AccountID = 2000000
	s.AuthCode = 123
	s.UserLevel = 99
	s.Sex = 1
	s.NoShift = true
	s.NoCtrl = true
	s.LessEffects = true
	s.HomunculusCustomAI = true
	s.HomunculusAggressive = true
	s.MercenaryCustomAI = true
	s.MercenaryAggressive = true
	s.SnapTargets = true
	s.SnapItems = true
	s.Whisper = WhisperSettings{OpenFriends: true, Configured: true}
	s.CharServers = []CharServer{{Name: "local"}}
	s.Characters = []Character{{ID: 150000, Name: "Old"}, {ID: 150001, Name: "New"}}

	s.AttackRange = 14
	s.ShowEquip = true
	s.Zone = ZoneServer{MapName: "prontera"}
	s.ServerTick = 100
	s.ServerTickAt = time.Now()
	s.PlayerX = 10
	s.PlayerY = 20
	s.PlayerDir = 4
	s.Inventory = Inventory{Zeny: 999, Weight: 100, MaxWeight: 1000, Items: []InventoryItem{{Index: 1, ItemID: 512, Amount: 3}}}
	s.Storage = Storage{Open: true, Items: []InventoryItem{{Index: 2, ItemID: 909, Amount: 1}}}
	s.Cart = Cart{Open: true, Items: []InventoryItem{{Index: 3, ItemID: 501, Amount: 2}}}
	s.Stats = Stats{Points: 7, Str: 99}
	s.Skills = Skills{Points: 3, List: []Skill{{ID: 19, Level: 10}}}
	s.Hotkeys = Hotkeys{Loaded: true, Version: 1, Slots: []HotkeySlot{{Type: 1, ID: 19, Level: 10}}}
	s.Statuses = Statuses{Active: map[uint16]StatusEffect{10: {ID: 10}}}
	s.Friends = Friends{List: []Friend{{Name: "Friend"}}}
	s.Party = Party{Name: "Party", Members: []PartyMember{{AccountID: 1, Name: "Old"}}}
	s.Movement = Movement{ServerSpeed: 120, HasServerSpeed: true}

	s.SelectCharacter(Character{
		ID:       150001,
		Money:    42,
		Name:     "New",
		Level:    12,
		JobLevel: 8,
		HP:       100,
		MaxHP:    120,
		SP:       30,
		MaxSP:    40,
		Str:      1,
		Agi:      2,
		Vit:      3,
		Int:      4,
		Dex:      5,
		Luk:      6,
	})

	if s.AccountID != 2000000 || s.AuthCode != 123 || s.UserLevel != 99 || s.Sex != 1 {
		t.Fatalf("account state was not preserved: %+v", s)
	}
	if !s.NoShift || !s.NoCtrl || !s.LessEffects || !s.HomunculusCustomAI || !s.HomunculusAggressive || !s.MercenaryCustomAI || !s.MercenaryAggressive || !s.SnapTargets || !s.SnapItems || !s.Whisper.Configured {
		t.Fatalf("client settings were not preserved: %+v", s)
	}
	if len(s.CharServers) != 1 || len(s.Characters) != 2 {
		t.Fatalf("character-select state was not preserved: servers=%+v characters=%+v", s.CharServers, s.Characters)
	}
	if s.CharID != 150001 || s.Selected.ID != 150001 || s.Selected.Name != "New" {
		t.Fatalf("selected character not applied: char_id=%d selected=%+v", s.CharID, s.Selected)
	}
	if s.Inventory.Zeny != 42 || len(s.Inventory.Items) != 0 || s.Inventory.Weight != 0 || s.Inventory.MaxWeight != 0 {
		t.Fatalf("inventory was not reset to selected character money: %+v", s.Inventory)
	}
	if s.Vitals.HP != 100 || s.Vitals.MaxHP != 120 || s.Vitals.SP != 30 || s.Vitals.MaxSP != 40 {
		t.Fatalf("vitals not seeded: %+v", s.Vitals)
	}
	if s.Progress.BaseLevel != 12 || s.Progress.JobLevel != 8 {
		t.Fatalf("progress not seeded: %+v", s.Progress)
	}
	if s.Stats.Str != 1 || s.Stats.Agi != 2 || s.Stats.Vit != 3 || s.Stats.Int != 4 || s.Stats.Dex != 5 || s.Stats.Luk != 6 || s.Stats.Points != 0 {
		t.Fatalf("stats not reset and seeded: %+v", s.Stats)
	}
	if s.AttackRange != 0 || s.ShowEquip || s.Zone != (ZoneServer{}) || s.ServerTick != 0 || !s.ServerTickAt.IsZero() || s.PlayerX != 0 || s.PlayerY != 0 || s.PlayerDir != 0 {
		t.Fatalf("runtime state leaked: %+v", s)
	}
	if len(s.Storage.Items) != 0 || s.Storage.Open || len(s.Cart.Items) != 0 || s.Cart.Open || len(s.Skills.List) != 0 || s.Skills.Points != 0 || len(s.Hotkeys.Slots) != 0 || s.Hotkeys.Loaded || len(s.Statuses.Active) != 0 || len(s.Friends.List) != 0 || s.Party.Active() || s.Movement.HasServerSpeed {
		t.Fatalf("character scoped state leaked: storage=%+v cart=%+v skills=%+v hotkeys=%+v statuses=%+v friends=%+v party=%+v movement=%+v", s.Storage, s.Cart, s.Skills, s.Hotkeys, s.Statuses, s.Friends, s.Party, s.Movement)
	}
}

func TestProgressFromCharacterUsesBaseLevel(t *testing.T) {
	progress := ProgressFromCharacter(Character{Level: 12, JobLevel: 7})
	if progress.BaseLevel != 12 || progress.JobLevel != 7 {
		t.Fatalf("progress = %+v", progress)
	}
}

func TestSelectedCharacterFallsBackToCharID(t *testing.T) {
	s := &Session{
		CharID: 150001,
		Characters: []Character{
			{ID: 150000, Name: "Old"},
			{ID: 150001, Name: "New"},
		},
	}

	if got := s.SelectedCharacter(); got.ID != 150001 || got.Name != "New" {
		t.Fatalf("selected character = %+v, want New", got)
	}
}

func TestServerTickEstimation(t *testing.T) {
	s := New()
	start := time.Unix(100, 0)
	s.SyncServerTick(1000, start)

	tick, ok := s.EstimatedServerTick(start.Add(250 * time.Millisecond))
	if !ok {
		t.Fatal("server tick not estimated")
	}
	if tick != 1250 {
		t.Fatalf("tick = %d, want 1250", tick)
	}

	elapsed, ok := s.ElapsedSinceServerTick(1100, start.Add(250*time.Millisecond))
	if !ok {
		t.Fatal("server tick elapsed not estimated")
	}
	if elapsed != 150*time.Millisecond {
		t.Fatalf("elapsed = %s, want 150ms", elapsed)
	}
}
