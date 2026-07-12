package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestShopAddSellCartItemTracksAmount(t *testing.T) {
	window := ShopWindow{
		mode: shopModeSell,
	}

	window.addCartItem(session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, network.ShopSellItem{Index: 7, Price: 10, OverchargePrice: 12})
	window.addCartItem(session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, network.ShopSellItem{Index: 7, Price: 10, OverchargePrice: 12})
	if len(window.cart) != 1 || window.cart[0].amount != 3 || window.cart[0].max != 3 || window.cart[0].over != 12 {
		t.Fatalf("cart = %+v", window.cart)
	}
}

func TestShopBuyCartTracksQuantityAndTotal(t *testing.T) {
	window := ShopWindow{mode: shopModeBuy}
	item := network.ShopBuyItem{ItemID: 501, Price: 100, DiscountPrice: 80}

	window.addBuyItem(item)
	window.addBuyItem(item)
	if got := window.buyCart[0].amount; got != 2 {
		t.Fatalf("buy amount = %d, want 2", got)
	}
	if got := window.total(); got != 160 {
		t.Fatalf("total = %d, want 160", got)
	}

	window.decrementBuyCartRow(0)
	if got := window.buyCart[0].amount; got != 1 {
		t.Fatalf("buy amount after decrement = %d, want 1", got)
	}
}

func TestInventoryBagClassifiesTabs(t *testing.T) {
	tests := []struct {
		name string
		item session.InventoryItem
		tab  int
	}{
		{name: "healing item", item: session.InventoryItem{Type: 0}, tab: inventoryBagTabItem},
		{name: "usable item", item: session.InventoryItem{Type: 2}, tab: inventoryBagTabItem},
		{name: "equipment flag", item: session.InventoryItem{Type: 4, Equip: true}, tab: inventoryBagTabEquip},
		{name: "weapon type", item: session.InventoryItem{Type: 5}, tab: inventoryBagTabEquip},
		{name: "pet egg type", item: session.InventoryItem{Type: 7}, tab: inventoryBagTabEquip},
		{name: "etc", item: session.InventoryItem{Type: 3}, tab: inventoryBagTabEtc},
		{name: "card", item: session.InventoryItem{Type: 6}, tab: inventoryBagTabEtc},
		{name: "ammo", item: session.InventoryItem{Type: 10}, tab: inventoryBagTabEquip},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inventoryItemTab(tc.item); got != tc.tab {
				t.Fatalf("tab = %d, want %d", got, tc.tab)
			}
		})
	}
}

func TestInventoryItemDisplayNameAddsSlotCountForIdentifiedItems(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "idnum2itemdisplaynametable.txt"), []byte("2607#Clip#\n2608#Ring [1]#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "num2itemdisplaynametable.txt"), []byte("2607#Accessory#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "itemslotcounttable.txt"), []byte("2607#1#\n2608#1#\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}

	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2607, Identified: true}); got != "Clip [1]" {
		t.Fatalf("identified slotted name = %q, want Clip [1]", got)
	}
	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2607, Identified: false}); got != "Accessory" {
		t.Fatalf("unidentified slotted name = %q, want Accessory", got)
	}
	if got := inventoryItemDisplayName(manager, session.InventoryItem{ItemID: 2608, Identified: true}); got != "Ring [1]" {
		t.Fatalf("pre-suffixed slotted name = %q, want Ring [1]", got)
	}
}

func TestStorageAcceptInventoryDropWithoutNetworkConsumesDrop(t *testing.T) {
	window := StorageWindow{}
	sessionState := &session.Session{Storage: session.Storage{Open: true}}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)
	ok := window.AcceptInventoryDrop(Context{Session: sessionState}, session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, window.Window.x+12, window.Window.y+20)
	if !ok {
		t.Fatal("drop over storage was not consumed")
	}
}

func TestCartAcceptInventoryDropWithoutNetworkConsumesDrop(t *testing.T) {
	window := CartWindow{}
	sessionState := &session.Session{Cart: session.Cart{Open: true}}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}
	window.OpenWindow(ctx)
	ok := window.AcceptInventoryDrop(Context{Session: sessionState}, session.InventoryItem{Index: 7, ItemID: 938, Amount: 3}, window.Window.x+12, window.Window.y+20)
	if !ok {
		t.Fatal("drop over cart was not consumed")
	}
}

