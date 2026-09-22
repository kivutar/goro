# Bot Scripting

Goro can run a Lua script while the player is in-game. This is intended for
local experimentation and simple automation.

Run a script with:

```sh
./goro --data-dir ~/OldRO --script scripts/loot-and-attack.lua
```

The script must define a global `tick()` function. Goro calls it roughly every
150 ms while the world mode is active. Lua state is retained across map changes
and map-server transfers for the current character.

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

There is no automatic reconnect or Lua API for answering interactive dialogs.
`--no-ui` only hides the graphical client's UI.

## Acolyte companion

Run [`scripts/companion.lua`](../scripts/companion.lua) on a second character:

```sh
./goro --data-dir ~/OldRO --script scripts/companion.lua
```

Say **follow** in public or party chat while near the bot. It replies
"Following you." in the same channel and switches to the speaker, even without
a party. Any visible player can become its leader this way. Commands ignore
case and surrounding whitespace; acknowledgements are limited to one every
two seconds. Say **heal** to request one Heal on yourself, even outside a party
or when your HP is unknown. Stay within eight cells. The bot stands and stops
walking first, waits for an ongoing cast, and reports insufficient SP or a
locally rejected skill request. Healing someone does not change its leader.

Edit the configuration at the top of the script to choose an initial `leader`,
or set it to `""` to wait for a chat command. Heal, Blessing, and Increase AGI
default to level 1; set these to levels your Acolyte has learned, or use 0 to
disable a skill. The same script works with the headless login options above.

The companion follows its visible leader when more than six cells away and
aims for a cell at least four cells away from that player. The distances are
configurable at the top of the script.
If the direct approach lands on a blocked cell, it tries nearby cells around
the leader while keeping the same gap. Movement requests are limited to once
per second.
It heals itself and its leader below 70% HP, prioritizing itself below 35%.
Automatic healing of the leader needs party HP updates; outside a party it
can follow and buff, but does not know when that player needs healing; use
**heal** to request it explicitly.
At critical HP it also uses red, orange, yellow, or white potions, including
the condensed variants, from its inventory. It periodically attempts Blessing
and Increase AGI on both characters, retaining SP for one Heal. At low SP it
sits to recover, standing when its companion moves away, danger appears,
it has enough SP for a needed Heal, or enough SP has recovered. Any nearby enemy
is treated conservatively as danger when deciding whether to rest or buff.

The script cannot inspect active buffs or learned skill levels, or confirm
that the server accepted a skill. Buff refresh timers are estimates based on
classic skill levels, and short pauses/retry limits avoid flooding requests.
If the leader disappears, it approaches the last observed cell for up to
`catch_up_seconds` (10 by default), including the usual gap between characters.
This lets it try to enter the same map portal. It waits after reaching that
cell or timing out, and resumes when the leader is visible again. It stops
pursuing a known dead leader.

The selected leader survives the bot's own map changes. Movement and resting
state are reset on arrival, so old coordinates are never used on the new map.
This is local catch-up, not route planning: it cannot track a distant teleport
or operate NPC travel dialogs.
Restart the script's client after editing its configuration; file changes are
not automatically reloaded.

## API

All functions are exposed through the global `goro` table.

### Map changes

An optional global `map_changed()` callback runs after map loading and the
load acknowledgement, including a warp within the current map. Actions sent
from the callback follow that acknowledgement. Use it to clear cached
positions, walking or casting decisions, while retaining long-lived choices
such as a leader. It is not called for a script's initial load. The `goro`
functions, including cached references, use the current world mode.

The WASD script clears movement and targeting state here, so a key held through
a portal starts movement from the new position.

Selecting another character starts a fresh script. Changing or removing
`--script` also replaces or closes the previous instance. Callback errors
disable the script as with `tick()` errors.

### Receiving chat

Define an optional global `chat(message)` callback to receive other players'
public and party messages while the script is loaded. It runs during game
updates, on the same thread as `tick()`. Normal chat display is unaffected.

```lua
function chat(message)
	-- message.channel: "public" or "party"
	-- message.sender_id: the speaker's ID from the server packet
	-- message.sender_name: known name, or the packet's name prefix if not loaded
	-- message.text: message body, with the matching "Name : " prefix removed
end
```

Public speakers must be visible players; party speakers can also be identified
from the party list. Names can be empty; use `sender_id` to identify a player.
Self echoes, NPC speech, announcements, chat-room messages, and whispers are
excluded. A callback error disables the script, as with `tick()` errors.

### Keyboard callbacks

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

## Bundled Keyboard Profile

Run [`scripts/wasd.lua`](../scripts/wasd.lua) to enable an optional
keyboard-oriented control profile:

- Hold the physical WASD positions to move, including diagonally. These
  positions are ZQSD on AZERTY.
- Hold Space to pick up nearby items one at a time.
- Hold the physical F key to attack a nearby enemy.
- After arming an actor-targeted skill, use Tab or Shift+Tab to cycle valid
  targets and Enter to cast.

Ctrl, Alt, and Super/Command combinations remain available to the client;
holding these modifiers also pauses the profile's continuous controls.

The profile is implemented entirely in Lua. The Go API only exposes generic
keyboard state, movement, actions, target information, and highlighting
primitives.
