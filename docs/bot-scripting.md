# Bot Scripting

Goro can run a Lua script during login and while the player is in-game. This is intended for
local experimentation and simple automation.

Run a script with:

```sh
./goro --data-dir ~/OldRO --script scripts/loot-and-attack.lua
```

The keyboard/gamepad controls script is also bundled in every binary:

```sh
./goro --data-dir ~/OldRO --script builtin:wasd
```

Use `--script scripts/wasd.lua` to load an editable copy, or `--script none` to
disable scripting. The same values work as `path` under `[script]` in `goro.ini`.

In game, use `/script wasd` in chat to select the bundled controls, `/script none`
to disable scripting, or `/script` to list bundled scripts. `wasd` is currently
the only bundled script. The selection replaces any configured script for the
current run, survives map changes, and does not modify `goro.ini`.

Callbacks are optional. Goro calls `tick()` roughly every 150 ms only while the
world mode is active. `gamepad(dt)` and `input()` also run during login, with
gameplay input unavailable there. The script is reloaded when changing modes
(including map changes), clearing its local targets and other transient state.
Held controller buttons are ignored until released when entering another mode.

```lua
function tick()
	-- bot logic here
end
```

## Headless mode

Run the same scripts without a window or audio:

```sh
./goro --headless --data-dir ~/OldRO \
  --username tester --password secret --char-slot 0 \
  --script scripts/loot-and-attack.lua
```

`--headless` enables automatic login and requires credentials and a character
slot (0–8). These can also come from the existing `[login]` configuration.
As with `--autologin`, the first login server and first character server are
selected by default. Use `--server-slot N` to select a login server from
`clientinfo.xml`, and `--char-server-slot N` to select a character server from
the list returned after login. Both count from 0; an unavailable slot stops
autologin with an error. The corresponding `[login]` settings are `server_slot`
and `char_server_slot`. The script is optional; without one the client stays
connected.

Headless mode updates at 60 Hz without drawing or loading scene assets. It
keeps the collision grid, game data, network updates, and Lua scripts. Combat
uses server timings and existing fallback durations when no sprite is loaded.
Stop the process with Ctrl+C.

There is no automatic reconnect. Scripts can answer NPC dialogs with `goro.npc_dialog`.
`--no-ui` only hides the graphical client's UI.

## API

All functions are exposed through the global `goro` table.

Scripts may also define an optional global `input()` function. Goro calls it
once per frame so keyboard edges can be handled without waiting for the slower
bot tick.

An optional `keypress(code)` callback runs on a fresh physical key press,
before default UI and shortcut handling, when gameplay keyboard input is
available. Focused chat, forms and modals take priority. Use
`goro.keyboard.consume_press(code)` inside this callback to claim a key;
its associated text and repeats will not reach the UI until it is released.
Unconsumed keys retain their normal behavior, even when a script is loaded.

```lua
function keypress(code)
	if code == "Space" then
		goro.keyboard.consume_press(code)
	end
end

function input()
	-- Poll goro.keyboard.is_down("Space") here for continuous behavior.
end
```

### `goro.keyboard`

The keyboard API uses layout-independent physical key names such as `"KeyW"`,
`"Tab"`, and `"ShiftLeft"`. Letter codes describe physical key positions, not
the glyph printed by the current layout. For example, the physical WASD
positions are ZQSD on an AZERTY keyboard.

- `available()` reports whether keyboard input is available to the script. It is `false` while a UI control has keyboard focus.
- `is_down(code)` reports held state.
- `was_pressed(code)` and `was_released(code)` inspect edges without consuming them.
- `consume_press(code)` consumes a press edge and returns whether one was available. Held state is unchanged. Use it in `keypress(code)` to intercept UI input; `input()` runs after UI event dispatch.
- `text()` returns the frame's raw layout-translated text, including consumed keys, so scripts can implement their own text input. It is empty while UI owns the keyboard.

The keyboard API only reports input. Movement, combat, prompts, and other
behavior remain Lua policy built from the generic functions below.

### `goro.gamepad`

The same interface works on Windows, Linux, macOS and Android. Goro selects the
first detected controller and keeps it selected until it disconnects. Input
snapshots become visible to Lua once per graphical frame; use `input()` for
press/release edges. Linux and Windows device discovery and polling run in a
background worker so driver calls cannot block game updates. Queries read shared
frame state without consuming it; repeated queries during a frame return the same
edges. Only the window loop drains the device event queues, once per update.

- `connected()` reports whether a controller is connected, regardless of UI focus.
- `name()` returns its name, or an empty string when disconnected.
- `available()` reports whether gameplay input is allowed and a controller is connected.
- `is_down(button)`, `was_pressed(button)`, `was_released(button)` report held state and frame edges.
- `axis(name)` returns a normalized axis value. Sticks range from -1 to 1 (negative is left/up); triggers range from 0 to 1.

