-- Physical WASD positions: these keys are ZQSD on an AZERTY keyboard.
local controls = { "KeyW", "KeyA", "KeyS", "KeyD" }
-- Positional names work across Xbox, PlayStation and other controller labels.
local stick_deadzone = 0.3
local attack_button = "south"
local loot_button = "west"
local skill_buttons = { "south", "east", "west", "north", "dpad_up", "dpad_right", "dpad_down", "dpad_left" }
local skill_buttons_held = {}
local selected_enemy_id = nil
local pad_attack_down = false
local ui_pointer = false
local horizon = 8
local refill_distance = 3
local action_radius = 8
local active_dx = 0
local active_dy = 0
local target_x = nil
local target_y = nil
local walk_sequence = nil
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
	walk_sequence = nil
end

local function clear_skill_target()
	if skill_target_id ~= nil or skill_target_skill_id ~= nil then
		goro.highlight_actor(selected_enemy_id)
	end
	skill_target_id = nil
	skill_target_skill_id = nil
end

local function sort_targets(targets, x, y)
	table.sort(targets, function(a, b)
		local adx = a.x - x
		local ady = a.y - y
		local bdx = b.x - x
		local bdy = b.y - y
		local adistance = adx * adx + ady * ady
		local bdistance = bdx * bdx + bdy * bdy
		if adistance == bdistance then
			return a.id < b.id
		end
		return adistance < bdistance
	end)
	return targets
end

local function skill_targets(pending)
	local caster_x = pending.caster_x or goro.player().x
	local caster_y = pending.caster_y or goro.player().y
	-- Shift reverses keyboard cycling; /noshift controls both devices' targets.
	return sort_targets(goro.skill_targets(true), caster_x, caster_y)
end

local function contains_target(targets, id)
	for _, target in ipairs(targets) do
		if target.id == id then return true end
	end
	return false
end

local function highlight_target(id)
	if goro.highlight_actor(id) then return id end
	goro.highlight_actor(nil)
	return nil
end

local function next_target_id(targets, current_id, reverse)
	if #targets == 0 then return nil end
	local current = nil
	for index, target in ipairs(targets) do
		if target.id == current_id then
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

	return targets[next_index].id
end

local function sync_skill_target()
	local pending = goro.pending_skill()
	if pending == nil or pending.target ~= "actor" then
		clear_skill_target()
	elseif skill_target_skill_id ~= pending.id then
		clear_skill_target()
		skill_target_skill_id = pending.id
	end
	return pending
end

local function cycle_target(reverse)
	local pending = sync_skill_target()
	if pending ~= nil then
		if pending.target ~= "actor" then return false end
		skill_target_id = highlight_target(next_target_id(skill_targets(pending), skill_target_id, reverse))
	else
		local player = goro.player()
		local targets = sort_targets(goro.enemies(), player.x, player.y)
		selected_enemy_id = highlight_target(next_target_id(targets, selected_enemy_id, reverse))
		attack_target_id = nil
	end
	return true
end

local function confirm_skill()
	local pending = sync_skill_target()
	if pending == nil or pending.target ~= "actor" or skill_target_id == nil then return false end
	goro.use_pending_skill(skill_target_id)
	clear_skill_target()
	skill_input_handled = true
	return true
end

local function cancel_target()
	if goro.pending_skill() == nil and selected_enemy_id == nil then return false end
	goro.cancel_skill()
	selected_enemy_id = nil
	clear_skill_target()
	goro.highlight_actor(nil)
	goro.stop()
	skill_input_handled = true
	return true
end

