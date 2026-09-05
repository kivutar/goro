# classic-ro-client Gap Checklist

This checklist records features and smaller behaviors present in
`~/src/classic-ro-client` that are absent or incomplete in Goro.

The reference client's `Features.md` says it was validated with rAthena
packet version `20120307`, while Goro targets `20080910`. Items in the main
checklist are applicable to the 2008 client or are packet-independent client
features. Anything that may depend on later client behavior is kept in a
separate validation section and must not be implemented blindly.

## Gameplay and protocol compatibility

### Stealth

- [ ] Treat Hide, Cloaking, Invisible, and Chase Walk consistently when deciding how an actor is rendered.
- [ ] Render the local player with the correct translucent/hidden-viewer appearance for every supported stealth state, not only `SC_HIDING`.
- [ ] Apply the appropriate movement and input restrictions while the local player is hidden.
- [ ] Exclude actors that should not be targetable or pickable while hidden.
- [ ] Add focused tests for the local player, an allowed hidden viewer, an ordinary remote player, PvP, and WoE.

Current limitation: `localActorHidden` only checks `db.StatusHiding`, although
the siege-emblem code already recognizes Hide, Cloak, Invisible, and Chase
Walk as hidden states.

### Quest journal and markers

- [ ] Parse the 2008 quest list packets (`0x02B1` through `0x02B5` and relevant updates).
- [ ] Store active quest state, descriptions, objectives, and hunt progress in the session.
- [ ] Implement the quest list and quest detail UI.
- [ ] Handle quest activation/state acknowledgement (`0x02B6`).
- [ ] Handle quest removal and state updates (`0x02B7` where applicable).
- [ ] Display quest NPC markers in the world.
- [ ] Display quest markers and objective dots on the minimap.
- [ ] Preserve the existing quest EXP console notifications.
- [ ] Add packet, session-state, UI, and marker regression tests.

See [the packet audit](packet-coverage-20080910.md), especially the currently
untracked `0x02B1`-`0x02B7` family.

### Legacy mail

- [ ] Implement mailbox refresh and inbox listing.
- [ ] Implement reading mail.
- [ ] Implement composing and sending mail.
- [ ] Implement item and Zeny attachments.
- [ ] Implement taking attachments.
- [ ] Implement deleting and returning mail.
- [ ] Implement mailbox and read-mail windows using Goro's existing window components.
- [ ] Handle the legacy 2008 mail packets `0x023F`, `0x0241`, `0x0243`, `0x0244`, `0x0246`, `0x0247`, `0x0248`, and `0x0273`.
- [ ] Test empty mailboxes, full mailboxes, attachments, server errors, and reconnects.

This means the legacy mail system, not a newer RODEX-only implementation. See
[the per-feature packet audit](packet-coverage-20080910-per-feature.md).

### Skill-specific client flows

- [x] Implement the Sage Auto Spell choice list and send selection packet `0x01CE`.
- [x] Turn the Star Gladiator place/Feel request into the original confirmation flow instead of only logging it, then send `0x0254`.
- [x] Implement the original `/doridori` client behavior and packet `0x01E7`.
- [ ] Verify whether the 2008 Novice Explosion Spirits request (`0x01ED`) needs a distinct client action, then implement it if applicable.
- [x] Implement Token of Siegfried self-revival through the 2008 auto-revive request (`0x0292`).
- [ ] Add end-to-end tests ensuring these skills cannot silently stall while waiting for a client response.

### Server-driven progress and information

- [x] Parse and render NPC progress-bar start packet `0x02F0`.
- [x] Send progress-bar completion/cancel acknowledgement `0x02F1`.
- [x] Handle server cancellation (`0x02F2`, if applicable to the 2008 profile).
- [x] Ensure the progress display swallows input where the original client did.
- [x] Implement server `ShowDigit` countdown displays.
- [x] Implement boss information, map marker, death, and respawn-time updates.
- [x] Implement remaining skill-message feedback that currently has no dedicated presentation.

### Storage password

