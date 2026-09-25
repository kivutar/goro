-- A small Acolyte companion. Say "follow" nearby to become its leader.
-- A party is optional for following, but supplies HP for automatic healing.
-- Skill levels are explicit because Lua cannot inspect learned skills yet.
local config = {
	leader = "Kivy", -- Initial leader; use "" to wait for a chat command.
	heal_level = 1,
	blessing_level = 1, -- Set a skill level to 0 to disable it.
	increase_agi_level = 1,
	follow_start = 6,
	follow_stop = 4,
	catch_up_seconds = 10,
	heal_below = 0.70,
	critical_hp = 0.35,
	rest_below_sp = 0.15,
	resume_at_sp = 0.70,
}

-- Classic costs/ranges match roBrowser SkillInfo; buff durations and the
-- Increase AGI HP cost match rAthena's pre-renewal skill_db.yml.
local heal = { name = "AL_HEAL", level = config.heal_level, sp = 10 + 3 * config.heal_level }
local buffs = {
	{ name = "AL_BLESSING", level = config.blessing_level, sp = 24 + 4 * config.blessing_level, hp = 0 },
	{ name = "AL_INCAGI", level = config.increase_agi_level, sp = 15 + 3 * config.increase_agi_level, hp = 15 },
}
local healing_items = { [501] = true, [502] = true, [503] = true, [504] = true,
	[545] = true, [546] = true, [547] = true }
local following, resting = false, false
local next_action, next_walk, next_potion = 0, 0, 0
local last_hp, unsafe_until = nil, 0
local skill_retry, buff_due = {}, {}
local leader_id, leader_changed = nil, false
local last_leader = nil
local heal_request = nil
local next_reply = 0

local function hp_ratio(actor)
	if actor == nil or actor.max_hp <= 0 then
		return 1 -- Unknown HP is not an injury.
	end
	return actor.hp / actor.max_hp
end

local function find_leader()
	for _, actor in ipairs(goro.players()) do
		if (leader_id ~= nil and actor.id == leader_id)
			or (leader_id == nil and config.leader ~= "" and actor.name == config.leader) then
			if actor.dead or (actor.max_hp > 0 and actor.hp <= 0) then
				buff_due[actor.id] = nil
				last_leader = nil
				return nil
			end
			return actor
		end
	end
	return nil
end

local function reply(channel, text, now)
	if now >= next_reply then
		goro.message((channel == "party" and "%" or "") .. text)
		next_reply = now + 2
	end
end

function chat(message)
	if message.channel ~= "public" and message.channel ~= "party" then
		return
	end
	local command = message.text:match("^%s*(.-)%s*$"):lower()
	local me, now = goro.player(), os.clock()
	if (command ~= "follow" and command ~= "heal") or me.dead then
		return
	end
	for _, actor in ipairs(goro.players()) do
		if actor.id == message.sender_id and not actor.dead and hp_ratio(actor) > 0 then
			if command == "heal" then
				if heal.level <= 0 then
					reply(message.channel, "Heal is disabled in my configuration.", now)
				elseif actor.distance > 8 then
					reply(message.channel, "Come a little closer for Heal.", now)
				elseif me.sp < heal.sp then
					reply(message.channel, "I need more SP to heal.", now)
				elseif now < (skill_retry[heal.name] or 0) then
					reply(message.channel, "I can't cast Heal right now.", now)
				else
					heal_request = { id = actor.id, expires = now + 5, channel = message.channel }
				end
				return
			end
			leader_changed = leader_changed or leader_id ~= actor.id
			leader_id = actor.id
			next_walk = 0
			last_leader = { x = actor.x, y = actor.y, seen_at = now }
			reply(message.channel, "Following you.", now)
			return
		end
	end
end

function map_changed()
	-- A warp cancels walking/casting and changes coordinates, but keeps our
	-- chosen leader and buff timers. Never reuse an old map's destination.
	following, resting, leader_changed = false, false, false
	next_action, next_walk = 0, 0
	last_hp, unsafe_until, last_leader = nil, 0, nil
	heal_request = nil
end