local function use_shortcut(slot)
	sync_skill_target()
	local target_id = skill_target_id or selected_enemy_id
	local used, skill_id = goro.use_shortcut(slot)
	if not used then return end
	local pending = sync_skill_target()
	if pending ~= nil and pending.id == skill_id and pending.target == "actor" then
		local targets = skill_targets(pending)
		if contains_target(targets, target_id) and goro.use_pending_skill(target_id) then
			clear_skill_target()
		else
			skill_target_id = highlight_target(next_target_id(targets, nil, false))
		end
	end
	skill_input_handled = true
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
			walk_sequence = goro.player().walk_sequence
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
	local target = select_nearest(goro.enemies(), selected_enemy_id or attack_target_id)
	if target == nil then
		attack_target_id = nil
		return false
	end
	if target.id ~= attack_target_id or now - last_attack_at >= attack_retry_seconds then
		if goro.attack(target.id) then
			selected_enemy_id = highlight_target(target.id)
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
	if selected_enemy_id ~= nil and not contains_target(goro.enemies(), selected_enemy_id) then
		selected_enemy_id = nil
		goro.highlight_actor(skill_target_id)
	end
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
	if not goro.in_game() then return end
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
	local slot = code:match("^F([1-9])$")
	if slot ~= nil and goro.keyboard.consume_press(code) then
		use_shortcut(tonumber(slot))
	elseif code == "Tab" and cycle_target(goro.keyboard.is_down("ShiftLeft") or goro.keyboard.is_down("ShiftRight"))
		or code == "Enter" and confirm_skill()
		or code == "Escape" and cancel_target() then
		goro.keyboard.consume_press(code)
	end
end

-- Runs before the client's pointer fallback. All controller bindings live here;
-- the Go bridge only exposes camera, hotbar and dialog actions.
function gamepad(dt)
	if goro.gamepad.was_released("south") or not goro.gamepad.is_down("south") then
		pad_attack_down = false
	end
	if pad_attack_down then goro.gamepad.consume("south") end
	for _, button in ipairs(skill_buttons) do
		if goro.gamepad.was_released(button) or not goro.gamepad.is_down(button) then
			skill_buttons_held[button] = nil
		end
		if skill_buttons_held[button] then goro.gamepad.consume(button) end
	end
	if not goro.gamepad.connected() then return end
	local ui_active = goro.ui.active()
	if not ui_active then ui_pointer = false end
	if ui_active then
		-- Moving the pointer restores clicks; using the D-pad resumes menu
		-- navigation. Forms remain usable with either style of control.
		if not goro.in_game() and (math.abs(goro.gamepad.axis("right_x")) > 0.2 or math.abs(goro.gamepad.axis("right_y")) > 0.2) then
			ui_pointer = true
		end
		for _, button in ipairs({ "dpad_up", "dpad_down", "dpad_left", "dpad_right" }) do
			if goro.gamepad.was_pressed(button) then ui_pointer = false end
		end
		if ui_pointer then return end
		goro.gamepad.consume("south")
		goro.gamepad.consume("east")
		if goro.gamepad.was_pressed("dpad_up") then goro.ui.control("up") end
		if goro.gamepad.was_pressed("dpad_down") then goro.ui.control("down") end
		if goro.gamepad.was_pressed("dpad_left") then goro.ui.control("left") end
		if goro.gamepad.was_pressed("dpad_right") then goro.ui.control("right") end
		if goro.gamepad.was_pressed("south") then
			skill_buttons_held.south = true
			goro.ui.control("confirm")
		elseif goro.gamepad.was_pressed("east") then
			skill_buttons_held.east = true
			goro.ui.control("cancel")
		end
		return
	end
	if not goro.gamepad.available() then return end

	if goro.gamepad.axis("left_trigger") > 0.5 or goro.gamepad.axis("right_trigger") > 0.5 then
		goro.gamepad.consume_pointer()
		local function camera_axis(name)
			local value = goro.gamepad.axis(name)
			if math.abs(value) < 0.2 then return 0 end
			return value
		end
		if goro.gamepad.axis("right_trigger") > 0.5 then
			goro.zoom_camera(camera_axis("right_y") * dt * 60)
		else
			goro.rotate_camera(camera_axis("right_x") * dt * 100, camera_axis("right_y") * dt * 45)
		end
	end
	local pending = sync_skill_target()
	local previous = goro.gamepad.was_pressed("left_shoulder")
	local next_target = goro.gamepad.was_pressed("right_shoulder")
	if previous or next_target then
		cycle_target(previous)
	end
	if goro.gamepad.axis("right_trigger") > 0.5 then
		pad_attack_down = false
		for _, button in ipairs(skill_buttons) do
			goro.gamepad.consume(button)
			if goro.gamepad.is_down(button) then skill_buttons_held[button] = true end
		end
		for slot, button in ipairs(skill_buttons) do
			if goro.gamepad.was_pressed(button) then
				use_shortcut(slot)
				break
			end
		end
		return
	end
	if goro.gamepad.was_pressed("east") and not skill_buttons_held.east
		and cancel_target() then
		goro.gamepad.consume("east")
		skill_buttons_held.east = true
		return
	end
	if not goro.pointer_over_ui() and not skill_buttons_held.south then
		if pending ~= nil and pending.target == "actor" and skill_target_id ~= nil then
			goro.gamepad.consume("south")
			if goro.gamepad.was_pressed("south") then
				confirm_skill()
				skill_buttons_held.south = true
			end
		elseif pending == nil and goro.gamepad.was_pressed("south")
			and select_nearest(goro.enemies(), selected_enemy_id) ~= nil then
			pad_attack_down = true
			goro.gamepad.consume("south")
		end
	end