- [x] Parse the storage-password prompt and result states.
- [x] Implement setting a new storage password.
- [x] Implement entering an existing storage password.
- [x] Handle confirmation mismatch, failure counts, and lockout/penalty state.
- [x] Ensure storage does not open before successful authentication when the server requires a password.
- [x] Implement the missing `0x023B` request flow and verify its 2008 wire layout against rAthena's packet database.
- [ ] Run the flow end to end against a server implementation. Current rAthena accepts the packet layout but leaves `clif_parse_StoragePassword` as an upstream TODO.

### Rankings and fame feedback

- [x] Implement `/blacksmith` top-ten ranking request and response (`0x0217`).
- [x] Implement `/alchemist` top-ten ranking request and response (`0x0218`).
- [x] Format ranking entries consistently in the console.
- [x] Handle Blacksmith and Alchemist fame-point gain notifications.
- [x] Preserve the existing TaeKwon mission and ranking implementation.
- [ ] Investigate the `0x0237` killer ranking separately before treating it as a 2008 parity requirement.

## Client UI and quality-of-life

### Window state persistence

- [ ] Assign a stable persistence key to every persistable game window.
- [ ] Save and restore window positions between application sessions.
- [ ] Save and restore open/closed state where appropriate.
- [ ] Add original-style window collapse/minimize behavior.
- [ ] Save and restore collapsed state.
- [ ] Save the shortcut bar's visible row count and keyboard mode.
- [ ] Clamp restored windows to the current screen after a resolution or scale change.
- [ ] Do not persist transient dialogs, context menus, tooltips, or server-owned modal state.
- [ ] Test logout, character change, application restart, resolution change, and fractional scaling.

Goro currently persists the native window size and fullscreen state, but not
individual game-window layout.

### Keyboard shortcuts

- [ ] Add a shortcut configuration window instead of relying only on the hardcoded F1-F9, 1-9, and Q-O mapping.
- [ ] Persist physical-key bindings so layouts such as AZERTY remain correct.
- [ ] Add the original Alt+M shortcut-list window for chat-command bindings.
- [ ] Support editing and clearing all Alt+1 through Alt+0 command slots.
- [ ] Verify original Battle Mode behavior for the 2008 client and expose it cleanly if it differs from Goro's always-available extra rows.
- [ ] Keep server-side item/skill hotkey slots distinct from client-side physical key bindings.

### Missing-map recovery

- [ ] Preserve and surface the missing/unreadable GAT error instead of silently returning from map initialization.
- [ ] Show a fallback UI that does not depend on the missing GRF assets.
- [ ] Display the missing map name and a useful explanation.
- [ ] Offer return to character selection.
- [ ] Optionally offer a recovery warp to Prontera when the connected server permits it.
- [ ] Avoid leaving the player connected on a black, unusable map.
- [ ] Test the known `new_1-1`-missing scenario.

This is a safety improvement from classic-ro-client rather than strict
original-client behavior, but it directly addresses a failure already seen in
Goro.

### Inventory and item presentation

- [x] Show a quantity prompt when dropping a stackable item with an amount greater than one.
- [x] Default the prompt safely and clamp the result to the available amount and packet range.
- [x] Keep the direct one-item drop path for non-stackable items.
- [x] Implement the GRF-backed book reader for readable items.
- [x] Support page navigation, book titles, wrapping, and malformed/missing book data.
- [x] Add the original floor-item shadow beneath dropped item sprites.

### Level-up availability notifications

- [x] Show the small Base-level/stat notification when unspent stat points become available.
- [x] Show the Job-level/skill notification when unspent skill points become available.
- [x] Make each notification open the corresponding window or section.
- [x] Dismiss the notification after activation without sharing the click with the map or another window.
- [x] Keep these notifications distinct from the existing level-up world effect and sound.

### Monster information / Sense

- [x] Parse the original Monster Info/Sense response (`0x018C`).
- [x] Implement a Monster Info window.
- [x] Display monster name/class, level, HP, DEF, MDEF, race, size, property, and elemental resistances.
- [x] Reuse the monster life information Goro already caches for Sense.
- [x] Do not turn this into permanent 2012-style monster HP bars.

### NPC cut-in illustrations

- [x] Parse NPC cut-in packet `0x01B3`.
- [x] Load the requested illustration from the GRF.
- [x] Support the original left, center, right, windowed, and windowless positions.
- [x] Support clearing/replacing an existing cut-in.
- [x] Clear cut-ins on dialog close, map transition, character change, and disconnect.
- [x] Ensure cut-ins and NPC dialogs compose correctly without leaking map clicks.

