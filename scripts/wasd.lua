-- Physical WASD positions: these keys are ZQSD on an AZERTY keyboard.
local controls = { "KeyW", "KeyA", "KeyS", "KeyD" }
local horizon = 8
local refill_distance = 3
local action_radius = 8
local active_dx = 0
local active_dy = 0
local target_x = nil
local target_y = nil
local loot_target_id = nil
local attack_target_id = nil
local fight_down = false
local loot_down = false
local attack_retry_seconds = 1.2
local loot_retry_seconds = 1.0
local last_attack_at = -math.huge
local last_loot_at = -math.huge
local skill_target_id = nil
local skill_target_skill_id = nil
local skill_input_handled = false

local skill_target_enemy = 1
local skill_target_self = 4
local skill_target_friend = 16
local skill_target_pet = 64
local skill_target_homunculus = 128

-- Leave modified shortcuts to the client. Shift remains available for
-- movement and reverse target cycling with Shift+Tab.
local function shortcut_modifier_down()
	return goro.keyboard.is_down("ControlLeft") or goro.keyboard.is_down("ControlRight")
		or goro.keyboard.is_down("AltLeft") or goro.keyboard.is_down("AltRight")
		or goro.keyboard.is_down("MetaLeft") or goro.keyboard.is_down("MetaRight")
end

local function clear_target()
	active_dx = 0
	active_dy = 0
	target_x = nil
	target_y = nil
end

local function clear_skill_target()
	if skill_target_id ~= nil or skill_target_skill_id ~= nil then
		goro.highlight_actor(nil)
	end
	skill_target_id = nil
	skill_target_skill_id = nil
end

function map_changed()
	clear_target()
	clear_skill_target()
	loot_target_id, attack_target_id = nil, nil
	fight_down, loot_down = false, false
	last_attack_at, last_loot_at = -math.huge, -math.huge
	skill_input_handled = false
end

local function has_flag(value, flag)
	return value % (flag * 2) >= flag
end

local function append_skill_targets(targets, seen, entries)
	for _, entry in ipairs(entries) do
		if entry.id ~= nil and entry.id ~= 0 and not entry.dead and not seen[entry.id] then
			seen[entry.id] = true
			table.insert(targets, entry)
		end
	end
end

local function skill_targets(pending)
	local targets = {}
	local seen = {}
	local flags = pending.type
	if has_flag(flags, skill_target_enemy) or has_flag(flags, skill_target_pet) then
		append_skill_targets(targets, seen, goro.enemies())
	end
	if has_flag(flags, skill_target_friend) or has_flag(flags, skill_target_self) then
		append_skill_targets(targets, seen, { goro.player() })
		append_skill_targets(targets, seen, goro.players())
		append_skill_targets(targets, seen, goro.companions())
	end
	if has_flag(flags, skill_target_homunculus) then
		append_skill_targets(targets, seen, goro.companions())
	end

	local caster_x = pending.caster_x or goro.player().x
	local caster_y = pending.caster_y or goro.player().y
	table.sort(targets, function(a, b)
		local adx = a.x - caster_x
		local ady = a.y - caster_y
		local bdx = b.x - caster_x
		local bdy = b.y - caster_y
		local adistance = adx * adx + ady * ady
		local bdistance = bdx * bdx + bdy * bdy
		if adistance == bdistance then
			return a.id < b.id
		end
		return adistance < bdistance
	end)
	return targets
end

local function cycle_skill_target(pending, reverse)
	local targets = skill_targets(pending)
	if #targets == 0 then
		clear_skill_target()
		skill_target_skill_id = pending.id
		return
	end

	local current = nil
	for index, target in ipairs(targets) do
		if target.id == skill_target_id then
			current = index
			break
		end
	end
	local next_index = 1
	if reverse then
		next_index = #targets
		if current ~= nil then
			next_index = (current - 2) % #targets + 1
		end
	elseif current ~= nil then
		next_index = current % #targets + 1
	end

	local id = targets[next_index].id
	if goro.highlight_actor(id) then
		skill_target_id = id
	else
		clear_skill_target()
		skill_target_skill_id = pending.id
	end
end