local function requested_heal_target(now)
	if heal_request == nil then
		return nil
	end
	if now < heal_request.expires then
		for _, actor in ipairs(goro.players()) do
			if actor.id == heal_request.id and actor.distance <= 8 and not actor.dead and hp_ratio(actor) > 0 then
				return actor
			end
		end
	end
	heal_request = nil
	return nil
end

local function follow_destination(leader, me, now)
	if leader ~= nil then
		last_leader = { x = leader.x, y = leader.y, seen_at = now }
		return leader, false
	end
	if last_leader == nil then
		return nil, true
	end
	local dx, dy = last_leader.x - me.x, last_leader.y - me.y
	if now - last_leader.seen_at >= config.catch_up_seconds or (dx == 0 and dy == 0) then
		last_leader = nil
		return nil, true
	end
	-- A map portal removes the leader before we arrive. Finish approaching
	-- the last observed cell, even inside the usual following distance.
	return { x = last_leader.x, y = last_leader.y, distance = math.sqrt(dx * dx + dy * dy) }, true
end

local function stop_following(now)
	if not following then
		return false
	end
	if goro.stop() then
		following = false
	end
	next_action = now + 0.3
	return true
end

local function set_rest(want, now)
	if stop_following(now) then
		return
	end
	if goro.message(want and "/sit" or "/stand") then
		resting = want
	end
	next_action = now + 0.5
end

local function use_potion(now)
	if now < next_potion then
		return false
	end
	for _, item in ipairs(goro.inventory()) do
		if item.usable and item.amount > 0 and healing_items[item.item_id] then
			next_potion = now + 1
			return goro.use_item(item.index)
		end
	end
	return false
end

local function cast(skill, target, me, now)
	if skill.level <= 0 or me.sp < skill.sp or now < (skill_retry[skill.name] or 0) then
		return false
	end
	if stop_following(now) then
		return true
	end
	local accepted = goro.skill(target.id, skill.name, skill.level)
	if accepted then
		-- Leave time for Increase AGI's cast and the action delay. The current
		-- API reports accepted requests, not server-confirmed skill results.
		next_action = now + 2.2
		if skill ~= heal then
			buff_due[target.id] = buff_due[target.id] or {}
			buff_due[target.id][skill.name] = now + 40 + 20 * skill.level - 5
		end
	else
		-- Unlearned skills and rejected requests must not be retried each tick.
		skill_retry[skill.name] = now + 10
		next_action = now + 0.5
	end
	return true, accepted
end

local function follow_cells(leader, me)
	local cells, seen = {}, {}
	local function add(x, y)
		local key = x .. "," .. y
		if not seen[key] then
			seen[key] = true
			table.insert(cells, { x = x, y = y })
		end
	end
	local dx, dy = me.x - leader.x, me.y - leader.y
	local distance = math.sqrt(dx * dx + dy * dy)
	if distance == 0 then return cells end
	local function outward(v) return v < 0 and math.floor(v) or math.ceil(v) end
	for gap = config.follow_stop, config.follow_stop + 1 do
		add(leader.x + outward(dx * gap / distance), leader.y + outward(dy * gap / distance))
	end
	-- If the direct approach is blocked, try surrounding cells at the same
	-- distance from the leader, starting with those closest to us.
	local nearby = {}
	local radius = math.ceil(config.follow_stop + 1)
	for x = -radius, radius do
		for y = -radius, radius do
			local gap_squared = x * x + y * y
			if gap_squared >= config.follow_stop ^ 2 and gap_squared <= (config.follow_stop + 1) ^ 2 then
				table.insert(nearby, { x = leader.x + x, y = leader.y + y,
					distance = (x - dx) ^ 2 + (y - dy) ^ 2 })
			end
		end
	end
	table.sort(nearby, function(a, b)
		if a.distance ~= b.distance then return a.distance < b.distance end
		if a.x ~= b.x then return a.x < b.x end
		return a.y < b.y
	end)
	for _, cell in ipairs(nearby) do add(cell.x, cell.y) end
	return cells
end

