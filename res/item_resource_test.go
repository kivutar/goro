package res

import (
	"strconv"
	"testing"
)

func TestParseItemPairTable(t *testing.T) {
	got := parseItemPairTable([]byte("// comment\n909#Jellopy#\r\n# comment\n0#ignored#\n1002#Poring_Card#extra\n"))
	if got[909] != "Jellopy" {
		t.Fatalf("item 909 = %q", got[909])
	}
	if got[1002] != "Poring_Card" {
		t.Fatalf("item 1002 = %q", got[1002])
	}
	if _, ok := got[0]; ok {
		t.Fatal("zero id should be ignored")
	}
}

func TestParseItemPairTableDecodesEUC(t *testing.T) {
	got := parseItemPairTable([]byte{0x32, 0x32, 0x32, 0x36, 0x23, 0xc4, 0xb8, 0x23, 0x0a})
	if got[2226] != "캡" {
		t.Fatalf("item 2226 = %q, want 캡", got[2226])
	}
}

func TestParseItemIDSetTable(t *testing.T) {
	got := parseItemIDSetTable([]byte("// comment\n4001#\n0#\n4002#ignored\n"))
	if _, ok := got[4001]; !ok {
		t.Fatal("missing id 4001")
	}
	if _, ok := got[4002]; !ok {
		t.Fatal("missing id 4002")
	}
	if _, ok := got[0]; ok {
		t.Fatal("zero id should be ignored")
	}
}

func TestParseItemDescriptionTable(t *testing.T) {
	got := parseItemDescriptionTable([]byte("// comment\n909#\r\nSmall_Jellopy.\n^0000FFColor^000000 text.\n#\n# comment\n1002# trailing text ignored\n1003#\nSingle line.\n#\n"))
	if lines := got[909]; len(lines) != 2 || lines[0] != "Small_Jellopy." || lines[1] != "^0000FFColor^000000 text." {
		t.Fatalf("item 909 description = %#v", lines)
	}
	if _, ok := got[1002]; ok {
		t.Fatal("header with trailing text should be ignored")
	}
	if lines := got[1003]; len(lines) != 1 || lines[0] != "Single line." {
		t.Fatalf("item 1003 description = %#v", lines)
	}
}

func TestNormalizeItemDisplayToken(t *testing.T) {
	if got := normalizeItemDisplayToken("Poring_Card"); got != "Poring Card" {
		t.Fatalf("display = %q", got)
	}
}

func TestItemSpriteResourceCandidates(t *testing.T) {
	got := ItemSpriteResourceCandidates("apple", "spr")
	want := "data\\sprite\\아이템\\apple.spr"
	if len(got) == 0 || got[0] != want {
		t.Fatalf("leading candidates = %#v", got[:minIntForTest(len(got), 2)])
	}
	found := false
	for _, candidate := range got {
		if candidate == "apple.spr" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing bare fallback in %#v", got)
	}
}

func TestItemCollectionTextureCandidates(t *testing.T) {
	got := ItemCollectionTextureCandidates("apple")
	want := "data\\texture\\유저인터페이스\\collection\\apple.bmp"
	found := false
	for _, candidate := range got {
		if candidate == want {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing collection candidate %q in %#v", want, got)
	}
}

func TestCardIllustrationTextureCandidates(t *testing.T) {
	got := CardIllustrationTextureCandidates("poring_card")
	want := "data\\texture\\유저인터페이스\\cardbmp\\poring_card.bmp"
	if len(got) == 0 || got[0] != want {
		t.Fatalf("card illustration candidates = %#v, want %q first", got, want)
	}
}

func TestItemMetadataLookupFallbacks(t *testing.T) {
	manager := &Manager{
		itemMetadataLoaded: true,
		itemMetadata: map[int]ItemMetadata{
			909: {
				UnidentifiedDisplayName: "Unknown Item",
				IdentifiedDisplayName:   "Jellopy",
				IdentifiedResource:      "jellopy",
				IdentifiedDescription:   []string{"A tiny crystalline item."},
				SlotCount:               1,
				ClassNum:                10,
				ClassNumSet:             true,
			},
			4001: {
				CardIllustration: "poring_card",
			},
		},
	}
	if got, ok := manager.ItemDisplayName(909, true); !ok || got != "Jellopy" {
		t.Fatalf("identified display = %q ok=%v", got, ok)
	}
	if got, ok := manager.ItemDisplayName(909, false); !ok || got != "Unknown Item" {
		t.Fatalf("unidentified display = %q ok=%v", got, ok)
	}
	if got, ok := manager.ItemResourceName(909, false); !ok || got != "jellopy" {
		t.Fatalf("resource fallback = %q ok=%v", got, ok)
	}
	if got, ok := manager.ItemDescription(909, false); !ok || len(got) != 1 || got[0] != "A tiny crystalline item." {
		t.Fatalf("description fallback = %#v ok=%v", got, ok)
	}
	if got, ok := manager.ItemClassNum(909); !ok || got != 10 {
		t.Fatalf("class num = %d ok=%v, want 10/true", got, ok)
	}
	if got, ok := manager.ItemSlotCount(909); !ok || got != 1 {
		t.Fatalf("slot count = %d ok=%v, want 1/true", got, ok)
	}
	if got, ok := manager.ItemCardIllustrationName(4001); !ok || got != "poring_card" {
		t.Fatalf("card illustration = %q ok=%v, want poring_card/true", got, ok)
	}
	manager.itemMetadata[4001] = ItemMetadata{CardPrefixName: "Poring"}
	if got, ok := manager.ItemCardPrefixName(4001); !ok || got != "Poring" {
		t.Fatalf("card prefix = %q ok=%v", got, ok)
	}
	manager.itemMetadata[4002] = ItemMetadata{CardPrefixName: "Fabre", CardPostfix: true}
	if !manager.ItemCardPostfix(4002) {
		t.Fatal("card postfix not set")
	}
}

func TestParseItemSlotCountTable(t *testing.T) {
	manager := &Manager{
		itemMetadataLoaded: true,
		itemMetadata: map[int]ItemMetadata{
			2607: {IdentifiedDisplayName: "Clip"},
		},
	}
	for id, value := range parseItemPairTable([]byte("2607#1#\n2608#0#\n2609#bad#\n")) {
		slotCount, err := strconv.Atoi(value)
		if err != nil || slotCount <= 0 {
			continue
		}
		metadata := manager.itemMetadata[id]
		metadata.SlotCount = slotCount
		manager.itemMetadata[id] = metadata
	}
	if got, ok := manager.ItemSlotCount(2607); !ok || got != 1 {
		t.Fatalf("slot count = %d ok=%v, want 1/true", got, ok)
	}
	if got, ok := manager.ItemSlotCount(2608); ok || got != 0 {
		t.Fatalf("zero slot count = %d ok=%v, want 0/false", got, ok)
	}
}

func TestFormatGroundItemLabel(t *testing.T) {
	if got := FormatGroundItemLabel("Jellopy", 3); got != "Jellopy: 3 ea" {
		t.Fatalf("label = %q", got)
	}
	if got := FormatGroundItemLabel("", 0); got != "Item: 1 ea" {
		t.Fatalf("fallback label = %q", got)
	}
}

func minIntForTest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