local function handle_skill_target_input(code)
	local pending = goro.pending_skill()
	if pending == nil or pending.target ~= "actor" then
		clear_skill_target()
		return false
	end
	if skill_target_skill_id ~= pending.id then
		clear_skill_target()
		skill_target_skill_id = pending.id
	end
	if code == "Escape" then
		clear_skill_target()
		return false
	end
	if code == "Tab" and goro.keyboard.consume_press(code) then
		local reverse = goro.keyboard.is_down("ShiftLeft")
			or goro.keyboard.is_down("ShiftRight")
		cycle_skill_target(pending, reverse)
		return true
	end
	if code == "Enter" and skill_target_id ~= nil and goro.keyboard.consume_press(code) then
		goro.use_pending_skill(skill_target_id)
		clear_skill_target()
		return true
	end
	return false
end

local function request_walk(player, dx, dy, max_distance)
	for distance = max_distance, 1, -1 do
		local x = player.x + dx * distance
		local y = player.y + dy * distance
		if goro.walk(x, y) then
			active_dx = dx
			active_dy = dy
			target_x = x
			target_y = y
			return true
		end
	end
	return false
end

local function select_nearest(entries, current_id)
	local nearest = nil
	for _, entry in ipairs(entries) do
		if entry.id == current_id then
			return entry
		end
		if entry.distance <= action_radius
			and (nearest == nil or entry.distance < nearest.distance) then
			nearest = entry
		end
	end
	return nearest
end

local function schedule_attack(now)
	local target = select_nearest(goro.enemies(), attack_target_id)
	if target == nil then
		attack_target_id = nil
		return false
	end
	if target.id ~= attack_target_id or now - last_attack_at >= attack_retry_seconds then
		if goro.attack(target.id) then
			attack_target_id = target.id
			last_attack_at = now
		end
	end
	return attack_target_id ~= nil
end

local function schedule_loot(now)
	local target = select_nearest(goro.items(), loot_target_id)
	if target == nil then
		loot_target_id = nil
		return false
	end
	if target.id ~= loot_target_id or now - last_loot_at >= loot_retry_seconds then
		if goro.loot(target.id) then
			loot_target_id = target.id
			last_loot_at = now
		end
	end
	return loot_target_id ~= nil
end

function tick()
	local now = os.clock()
	if fight_down and schedule_attack(now) then
		loot_target_id = nil
		clear_target()
		return
	end
	if loot_down and schedule_loot(now) then
		clear_target()
	end
end

function keypress(code)
	if shortcut_modifier_down() then
		return
	end
	for _, control in ipairs(controls) do
		if code == control then
			goro.keyboard.consume_press(code)
			return
		end
	end
	if code == "Space" or code == "KeyF" then
		goro.keyboard.consume_press(code)
		return
	end
	if handle_skill_target_input(code) then
		skill_input_handled = true
	end
end

function input()
	handle_skill_target_input(nil)
	if skill_input_handled then
		skill_input_handled = false
		fight_down = false
		loot_down = false
		attack_target_id = nil
		loot_target_id = nil
		clear_target()
		return
	end

	local controls_enabled = not shortcut_modifier_down()
	fight_down = controls_enabled and goro.keyboard.is_down("KeyF")
	loot_down = controls_enabled and goro.keyboard.is_down("Space")
	if not fight_down then
		attack_target_id = nil
	end
	if not loot_down then
		loot_target_id = nil
	end
	if attack_target_id ~= nil or loot_target_id ~= nil then
		clear_target()
		return
	end

	local dx = 0
	local dy = 0
	if controls_enabled then
		if goro.keyboard.is_down("KeyW") then dy = dy + 1 end
		if goro.keyboard.is_down("KeyA") then dx = dx - 1 end
		if goro.keyboard.is_down("KeyS") then dy = dy - 1 end
		if goro.keyboard.is_down("KeyD") then dx = dx + 1 end
	end

	if dx == 0 and dy == 0 then
		if active_dx ~= 0 or active_dy ~= 0 then
			if goro.stop() then
				clear_target()
			end
		end
		return
	end

	local player = goro.player()
	if dx ~= active_dx or dy ~= active_dy then
		request_walk(player, dx, dy, horizon)
		return
	end

	if target_x ~= nil and target_y ~= nil then
		local remaining = math.max(math.abs(target_x - player.x), math.abs(target_y - player.y))
		if remaining <= refill_distance then
			request_walk(player, dx, dy, horizon)
		end
	end
end