An optional `gamepad(dt)` callback runs before controller pointer dispatch in
graphical mode. `dt` is elapsed seconds, capped at 0.05. Within this callback,
button and axis queries also work while a dialog has focus; check `available()`
before performing gameplay actions. Call `consume(button)` or `consume_pointer()`
to claim controls for this frame. A claimed mouse button remains suppressed until
release, so releasing a modifier cannot turn a held skill button into a click.
The ordinary `input()` callback retains its gameplay focus filtering.

UI navigation is available in login and world mode:

- `goro.in_game()` reports whether the script is bound to the world. Gameplay
  actions return false/no result during login; world lists are empty and vitals
  are zero.
- `goro.ui.active()` reports whether login or an available NPC dialog owns
  directional navigation.
- `goro.ui.control(action)` accepts `"up"`, `"down"`, `"left"`, `"right"`,
  `"confirm"`, or `"cancel"`. Login routes these to the active screen or modal;
  in game they operate the NPC dialog using its normal rules.

Login confirmation never bypasses a modal or a screen transition. Scripts should
consume their UI button bindings before returning to the pointer fallback.

The following actions support scripted controller bindings:

- `goro.use_shortcut(slot)` activates slot 1–9 of the active hotbar row and returns `used, skill_id`: a success flag and the activated skill ID (zero for items).
- `goro.rotate_camera(yaw, pitch)` adds angles in degrees, respecting map camera locks.
- `goro.zoom_camera(delta)` adjusts camera distance, respecting map zoom locks and limits; positive zooms out.
- `goro.camera_yaw()` returns the current map camera yaw in degrees.
- `goro.cancel_skill()` cancels skill targeting.
- `goro.npc_dialog()` reports whether an NPC dialog is available; pass `"up"`, `"down"`, `"confirm"`, or `"cancel"` to operate it. Cancellation follows the normal NPC dialog rules.
- `goro.pointer_over_ui()` reports whether the pointer is over a UI overlay.

Button names are positional: `south`, `east`, `west`, `north`, `left_shoulder`,
`right_shoulder`, `back`, `start`, `left_stick`, `right_stick`, `dpad_up`,
`dpad_down`, `dpad_left`, `dpad_right`. Axis names are `left_x`, `left_y`,
`right_x`, `right_y`, `left_trigger`, `right_trigger`. Unknown names return
false or zero. The API leaves deadzones to the script; `wasd.lua` uses 0.3.

Outside `gamepad(dt)`, gameplay queries return neutral input while chat/forms
own the keyboard or the player cannot act. Disconnecting clears axes and held
buttons and reports release edges; losing window focus clears input without
generating press or release actions. Headless mode does not poll physical controllers.

`wasd.lua` moves relative to the camera with the left stick/D-pad, uses West for
loot, L2 + right stick to rotate/tilt the camera, R2 + right stick up/down to zoom,
and R2 + South/East/West/North for hotbar slots 1–4. R2 + D-pad Up/Right/Down/Left
uses slots 5–8; those directions resume movement only after release. Shoulders
cycle enemies or eligible skill targets; South attacks or confirms and East
cancels. NPC dialogs use D-pad up/down and South/East. During login, D-pad up/down
selects servers or credential fields, left/right selects character slots, South
confirms, and East goes back. Moving the right stick switches to pointer clicks;
using the D-pad resumes menu navigation. Character creation keeps pointer
controls for its appearance and stats, with Start/Escape to go back. Text entry
uses the normal keyboard (Select opens it on Android).
Unclaimed right-stick movement, South/East mouse clicks and Start/Escape remain
shared client menu controls and work even without a script.

### `goro.player()`

Returns the local player state.

Fields:

- `id`
- `x`
- `y`
- `hp`
- `max_hp`
- `sp`
- `max_sp`
- `dead`
- `moving` (whether the server-confirmed walk is still in progress)
- `walk_sequence` (a counter incremented whenever the client sends a walk request)

A script can save `walk_sequence` after `goro.walk()` succeeds. If it changes,
another movement request, such as a pointer action or skill chase, has replaced
that walk. `moving` can remain false while a request awaits the server's reply.

### `goro.hp()`

Returns two values:

```lua
local hp, max_hp = goro.hp()
```

### `goro.sp()`

Returns two values:

```lua
local sp, max_sp = goro.sp()
```

### `goro.walk(x, y)`

Requests a walk to the map cell at `x`, `y`. It returns `true` when the movement
cooldown is ready, the target is in bounds, any available local walkability
data accepts it, and the request was sent. Otherwise it returns `false`.

This uses the normal client movement path and cancels an active attack intent,
just like manual movement. Scripts should wait for player position updates
instead of submitting a new destination on every frame.

### `goro.stop()`

