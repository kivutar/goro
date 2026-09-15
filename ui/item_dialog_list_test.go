package ui

import (
	"slices"
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
)

func TestItemDialogListTracksInventoryChanges(t *testing.T) {
	mutations := []struct {
		name   string
		change func(*session.InventoryItem)
	}{
		{"amount", func(item *session.InventoryItem) { item.Amount++ }},
		{"refine", func(item *session.InventoryItem) { item.Refine++ }},
		{"cards", func(item *session.InventoryItem) { item.Cards[2] = 4001 }},
		{"identified", func(item *session.InventoryItem) { item.Identified = true }},
		{"equipped", func(item *session.InventoryItem) { item.Equipped = true }},
		{"item_id", func(item *session.InventoryItem) { item.ItemID++ }},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
				{Index: 4, ItemID: 2301, Type: db.ItemTypeArmor, Amount: 1},
				{Index: 2, ItemID: 1201, Type: db.ItemTypeWeapon, Amount: 1},
			}}}
			var list itemDialogList
			indexes := []uint16{4, 2}
			before := list.get(s, indexes, false)
			if len(before) != 2 || before[0].Index != 2 || before[1].Index != 4 {
				t.Fatalf("initial list = %+v, want sorted indexes 2, 4", before)
			}
			original := slices.Clone(before)
			mutation.change(&s.Inventory.Items[0])
			after := list.get(s, indexes, false)
			if len(after) != 2 || after[1] != s.Inventory.Items[0] {
				t.Fatalf("updated list = %+v, want updated item %+v", after, s.Inventory.Items[0])
			}
			if !slices.Equal(before, original) {
				t.Fatal("refresh modified the previous table's captured items")
			}
		})
	}
}

func TestItemDialogListTracksServerListAndRemoval(t *testing.T) {
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 2, ItemID: 2301}, {Index: 3, ItemID: 2302}, {Index: 4, ItemID: 2303},
	}}}
	var list itemDialogList
	indexes := []uint16{4, 2}
	list.get(s, indexes, false)
	indexes[1] = 3
	items := list.get(s, indexes, false)
	if len(items) != 2 || items[0].Index != 3 || items[1].Index != 4 {
		t.Fatalf("changed server list = %+v, want indexes 3, 4", items)
	}
	s.Inventory.Items = s.Inventory.Items[:2]
	items = list.get(s, indexes, false)
	if len(items) != 1 || items[0].Index != 3 {
		t.Fatalf("list after removal = %+v, want index 3", items)
	}
	if items := list.get(nil, indexes, false); len(items) != 0 {
		t.Fatalf("items without inventory = %+v", items)
	}
	if items := list.get(s, indexes, false); len(items) != 1 || items[0].Index != 3 {
		t.Fatalf("restored inventory = %+v, want index 3", items)
	}
}

func TestItemDialogListTracksAppraisalEligibility(t *testing.T) {
	s := &session.Session{Inventory: session.Inventory{Items: []session.InventoryItem{
		{Index: 2, ItemID: 2301, Type: db.ItemTypeArmor},
		{Index: 3, ItemID: 512, Type: db.ItemTypeUsable},
		{Index: 4, ItemID: 1201, Type: db.ItemTypeWeapon, Identified: true},
	}}}
	var list itemDialogList
	indexes := []uint16{4, 3, 2, 9}
	if items := list.get(s, indexes, true); len(items) != 1 || items[0].Index != 2 {
		t.Fatalf("appraisal list = %+v, want index 2", items)
	}
	s.Inventory.Items[0].Identified = true
	s.Inventory.Items[1].Equip = true
	s.Inventory.Items[2].Identified = false
	items := list.get(s, indexes, true)
	if len(items) != 2 || items[0].Index != 3 || items[1].Index != 4 {
		t.Fatalf("updated appraisal list = %+v, want indexes 3, 4", items)
	}
}

func TestItemDialogListCopiesInvalidateIndependently(t *testing.T) {
	for _, change := range []string{"inventory", "indexes"} {
		t.Run(change, func(t *testing.T) {
			ctx, indexes := itemDialogBenchmarkContext(3)
			var original itemDialogList
			original.get(ctx.Session, indexes, false)
			next := original
			if change == "inventory" {
				ctx.Session.Inventory.Items[0].Refine = 7
			} else {
				indexes[0] = indexes[1]
			}
			want := original.get(ctx.Session, indexes, false)
			if got := next.get(ctx.Session, indexes, false); !slices.Equal(got, want) {
				t.Fatalf("copied cache = %+v, want refreshed items %+v", got, want)
			}
		})
	}
}

