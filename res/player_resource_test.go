package res

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestPlayerSexTokenUsesRagnarokSexEnum(t *testing.T) {
	if got := PlayerSexToken(0); got != playerFemaleSex {
		t.Fatalf("sex 0 token = %q, want female", got)
	}
	if got := PlayerSexToken(1); got != playerMaleSex {
		t.Fatalf("sex 1 token = %q, want male", got)
	}
}

func TestHasPlayerJobToken(t *testing.T) {
	if !HasPlayerJobToken(0) {
		t.Fatal("novice job token missing")
	}
	if HasPlayerJobToken(1002) {
		t.Fatal("unknown job token should not report as renderable")
	}
}

func TestPlayerIMFResourceCandidates(t *testing.T) {
	got := PlayerIMFResourceCandidates(1, 1)
	want := "data\\imf\\검사_남.imf"
	if len(got) == 0 || got[0] != want {
		t.Fatalf("first imf candidate = %q, want %q", got, want)
	}
	if got[len(got)-1] != "data\\imf\\초보자_남.imf" {
		t.Fatalf("fallback imf candidate = %q", got[len(got)-1])
	}
}

func TestPlayerCartResourceCandidates(t *testing.T) {
	got := PlayerCartResourceCandidates(1, "spr")
	want := "data\\sprite\\이팩트\\손수레.spr"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("cart 1 candidate = %q, want %q", got, want)
	}
	got = PlayerCartResourceCandidates(13, "act")
	want = "data\\sprite\\이팩트\\마도카트.act"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("cart 13 candidate = %q, want %q", got, want)
	}
}

func TestPlayerWeaponOverlayResourceCandidates(t *testing.T) {
	got := PlayerWeaponOverlayResourceCandidates(0, 1, 1201, false, "act")
	want := []string{
		"data\\sprite\\인간족\\초보자\\초보자_남_1201.act",
		"data\\sprite\\인간족\\초보자\\초보자_남_단검.act",
	}
	if len(got) != len(want) {
		t.Fatalf("weapon overlay = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("weapon overlay[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMercenaryWeaponOverlayResourceCandidates(t *testing.T) {
	got := MercenaryWeaponOverlayResourceCandidates(`남\검용병`, db.WeaponSword, false, "act")
	want := `data\sprite\인간족\용병\검용병_검.act`
	if len(got) != 1 || got[0] != want {
		t.Fatalf("sword mercenary overlay = %q, want %q", got, want)
	}

	got = MercenaryWeaponOverlayResourceCandidates(`남\검용병`, db.WeaponTwoHandSword, true, "spr")
	want = `data\sprite\인간족\용병\검용병_검_검광.spr`
	if len(got) != 1 || got[0] != want {
		t.Fatalf("sword mercenary light overlay = %q, want %q", got, want)
	}

	got = MercenaryWeaponOverlayResourceCandidates(`남\창용병`, db.WeaponTwoHandSpear, false, "act")
	want = `data\sprite\인간족\용병\창용병_창.act`
	if len(got) != 1 || got[0] != want {
		t.Fatalf("lancer mercenary overlay = %q, want %q", got, want)
	}

	if got := MercenaryWeaponOverlayResourceCandidates(`여\활용병`, db.WeaponBow, true, "act"); len(got) != 0 {
		t.Fatalf("bow mercenary light overlay = %q, want none", got)
	}
}

func TestPlayerWeaponViewIDUsesItemClassNumBeforeFallbackRange(t *testing.T) {
	manager := &Manager{
		itemMetadataLoaded: true,
		itemMetadata: map[int]ItemMetadata{
			1607: {ClassNum: 10, ClassNumSet: true},
			1615: {ClassNum: 70, ClassNumSet: true},
		},
	}
	if got := manager.PlayerWeaponViewID(1607); got != 10 {
		t.Fatalf("weapon view id 1607 = %d, want class num 10", got)
	}
	if got := manager.PlayerWeaponViewID(1615); got != 70 {
		t.Fatalf("weapon view id 1615 = %d, want class num 70", got)
	}
	if got := manager.PlayerWeaponViewID(1701); got != 11 {
		t.Fatalf("weapon view id fallback 1701 = %d, want bow type 11", got)
	}
}

func TestPlayerWeaponOverlayTypeForJobMatchesReferenceJobRules(t *testing.T) {
	if got := PlayerWeaponOverlayTypeForJob(2, 5, false); got != 10 {
		t.Fatalf("mage overlay type for weapon 5 = %d, want rod type 10", got)
	}
	if got := PlayerWeaponOverlayTypeForJob(16, 5, false); got != 10 {
		t.Fatalf("sage overlay type for weapon 5 = %d, want rod type 10", got)
	}
	if got := PlayerWeaponOverlayTypeForJob(1, 5, false); got != 5 {
		t.Fatalf("swordman overlay type for weapon 5 = %d, want spear type 5", got)
	}
}

func TestPlayerWeaponOverlayTokenUsesClientRodSpelling(t *testing.T) {
	if got := PlayerWeaponOverlayToken(10); got != "롯드" {
		t.Fatalf("rod token = %q, want client spelling", got)
	}
}

func TestNormalizePlayerWeaponShieldMovesLeftHandWeapon(t *testing.T) {
	weapon, shield := NormalizePlayerWeaponShield(0, 1601)
	if weapon != 1601 || shield != 0 {
		t.Fatalf("normalized left-hand weapon = weapon %d shield %d, want 1601/0", weapon, shield)
	}
	weapon, shield = NormalizePlayerWeaponShield(0, 2101)
	if weapon != 0 || shield != 2101 {
		t.Fatalf("normalized real shield = weapon %d shield %d, want 0/2101", weapon, shield)
	}
}

func TestPlayerShieldOverlayResourceCandidates(t *testing.T) {
	got := PlayerShieldOverlayResourceCandidates(0, 1, 2101, "spr")
	want := "data\\sprite\\방패\\초보자\\초보자_남_가드.spr"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("shield overlay = %q, want %q", got, want)
	}
}

func TestPlayerAccessoryResourceCandidates(t *testing.T) {
	got := PlayerAccessoryResourceCandidates(0, 3, 0, 100, "sample", "act")
	want := "data\\sprite\\악세사리\\여\\여_sample.act"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("accessory overlay = %q, want %q", got, want)
	}
}