Requests a controlled stop at the end of the current server-approved path
segment. It returns `true` when the player is already stopped or the request
was sent, otherwise `false`.

### `goro.enemies()`

Returns an array of currently attackable enemies. Actors already playing their death animation are filtered out.

Each enemy has:

- `id`
- `name`
- `x`
- `y`
- `job`
- `object_type`
- `distance`

### `goro.players()`

Returns an array of visible nearby player characters, excluding the local character.

Each player has:

- `id`
- `name`
- `x`
- `y`
- `job`
- `distance`
- `party_member`
- `hp`
- `max_hp`
- `dead`

HP and death information is available for party members when the server has provided it. For other players, `hp` and `max_hp` are `0`.
`name` can be empty until the client has received that actor's name; use `id` as the stable identity.

### `goro.companions()`

Returns an array of visible homunculi and mercenaries.

Each companion has:

- `id`
- `name`
- `kind` (`"homunculus"` or `"mercenary"`)
- `own`
- `x`
- `y`
- `job`
- `distance`
- `hp`
- `max_hp`
- `sp`
- `max_sp`
- `dead`

Vitals are available for the local player's companions and for other companions when the server has provided an actor HP update. Unknown values are `0`.

### `goro.attack(id)`

Requests a normal attack on the enemy actor with this id.

Returns `true` if the target exists and is attackable, otherwise `false`.

This uses the same path as a normal player click, including chase and range handling. Scripts should avoid calling it every tick for the same target; keep a small retry delay.

### `goro.target(id)`

Alias for `goro.attack(id)`.

### `goro.skill(id, skill[, level])`

Requests a skill on the actor with this id. `skill` can be either a numeric skill id or a learned skill name such as `"AC_DOUBLE"` or `"AL_HEAL"`. Self-targeted skills use `goro.player().id`, for example `goro.skill(goro.player().id, "AL_ANGELUS")`. Ground-targeted skills are not supported by this function.

Returns `true` if the actor is a valid target for the learned skill, otherwise `false`. Enemy skills remain limited to enemies, while friendly skills can target nearby players, homunculi, and mercenaries.

The optional `level` selects a level between `1` and the learned level for skills that support level selection. When omitted, the learned level is used.

This uses the same path as a skill-window or shortcut target click, including chase and range handling. Scripts should avoid calling it every tick for the same target; keep a small retry delay.

```lua
for _, player in ipairs(goro.players()) do
	if player.party_member and player.max_hp > 0 and player.hp / player.max_hp < 0.5 then
		goro.skill(player.id, "AL_HEAL")
		break
	end
end
```

Friendly skills can target companions in the same way:

```lua
for _, companion in ipairs(goro.companions()) do
	if companion.own and companion.kind == "homunculus" then
		goro.skill(companion.id, "AM_POTIONPITCHER", 3)
	end
end
```

### `goro.pending_skill()`

Returns the skill currently waiting for a target, or `nil` when no skill is armed or a chosen target is already being chased.

Fields:

- `id`
- `name`
- `level`
- `max_level`
- `type` (the server target flags)
- `range`
- `target` (`"actor"`, `"ground"`, or `"self"`)
- `caster_id`
- `caster_kind` (`"player"`, `"homunculus"`, or `"mercenary"`)
- `caster_x`
- `caster_y`

The caster fields are omitted when the caster is not currently available.

### `goro.skill_targets(ignore_shift = false)`

Returns the living actors eligible for the currently armed actor-target skill,
including yourself when allowed. Each entry has `id`, `name`, `x`, `y`, `job`,
and `distance`; the array is unordered. It is empty when no actor-target skill
is waiting for selection.

Eligibility follows pointer targeting, including Shift, `/noshift` and PvP
rules. With `/noshift` enabled, Heal can select monsters, including undead.
The WASD script uses this list for both Tab and controller shoulder cycling.
Pass `true` to ignore held Shift while still respecting `/noshift` and PvP.
WASD does this on both devices so Tab/Shift+Tab and the shoulders cycle the
same targets; `/noshift` controls whether friendly skills can select enemies.

### `goro.use_pending_skill(id)`

Submits an actor as the target of the skill returned by `goro.pending_skill()`. It returns `true` when the target is valid and the use or chase was started, otherwise `false`.

Unlike `goro.skill()`, this uses the exact armed skill and selected level, including homunculus and mercenary skills.

### `goro.highlight_actor(id)`

Shows the standard target marker on a visible actor. Pass `nil` or `0` to clear it. The function returns `false` when a nonzero actor id is not visible or is dying.

This is a presentation primitive and does not select, attack, or cast on the actor. For example, `scripts/wasd.lua` combines it with the generic keyboard API and the pending-skill functions to implement Tab target cycling entirely in Lua.

### `goro.items()`

Returns an array of visible floor items.

Each item has:

- `id`
- `item_id`
- `amount`
- `x`
- `y`
- `identified`
- `distance`

### `goro.loot(id)`

Requests pickup for the floor item with this id.

Returns `true` if the item exists, otherwise `false`.

This uses the same path as a normal player click, including walking into pickup range. Scripts should avoid calling it every tick for the same item; keep a small retry delay.

### `goro.message(message)`

Sends console-style chat input. Returns `true` when the request was sent, otherwise `false`.

- Plain text sends a public message.
- Text beginning with `@` sends an atcommand as public chat for the server to interpret.
- Text beginning with `%` sends a party message.
- Text beginning with `$` sends a guild message.
- `/w Name message` or `/whisper Name message` sends a whisper.
- `/sit` and `/stand` change the player's resting state.

Scripts should keep a delay between messages instead of calling this every tick.

### `goro.inventory()`

Returns an array of carried inventory entries, ordered by inventory index.

Each entry has:

- `index`
- `item_id`
- `amount`
- `identified`
- `usable`

The `index` identifies this exact inventory entry and is the value accepted by `goro.use_item()`.

### `goro.use_item(index)`

Requests use of the usable inventory entry with this index.

Returns `true` when the entry exists, is usable, and the request was sent, otherwise `false`. Scripts should wait for the server inventory update or keep a retry delay instead of calling it every tick.

```lua
for _, item in ipairs(goro.inventory()) do
	if item.item_id == 501 then -- Red Potion
		goro.use_item(item.index)
		break
	end
end
```

### `goro.revive()`

Requests self-resurrection with a Token of Siegfried. It returns `true` when
the character is dead, a token is available, the current map permits its use,
and the request was sent. The server consumes the token and performs the
resurrection.

Bot ticks continue while the character is dead so scripts can decide whether
to revive, return to the save point manually, or remain dead.

## Example

This loots the nearest item first, then attacks the nearest enemy. It stops when HP is under 25%.

```lua
local function nearest(entries)
	local best = nil
	for _, entry in ipairs(entries) do
		if best == nil or entry.distance < best.distance then
			best = entry
		end
	end
	return best
end

local current_target = nil
local last_attack_at = 0
local attack_retry_seconds = 1.2
local double_strafe_id = 46
local current_item = nil
local last_loot_at = 0
local loot_retry_seconds = 1.0

function tick()
	local hp, max_hp = goro.hp()
	if max_hp > 0 and hp / max_hp < 0.25 then
		return
	end

	local item = nearest(goro.items())
	if item ~= nil then
		local now = os.clock()
		current_target = nil
		if current_item ~= item.id or now - last_loot_at >= loot_retry_seconds then
			current_item = item.id
			last_loot_at = now
			goro.loot(item.id)
		end
		return
	end
	current_item = nil

	local enemy = nearest(goro.enemies())
	if enemy ~= nil then
		local now = os.clock()
		if current_target ~= enemy.id or now - last_attack_at >= attack_retry_seconds then
			current_target = enemy.id
			last_attack_at = now
			if not goro.skill(enemy.id, double_strafe_id) then
				goro.attack(enemy.id)
			end
		end
	else
		current_target = nil
	end
end
```

The same script is available as
[`scripts/loot-and-attack.lua`](../scripts/loot-and-attack.lua).

## Bundled Keyboard and Controller Profile

Run [`scripts/wasd.lua`](../scripts/wasd.lua) to enable an optional
keyboard and controller control profile:

- Hold the physical WASD positions to move, including diagonally. These
  positions are ZQSD on AZERTY.
- Hold Space or West to pick up nearby items one at a time.
- Hold the physical F key or South to attack the selected enemy, or the nearest
  enemy within eight cells when none is selected. The character approaches if
  needed. Controller pointer clicks on UI and NPC dialogs take priority.

Targeting follows the same rules on both devices:

| Action | Keyboard | Controller |
| --- | --- | --- |
| Next / previous target | Tab / Shift+Tab | Right / left shoulder |
| Use hotbar slot | F1–F9 | R2 + South/East/West/North for slots 1–4; R2 + D-pad Up/Right/Down/Left for slots 5–8 |
| Confirm skill target | Enter | South |
| Cancel targeting | Escape | East |

Cycle enemies before choosing a skill to cast immediately on the selected
target if eligible. Alternatively, choose a skill first: it selects the nearest
eligible target, then cycling and confirmation let you choose whom to cast on.
An ineligible preselected target also enters this selection step. Enable
`/noshift` to include monsters when selecting Heal targets. With no skill armed,
Enter retains its normal chat behavior.

Ctrl, Alt, and Super/Command combinations remain available to the client;
holding these modifiers also pauses the profile's continuous controls.

The profile is implemented entirely in Lua. The Go API only exposes generic
keyboard state, movement, actions, target information, and highlighting
primitives.