func TestStorageAndCartAcceptCrossDropsWithoutNetwork(t *testing.T) {
	sessionState := &session.Session{
		Storage: session.Storage{Open: true},
		Cart:    session.Cart{Open: true},
	}
	ctx := Context{Session: sessionState, ScreenW: 800, ScreenH: 600}

	storage := StorageWindow{}
	storage.OpenWindow(ctx)
	if !storage.AcceptCartDrop(ctx, session.InventoryItem{Index: 4, ItemID: 938, Amount: 2}, storage.Window.x+12, storage.Window.y+20) {
		t.Fatal("drop from cart over storage was not consumed")
	}

	cart := CartWindow{}
	cart.OpenWindow(ctx)
	if !cart.AcceptStorageDrop(ctx, session.InventoryItem{Index: 5, ItemID: 938, Amount: 2}, cart.Window.x+12, cart.Window.y+20) {
		t.Fatal("drop from storage over cart was not consumed")
	}
}

func TestInventoryBagShowsCartButtonOnlyWhenPlayerHasCart(t *testing.T) {
	if inventoryBagHasCart(Context{}) {
		t.Fatal("empty context should not show cart button")
	}
	world := worldstate.New()
	world.Player.HasCartState = true
	world.Player.HasCart = true
	if !inventoryBagHasCart(Context{World: world}) {
		t.Fatal("player cart state should show cart button")
	}
	if !inventoryBagHasCart(Context{Session: &session.Session{Selected: session.Character{ID: 150004, Option: inventoryBagCartOptionMask}}}) {
		t.Fatal("selected character cart option should show cart button")
	}
	world.Player.HasCart = false
	if inventoryBagHasCart(Context{World: world, Session: &session.Session{Selected: session.Character{ID: 150004, Option: inventoryBagCartOptionMask}}}) {
		t.Fatal("explicit world cart removal should hide cart button")
	}
}

func TestInventoryBagTooltipTracksHoveredItem(t *testing.T) {
	window := InventoryBagWindow{}
	item := session.InventoryItem{Index: 8, ItemID: 501, Identified: true}
	ctx := Context{Input: input.NewState(), ScreenW: 800, ScreenH: 600, UIManager: NewManager()}

	window.showTooltip(ctx, item)
	if !window.tooltip.Open() {
		t.Fatal("tooltip should be open")
	}
	if got := window.tooltip.Text(); got != "item 501" {
		t.Fatalf("tooltip text = %q", got)
	}

	window.hideTooltip()
	if window.tooltip.Open() {
		t.Fatal("tooltip not cleared")
	}
}

func TestStorageDragReleaseOverInventoryWithdraws(t *testing.T) {
	inputState := input.NewState()
	inputState.SetMousePosition(40, 40)
	inputState.SetMouseButton(render.MouseButtonLeft, true)
	inputState.EndFrame()
	inputState.SetMouseButton(render.MouseButtonLeft, false)

	inventory := InventoryBagWindow{}
	inventory.EnsureWindow(inventoryBagWidth, inventoryBagHeight)
	inventory.Window.OpenAt(24, 24, nil)
	storage := StorageWindow{
		dragItem:   session.InventoryItem{Index: 9, ItemID: 938, Amount: 2},
		dragActive: true,
		dragFrom:   time.Now().Add(-time.Second),
	}
	consumed := storage.UpdateDrag(Context{Input: inputState}, &inventory, nil)
	if !consumed {
		t.Fatal("storage drag release was not consumed")
	}
	if storage.dragActive {
		t.Fatal("storage drag stayed active after release")
	}
}

func TestInventoryDropAmountIsOneUnit(t *testing.T) {
	if got := inventoryDropAmount(session.InventoryItem{Amount: 9}); got != 1 {
		t.Fatalf("stack drop amount = %d, want 1", got)
	}
	if got := inventoryDropAmount(session.InventoryItem{}); got != 1 {
		t.Fatalf("zero drop amount = %d, want 1", got)
	}
}

func TestIdentifyWindowShowsOnlyUnidentifiedEquipmentFromServerList(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 512, Type: 0, Identified: false},
				{Index: 5, ItemID: 1201, Type: 5, Identified: false, Equip: true},
				{Index: 7, ItemID: 1202, Type: 5, Identified: true, Equip: true},
			},
		},
	}
	window := IdentifyWindow{}
	window.OpenList(Context{Session: sessionState, ScreenW: 800, ScreenH: 600}, network.ItemIdentifyList{Indexes: []uint16{3, 5, 7, 9}})

	items := window.items(sessionState)
	if len(items) != 1 || items[0].Index != 5 {
		t.Fatalf("identify items = %+v, want only index 5", items)
	}
	if !window.IsOpen() {
		t.Fatal("identify window did not open")
	}
}