local function follow(leader, me, now, catch_up)
	if leader == nil then
		return false
	end
	if leader.distance <= (catch_up and 0 or config.follow_start) and not following then
		return false
	end
	if now >= next_walk then
		next_walk = now + 1
		if catch_up then
			if goro.walk(leader.x, leader.y) then following = true end
		else
			for _, cell in ipairs(follow_cells(leader, me)) do
				if cell.x == me.x and cell.y == me.y then following = false; return false end
				if goro.walk(cell.x, cell.y) then
					following = true
					break
				end
			end
		end
	end
	return true
end

function tick()
	local now = os.clock()
	local me = goro.player()
	if me.dead then
		following, resting = false, false
		buff_due, skill_retry = {}, {}
		next_action, next_walk, next_potion = 0, 0, 0
		last_hp = nil
		last_leader = nil
		heal_request = nil
		return
	end
	if me.max_hp <= 0 or me.max_sp <= 0 then
		return
	end
	local leader = find_leader()
	local requested = requested_heal_target(now)
	local destination, catch_up = follow_destination(leader, me, now)
	local hp, sp = hp_ratio(me), me.sp / me.max_sp
	if last_hp ~= nil and me.hp < last_hp then
		unsafe_until = now + 5
	end
	last_hp = me.hp
	local threatened = now < unsafe_until
	for _, enemy in ipairs(goro.enemies()) do
		if enemy.distance <= 6 then
			threatened = true
			break
		end
	end

	if hp < config.critical_hp and use_potion(now) then
		return
	end
	if now < next_action then
		return
	end
	if leader_changed then
		if resting then
			set_rest(false, now)
			return
		end
		if stop_following(now) then
			return
		end
		leader_changed = false
	end
	local leader_near = leader ~= nil and leader.distance <= 8
	local needs_heal = requested ~= nil or hp < config.heal_below
		or (leader_near and hp_ratio(leader) < config.heal_below)
	local can_heal = heal.level > 0 and me.sp >= heal.sp and now >= (skill_retry[heal.name] or 0)
	if resting then
		if threatened or (destination ~= nil and destination.distance > (catch_up and 0 or config.follow_start))
			or (needs_heal and can_heal)
			or (sp >= config.resume_at_sp and hp >= config.heal_below) then
			set_rest(false, now)
		end
		return
	end
	if following and (destination == nil or destination.distance <= (catch_up and 0 or config.follow_stop)) then
		stop_following(now)
		return
	end

	-- Prefer our own survival when critically hurt; otherwise heal the more
	-- injured member. Stay inside skill range to avoid an unobserved chase.
	local target = nil
	if hp < config.heal_below then
		target = me
	end
	if hp >= config.critical_hp and leader_near and hp_ratio(leader) < config.heal_below
		and (target == nil or hp_ratio(leader) < hp) then
		target = leader
	end
	if hp >= config.critical_hp and requested ~= nil then target = requested end
	if target ~= nil then
		local acted, accepted = cast(heal, target, me, now)
		if target == requested and accepted ~= nil then
			reply(heal_request.channel, accepted and "Healing you." or "I couldn't cast Heal. Check my learned skill level.", now)
			heal_request = nil
		end
		if acted then return end
	end
	if follow(destination, me, now, catch_up) then
		return
	end
	local cannot_heal = heal.level <= 0 or me.sp < heal.sp or now < (skill_retry[heal.name] or 0)
	if not threatened and (sp < config.rest_below_sp or (hp < config.heal_below and cannot_heal)) then
		set_rest(true, now)
		return
	end

	-- Buff only while together and healthy, keeping at least one Heal's SP.
	-- Timers are estimates: Lua cannot observe active buffs or cast failures.
	if leader ~= nil and not threatened and hp >= config.heal_below and sp >= 0.5 then
		for _, ally in ipairs({ leader, me }) do
			for _, buff in ipairs(buffs) do
				local due = buff_due[ally.id] and buff_due[ally.id][buff.name] or 0
				if now >= due and me.sp >= buff.sp + heal.sp
					and me.hp - buff.hp >= me.max_hp * config.heal_below then
					if cast(buff, ally, me, now) then
						return
					end
				end
			end
		end
	end
end