func TestCardCompositionCopyRefreshesAfterOldCallback(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(3)
	var original CardCompositionWindow
	original.OpenList(ctx, 105, network.ItemCompositionList{Indexes: indexes})
	next := original // WorldMode.nextWorldMode copies persistent windows.
	ctx.Session.Inventory.Items[0].Refine = 7
	// An old button callback can run before the copied window's next Update.
	original.composeSelected(ctx)
	next.Update(ctx)
	if next.snapshot[0].Refine != 7 {
		t.Fatalf("copied window kept stale item: %+v", next.snapshot[0])
	}
}

func itemDialogBenchmarkContext(count int) (Context, []uint16) {
	ctx := Context{Input: input.NewState(), Session: session.New(), ScreenW: 800, ScreenH: 600}
	indexes := make([]uint16, count)
	for i := range count {
		index := uint16(i + 2)
		ctx.Session.Inventory.Items = append(ctx.Session.Inventory.Items, session.InventoryItem{
			Index: index, ItemID: 2301, Type: db.ItemTypeArmor, Amount: 1,
		})
		indexes[i] = index
	}
	return ctx, indexes
}

func TestCardCompositionIdleKeepsContentWithoutAllocations(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(100)
	var w CardCompositionWindow
	w.OpenList(ctx, 105, network.ItemCompositionList{Indexes: indexes})
	w.ensureScrollSignal().Set(64)
	content := w.content
	if allocs := testing.AllocsPerRun(100, func() { w.Update(ctx) }); allocs != 0 {
		t.Fatalf("idle update allocations = %v, want 0", allocs)
	}
	if w.content != content || w.ensureScrollSignal().Get() != 64 {
		t.Fatal("idle update rebuilt content or reset scroll")
	}
	ctx.Session.Inventory.Items[0].Refine++
	w.Update(ctx)
	if w.content == content || w.snapshot[0].Refine != 1 || w.ensureScrollSignal().Get() != 64 {
		t.Fatal("refine update did not refresh content while preserving scroll")
	}
	content = w.content
	ctx.Session.Inventory.Items[0].Cards[0] = 4001
	w.Update(ctx)
	if w.content == content || w.snapshot[0].Cards[0] != 4001 {
		t.Fatal("card update did not refresh content")
	}
	ctx.Session.Inventory.Items = ctx.Session.Inventory.Items[:3]
	w.Update(ctx)
	if len(w.snapshot) != 3 || w.ensureScrollSignal().Get() != 0 {
		t.Fatal("removal did not refresh content and clamp scroll")
	}
}

func TestIdentifyIdleKeepsSelectionWithoutAllocations(t *testing.T) {
	ctx, indexes := itemDialogBenchmarkContext(100)
	var w IdentifyWindow
	w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
	w.selectedRow = 5
	w.ensureScrollSignal().Set(64)
	content := w.content
	if allocs := testing.AllocsPerRun(100, func() { w.Update(ctx) }); allocs != 0 {
		t.Fatalf("idle update allocations = %v, want 0", allocs)
	}
	if w.content != content || w.selectedRow != 5 || w.ensureScrollSignal().Get() != 64 {
		t.Fatal("idle update rebuilt content or reset selection/scroll")
	}
	ctx.Session.Inventory.Items[0].Refine++
	w.Update(ctx)
	if w.content == content || w.snapshot[0].Refine != 1 || w.selectedRow != 5 || w.ensureScrollSignal().Get() != 64 {
		t.Fatal("refine update did not refresh content while preserving selection/scroll")
	}
	ctx.Session.Inventory.Items = ctx.Session.Inventory.Items[:3]
	w.Update(ctx)
	if len(w.snapshot) != 3 || w.selectedRow != -1 || w.ensureScrollSignal().Get() != 0 {
		t.Fatal("removal did not refresh content and clamp selection/scroll")
	}
	ctx.Session.Inventory.Items[0].Identified = true
	w.Update(ctx)
	if len(w.snapshot) != 2 || w.snapshot[0].Index != indexes[1] {
		t.Fatalf("inventory update left stale items: %+v", w.snapshot)
	}
}

func BenchmarkCardCompositionIdle(b *testing.B) {
	ctx, indexes := itemDialogBenchmarkContext(100)
	var w CardCompositionWindow
	w.OpenList(ctx, 105, network.ItemCompositionList{Indexes: indexes})
	b.ReportAllocs()
	for b.Loop() {
		w.Update(ctx)
	}
}

func BenchmarkIdentifyIdle(b *testing.B) {
	ctx, indexes := itemDialogBenchmarkContext(100)
	var w IdentifyWindow
	w.OpenList(ctx, network.ItemIdentifyList{Indexes: indexes})
	b.ReportAllocs()
	for b.Loop() {
		w.Update(ctx)
	}
}