end

function input()
	if not goro.in_game() then return end
	local pending = sync_skill_target()
	if goro.gamepad.was_pressed("left_stick") then confirm_skill() end
	if skill_input_handled then
		skill_input_handled = false
		fight_down = false
		loot_down = false
		attack_target_id = nil
		loot_target_id = nil
		-- Arming a skill leaves the current walk active; keep tracking its release.
		return
	end

	local controls_enabled = not shortcut_modifier_down()
	local pad_actions = goro.gamepad.axis("right_trigger") <= 0.5
	fight_down = controls_enabled and pending == nil and (goro.keyboard.is_down("KeyF")
		or (pad_actions and not skill_buttons_held.south
			and pad_attack_down and goro.gamepad.is_down(attack_button)))
	loot_down = controls_enabled and (goro.keyboard.is_down("Space")
		or (pad_actions and not skill_buttons_held.west and goro.gamepad.is_down(loot_button)))
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
		local x = goro.gamepad.axis("left_x")
		local y = goro.gamepad.axis("left_y")
		-- Skill chords keep their D-pad buttons until release, even after R2.
		if pad_actions then
			if not skill_buttons_held.dpad_left and goro.gamepad.is_down("dpad_left") then x = x - 1 end
			if not skill_buttons_held.dpad_right and goro.gamepad.is_down("dpad_right") then x = x + 1 end
			if not skill_buttons_held.dpad_up and goro.gamepad.is_down("dpad_up") then y = y - 1 end
			if not skill_buttons_held.dpad_down and goro.gamepad.is_down("dpad_down") then y = y + 1 end
		end
		if x * x + y * y > stick_deadzone * stick_deadzone then
			local angle = math.atan2(-y, x) + math.rad(goro.camera_yaw())
			local octant = math.floor(angle / (math.pi / 4) + 0.5) * math.pi / 4
			dx = dx + math.floor(math.cos(octant) + 0.5)
			dy = dy + math.floor(math.sin(octant) + 0.5)
		end
		-- Simultaneous keyboard/controller input must not double the stride.
		dx = math.max(-1, math.min(1, dx))
		dy = math.max(-1, math.min(1, dy))
	end

	local player = goro.player()
	-- A pointer action or skill chase may have replaced this profile's walk.
	if walk_sequence ~= nil and walk_sequence ~= player.walk_sequence then clear_target() end
	if dx == 0 and dy == 0 then
		if active_dx ~= 0 or active_dy ~= 0 then
			if goro.stop() then
				clear_target()
			end
		end
		return
	end

	if dx ~= active_dx or dy ~= active_dy or not player.moving then
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
