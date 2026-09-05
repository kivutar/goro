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
- [x] Exit confirmation modal
- [x] Death modal
- [x] Character selection window
- [x] Character creation window
- [x] Equipment window
- [x] Stats window
- [x] Inventory bag window
  - Uses shared vertical tabs to categorize carried items.
- [x] Shop windows
  - Includes transaction choice, buy list/cart, sell inventory/cart, native datatables, and item-info right click.
- [x] Storage window
  - Uses the same vertical-tab presentation to filter storage categories.
- [x] Item info window
- [x] Identify window
- [x] Skills window
  - Includes class-level vertical tabs, staged skill increases, footer buttons,
    native scrolling, and skill info tooltips.
- [x] Teleport / Warp destination modal
- [x] Shortcut bar
  - Published as a gogpu/ui HUD overlay with native mouse events.
  - Keeps coordinate acceptance helpers for cross-window item/skill drops until drag ghosts move to a shared gogpu/ui drag overlay.
- [x] Basic character window
- [x] Basic menu button grid
- [x] Console
  - Published as a dark translucent gogpu/ui overlay.
  - Uses native gogpu/ui textfield and scrollview widgets.
  - Keeps only game-side chat command dispatch and Up/Down history handling.
- [x] Minimap
  - Published as a gogpu/ui overlay with a common RO title window.
  - Uses a small custom canvas widget for the map bitmap and live actor markers.
- [x] NPC dialog and choice dialog
  - Keep color-code parsing, choice scrolling, and the subdialog layout.
- [x] Status/buff icons
  - HUD elements rather than dialogs.
  - Published as a small gogpu/ui overlay under the minimap.
  - Uses roBrowser-sourced status icon metadata from `db/status_icons.go`.
- [x] Trade window
- [x] Vending setup and buyer shop windows
- [x] Card composition window
- [x] Card illustration window
- [x] Friend/party window
  - Friends tab, party tab, party settings, invitations, and party HP rows.
- [x] Show-equipment window
- [x] Unified overlay text widgets
  - FPS meter, character names, speech bubbles, and item/skill tooltips share
    the same overlay text rendering path and console-style background.
- [x] Shared window interaction behavior
  - Focusing an overlapping window raises it to the front.
  - Dragged windows stay visible after release, and tooltips are suppressed
    during a drag.
  - The character button grid follows the character information window as one
    drag group.

## Completed Cross-Cutting Work

- [x] Shared drag ghost overlay
  - Inventory, storage, shop, and skill drag ghosts render in the top-level game overlay, above gogpu/ui windows and below the RO cursor.
- [x] Final UI sharpness pass
  - Text and images are snapped to physical pixels, including at fractional
    display scaling.
  - Vertical tab text uses native rotated glyph rendering rather than a rotated
    raster image.

## Cleanup After Each Migration

- [ ] Remove old `Update(ctx)` mouse handling for the migrated window.
- [ ] Remove old `Draw(...)` methods and render-loop calls when the migrated window is fully published through gogpu/ui overlays.
- [ ] Remove the migrated window's `CursorAction` and its call from `game/cursor.go`.
- [ ] Remove legacy transient status fields/helpers when the migrated gogpu/ui tree does not display them.
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