### Minimap details

- [ ] Add quest and guide-direction markers as part of the quest implementation.
- [x] Party-member minimap markers already exist.
- [x] Same-map guild-member markers already exist.
- [x] Server compass markers already exist.

Party-name hover belongs to the 2007 world-map feature, not to the small HUD
minimap. The latter only needs the existing coloured party markers for 2008
parity.

### Graphics options and small rendering details

- [ ] Expose runtime fog, effect, and aura toggles where they are meaningful for the 2008 renderer.
- [ ] Implement Hallucination's full-screen ripple/distortion effect.
- [ ] Ensure screen distortion affects the finished scene as intended without corrupting UI input or causing a persistent render target allocation.
- [ ] Consider smoothly fading the existing Taekwon Soul Link/Eske night tint instead of switching it abruptly.
- [ ] Verify the lighting behavior against the 2008 client before describing it as a general day/night cycle.

## Long-tail effect coverage

- [ ] Produce a machine-readable list of effect IDs used by the 2008 data/server profile that have no `db.EffectSpecs` entry.
- [ ] Separate truly used 2008 effects from later numeric constants.
- [ ] Compare each missing effect with robr and classic-ro-client before implementing it.
- [ ] Prefer generic SPR/ACT and STR players where the original data fully describes the effect.
- [ ] Add custom behavior only when the reference client requires it.
- [ ] Add visual or structural regression tests for each imported effect family.
- [ ] Keep unsupported effects explicit; do not guess visual behavior.

classic-ro-client claims handling for every effect ID from 1 through 1050 by
combining generic players and custom effect families. Goro intentionally leaves
an effect unsupported when its reference behavior has not been established.

## Version-sensitive items requiring validation

These are intentionally unchecked, but they are not yet approved implementation
work. First establish that the feature belongs to the 2008 client and that the
OldRO data contains the required assets/tables.

- [ ] Validate whether the world map UI and its map-position tables belong in the selected 2008 client profile.
- [ ] If valid, implement the world map, current-map/player indicator, party markers with names on hover, and per-map minimap inset.
- [ ] Validate skill/global cooldown packets for `20080910` before adding cooldown gating or shortcut overlays.
- [ ] Validate whether the status-icon clock-wedge display is appropriate for 2008; Goro already has tooltips and a duration bar.
- [ ] Validate party-booking packets `0x0802` and `0x0806`; they appear newer and should not be pulled into the 2008 backlog by default.
- [ ] Validate killer ranking `0x0237` independently.
- [ ] Keep later cash-shop, memorial-dungeon, auction, and newer RODEX behavior outside the 2008 parity scope unless explicitly requested.
- [ ] Do not treat classic-ro-client's JSON companion AI configuration window as a direct parity requirement; Goro intentionally supports Gravity-style `AI.lua`/`AI_M.lua` instead.

## Verified existing Goro behavior

These reference-client features were checked and should not be re-added as
missing work.

- [x] Party-member minimap markers.
- [x] Same-map guild-member minimap markers.
- [x] Server compass markers.
- [x] Status-icon tooltips and remaining-duration presentation.
- [x] Item information illustrations, full card illustrations, and card-slot/card composition support.
- [x] Talkie Box and Graffiti text prompts and outbound talkbox skill packet.
- [x] Positional actor, effect, and RSW audio.
- [x] Party creation, invitation by name, settings, and member actions.
- [x] Item-pickup notification.
- [x] TaeKwon mission and ranking support.
- [x] Cast bars, projectiles, camera shake, MVP effect, and level-up effects.
- [x] Loading transition with an initial black cover and fade.

## Reference locations

- `~/src/classic-ro-client/Features.md`
- `~/src/classic-ro-client/docs/TODO.md`
- `~/src/classic-ro-client/lib/ui-component/src/game/`
- [Goro 2008 feature packet audit](packet-coverage-20080910-per-feature.md)
- [Goro detailed 2008 packet audit](packet-coverage-20080910.md)
- [Goro database import backlog](db-import-todo.md)
