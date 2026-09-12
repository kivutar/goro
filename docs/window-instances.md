# Window instances

Goro's overlay manager identifies individual widgets, not window types. The
single-instance restriction came from `worldUI` owning one item description,
one card illustration, and one whisper conversation and sharing those objects
between all callers.

## Reference audit

Checked the local clones on 2026-09-13. These findings describe the checked
implementations; they do not establish unlimited-window behavior in the
original 2008 executable.

| Window | Evidence in the cloned clients | Goro behavior |
| --- | --- | --- |
| Item descriptions, including slotted cards | classic-ro-client keeps a primary description and a separate slotted-card description in `lib/ui-component/src/game/item_info_window.rs`. roBrowser's `ItemInfo` and open-midgard's `WID_ITEMINFOWND` are singletons. | Independent descriptions, as requested, including different copies of the same item. Opening a slotted card preserves its equipment description. |
| Card artwork | All three checked clones hold one artwork window: classic-ro-client's `card_illustration`, roBrowser's `CardIllustration`, open-midgard's `WID_ITEMCOLLECTIONWND`. | Independent artwork windows, as requested. This extends the checked references. |
| Whisper conversations | roBrowser's `WhisperBox.instances` is keyed by nickname. Its `PrivateMessage.js` enables automatic popups from `20090617`; classic-ro-client uses its main chat window. | Repair the already-existing Goro popup feature: one conversation per recipient, with independent history and drafts. This is not claimed as strict 2008 parity. |
| Books | classic-ro-client's `Windows.book_window` and roBrowser's `MakeReadBook` are singletons. | Keep one reader. Reading closes only the originating item description. |
| Mail reader/composer | classic-ro-client's `Windows.read_mail_window` and roBrowser's `ReadMail` are singletons. | Keep the existing mail flow and its attachment/transaction state. |
| Equipment inspection | roBrowser's `Item.js` updates `PlayerViewEquip.getUI()` with the latest response. | Keep one inspected equipment window; its item descriptions can be opened independently. |
| Monster information / Sense | classic-ro-client's `Windows.monster_info_window`, roBrowser's `Sense`, and the later dhxj snapshot's `UI/UIMonsterInfo.cpp` hold one window. | Keep one Sense result window. |
| Shop, storage, vending, trade, NPC dialogs | The references tie these windows to their current server interaction. | Keep each existing server interaction; do not duplicate a transaction by cloning its window. |
| Confirmation/input dialogs | Goro already has multiple instances of the same dialog types for different purposes. | Keep their modal and server-specific lifecycle. |

Reference entry points:

- `~/src/classic-ro-client/client/src/ui/windows.rs`
- `~/src/classic-ro-client/lib/ui-component/src/game/item_info_window.rs`
- `~/src/robr/src/UI/Components/WhisperBox/WhisperBox.js`
- `~/src/robr/src/Engine/MapEngine/PrivateMessage.js`
- `~/src/robr/src/UI/Components/{ItemInfo,CardIllustration,MakeReadBook,Sense}/`
- `~/src/robr/src/UI/Components/Mail/ReadMail.js`
- `~/src/robr/src/Engine/MapEngine/Item.js`
- `~/src/open-midgard/src/ui/UIWindowMgr.cpp`

The dhxj snapshot's original `UIWindowMgr.cpp` is mostly absent and its later
CEGUI replacements contain empty window cases. It is not sufficient evidence
for original-client instance limits.

## Ownership and input

`ItemWindows` owns pointers to item descriptions and artwork plus the existing
single book reader. Closing a description or artwork removes only that overlay;
the collection drops closed instances. Each description holds an item snapshot.
The inventory, equipment, cart, storage, trade, shop, vending, mail, and equipment
inspection paths all open descriptions through this collection.

`WhisperWindows` owns one conversation per case-insensitive recipient. Closing
a conversation preserves its bounded history and draft for the current session.
Incoming messages update the appropriate conversation without taking keyboard
focus. Opening a recipient explicitly raises and focuses that conversation.

Both collections survive map transitions and rebind callbacks without replacing
their overlays or changing their relative stacking order. A fresh world mode
starts with empty collections; the existing login transition clears overlays.

The existing `Window` and overlay manager handle placement, dragging, focus,
and Escape by instance. Polling consults the same overlay stack as pointer
dispatch; HUD windows do not consume window-close Escape actions. Modal dialogs
retain their existing cancellation rules. New coincident windows are offset and
clamped to the screen.

The 2008 whisper acknowledgement contains only a result byte. rAthena's
`src/map/clif.cpp:clif_parse_WisMessage` answers local recipients immediately,
but forwards remote recipients through `intif_wis_message`; replies come back
through `src/map/intif.cpp:intif_parse_WisEnd`. NPC and channel whispers can
return without an acknowledgement. A recipient FIFO would therefore be
incorrect. Server feedback stays in the main chat without guessing a recipient;
local send failures appear in their originating conversation too. No network
queue or packet changes are required.

## Verification

Automated tests exercise independent item snapshots, slotted-card navigation
and artwork buttons through UI events, individual close/rebind behavior,
book reader reuse, overlapping-window dragging and Escape, HUD/menu behavior,
trade/vending cancellation packets, mail/cart cleanup, on-screen placement,
whisper focus/drafts/recipient routing, popup preferences, map transitions, and
acknowledgements interleaved with console sends.

Validation commands:

```sh
CGO_ENABLED=0 go test -tags nofakecgo ./...
CGO_ENABLED=0 staticcheck -tags nofakecgo ./...
```

Live checks still needed:

- Open several descriptions from inventory, equipment, storage, and a shop.
  Compare two copies of an item, inspect their cards, and close them separately.
- Drag overlapping windows; press Escape with the pointer over a different
  window. Confirm that the visible top window closes and the map receives no click.
- Chat with two recipients while receiving a message from a third. Check draft
  preservation, recipient selection, popup preferences, and error feedback.
- Warp with descriptions, artwork, and conversations open, then log out and
  select another character. Check that map transitions preserve instances and
  a new session starts cleanly.
