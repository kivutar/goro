# gogpu/ui Migration Todo

Goal: migrate the remaining RO windows and dialogs to the gogpu/ui tree style:

- one readable widget tree per window/dialog when possible
- common `Window(...)` wrapper for title bar, footer, close button, dragging, opacity, and sizing
- gogpu/ui input callbacks instead of manual `pointInRect` hitboxes
- gogpu/ui cursor state instead of per-window `CursorAction` bridges
- a single global UI overlay layer for all in-game windows, so multiple windows can stay open and overlap
- delete legacy CPU-drawn helpers as each window is migrated

## Done

- [x] Login window
- [x] Settings window
- [x] Escape menu
- [x] Death modal
- [x] Character selection window
- [x] Character creation window
- [x] Equipment window

## High Priority

- [x] NPC dialog and choice dialog
  - Important because it is a core gameplay dialog and still has custom dragging, scrolling, button hitboxes, and text rendering.
  - Keep color-code parsing, choice scrolling, and the subdialog layout.

- [ ] Inventory bag window
  - Important because it has tabs, item grid, hover, double-click use/equip, drag to shortcut/storage/drop, and item info.
  - This should become the reference implementation for grid-based item widgets.

- [ ] Shop windows
  - Includes buy/sell selection, buy list, sell cart, and sell inventory.
  - Good migration target after inventory because it can reuse item row/grid pieces.

- [ ] Storage window
  - Similar enough to inventory/shop lists that it should reuse the same item-list primitives.

## Medium Priority

- [ ] Stats window
  - Mostly simple rows and plus buttons, but currently sends stat-increase packets from legacy input handling.
  - Good target for standardized row layout, footer buttons, and icon-sized plus buttons.

- [ ] Skills window
  - Needs skill rows, icons, pending level changes, confirm/cancel footer, and skill tooltip behavior.
  - Should reuse gogpu/ui list/table primitives if they are practical.

- [ ] Shortcut bar
  - Not a normal window, but it is UI-heavy and still has custom hitboxes.
  - Needs drag/drop, function-key activation, skill level display, item counts, and target cursor integration.

- [ ] Item info window
  - Should be a compact `Window(...)` with two-column layout: illustration on the left, item data on the right.

- [ ] Identify window
  - Simple scrollable list of unidentified equipment.
  - Good small migration after item-list primitives exist.

- [ ] Teleport / Warp destination modal
  - Small modal, but it should share the same footer/button system and scrollable choice list.

## Lower Priority

- [ ] Basic character window
  - Currently drawn as HUD, not a floating gogpu/ui window.
  - Migrate only if we decide HUD elements should also be gogpu/ui trees.

- [ ] Basic menu button grid
  - Small HUD control, but it should eventually use shared button styling and callbacks.

- [ ] Console
  - Keep its dark translucent style.
  - Migration is lower priority because it already behaves differently from RO windows and has specialized text history/input behavior.

- [ ] Minimap and status/buff icons
  - HUD elements rather than dialogs.
  - Migrate later only if gogpu/ui image/grid widgets are cheap enough for always-visible HUD.

## Cleanup After Each Migration

- [ ] Remove old `Update(ctx)` mouse handling for the migrated window.
- [ ] Remove the migrated window's `CursorAction` and its call from `game/cursor.go`.
- [ ] Do not re-add per-window cursor bridges for gogpu/ui windows. If blank UI areas need to block world cursors, solve it once in `WindowState`/the UI manager.
- [ ] Do not add custom `Root`/`Event` types inside migrated dialogs. Use the normal window tree, and use `WindowState.Publish(ctx)` / `UIManager.AddOverlay` for floating windows.
- [ ] Do not call `UIManager.SetRoot` or `UIManager.Clear` from in-game windows. Closing one window must unpublish only that window.
- [ ] Do not clear and re-add the same gogpu/ui widget every frame. gogpu/ui hover/cursor state depends on stable widget identity; publish once, then only republish when the visible widget actually changes.
- [ ] Remove stale `Bounds()` and `*Rect` helpers used only by old hitboxes.
- [ ] Do not keep per-window point-inside helpers when `WindowState.Update(ctx)` already handles inside-window consumption.
- [ ] Remove per-window clamp helpers when shared helpers like `clampWindowInt` already cover the same behavior.
- [ ] Remove tests that only preserve deleted hitbox helpers.
- [ ] Replace manual title/footer/button drawing with `Window(...)` and rotheme widgets.
- [ ] Keep network/game actions in callbacks or game-side methods, not inside pure layout code.
- [ ] Run:
  - `GOCACHE=/tmp/goro-go-build CGO_ENABLED=0 go test -tags nofakecgo ./...`
  - `XDG_CACHE_HOME=/tmp/goro-cache GOCACHE=/tmp/goro-go-build CGO_ENABLED=0 staticcheck -tags nofakecgo ./...`
