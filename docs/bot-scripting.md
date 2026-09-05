# Bot Scripting

Goro can run a Lua script while the player is in-game. This is intended for
local experimentation and simple automation.

Run a script with:

```sh
./goro --data-dir ~/OldRO --script scripts/loot-and-attack.lua
```

The script must define a global `tick()` function. Goro calls it roughly every
150 ms while the world mode is active.

```lua
function tick()
	-- bot logic here
end
```

## API

All functions are exposed through the global `goro` table.

Scripts may also define an optional global `input()` function. Goro calls it
once per frame so keyboard edges can be handled without waiting for the slower
bot tick.

### `goro.keyboard`

The keyboard API uses layout-independent physical key names such as `"KeyW"`,
`"Tab"`, and `"ShiftLeft"`. Letter codes describe physical key positions, not
the glyph printed by the current layout. For example, the physical WASD
positions are ZQSD on an AZERTY keyboard.

- `available()` reports whether keyboard input is available to the script. It is `false` while a UI control has keyboard focus.
- `is_down(code)` reports held state.
- `was_pressed(code)` and `was_released(code)` inspect edges without consuming them.
- `consume_press(code)` consumes a press edge and returns whether one was available. Held state is unchanged.
- `text()` returns the frame's layout-translated text input.

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

Requests an actor-targeted skill on the actor with this id. `skill` can be either a numeric skill id or a learned skill name such as `"AC_DOUBLE"` or `"AL_HEAL"`.

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

The profile is implemented entirely in Lua. The Go API only exposes generic
keyboard state, movement, actions, target information, and highlighting
primitives.
