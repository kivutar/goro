-- Local stress test: every standard first and second class (no babies).
-- Uses normal client actions; accounts need no GM permissions.
local index = tonumber(os.getenv("GORO_STRESS_INDEX")) or 1
local roster = dofile("scripts/stress-test/roster.lua")
local role = assert(roster[index], "unknown stress-test character")
local skills = role.skills
-- These waits are generated from the server's skill data and learned levels.
-- Bot ticks are at least 150 ms; cast times omit any reduction from DEX.
local skill_wait = role.waits
math.randomseed(goro.player().id)

local stagger = (index - 1) % 19 + 1
local ticks, next_action, next_skill, next_potion = 0, stagger * 2, 0, 0
local target, target_since
local resting = false
local attacks, casts, pickups = 0, 0, 0
local loot_attempts = 0
local next_support, support_index, next_heal = stagger * 3, 1, 0
local skill_index = (index - 1) % #skills + 1
local next_buff = {}
local center_x, center_y, radius = 216, 255, 10

local function inside(entry)
	return math.abs(entry.x - center_x) <= radius
		and math.abs(entry.y - center_y) <= radius
end

local function ratio(value, maximum)
	return maximum > 0 and value / maximum or 1
end

local function use_potion(item_id)
	for _, entry in ipairs(goro.inventory()) do
		if entry.item_id == item_id and entry.amount > 0 and goro.use_item(entry.index) then
			return true
		end
	end
	return false
end

local function cast(id, skill)
	if not goro.skill(id, skill) then return false end
	casts = casts + 1
	next_action = ticks + (skill_wait[skill] or 6)
	return true
end

print(string.format("[stress%02d] %s in %s active near prt_fild08 %d %d", index, role.name, role.party, center_x, center_y))

function tick()
	ticks = ticks + 1
	local player = goro.player()
	if ticks % 200 == 0 then
		print(string.format("[stress%02d] hp=%d/%d sp=%d/%d pos=%d,%d requests: attack=%d skill=%d loot=%d",
			index, player.hp, player.max_hp, player.sp, player.max_sp,
			player.x, player.y, attacks, casts, pickups))
	end
	if ticks < next_action then return end
	if player.dead then
		goro.revive()
		target, resting = nil, false
		next_buff = {}
		next_action = ticks + 30
		return
	end

	local hp = ratio(player.hp, player.max_hp)
	local sp = ratio(player.sp, player.max_sp)
	if ticks >= next_potion then
		if (hp < 0.65 and use_potion(503)) or (sp < 0.35 and use_potion(505)) then
			next_potion = ticks + 10
		end
	end
	if resting then
		if hp >= 0.85 and sp >= 0.70 then
			goro.message("/stand")
			resting = false
		end
		next_action = ticks + 10
		return
	elseif hp < 0.25 or sp < 0.08 then
		goro.stop()
		goro.message("/sit")
		resting, target = true, nil
		next_action = ticks + 10
		return
	end

	if not inside(player) then
		goro.walk(center_x, center_y)
		target = nil
		next_action = ticks + 20
		return
	end

	-- Party membership and HP come from the server, including its range rules.
	if (role.healing ~= "" or #role.support > 0)
		and (ticks >= next_heal or ticks >= next_support) then
		local allies = {player}
		local injured = role.healing ~= "AM_POTIONPITCHER" and hp < 0.8 and player or nil
		for _, ally in ipairs(goro.players()) do
			if ally.party_member and not ally.dead and inside(ally) and ally.distance <= 9 then
				allies[#allies + 1] = ally
				if ally.max_hp > 0 and ratio(ally.hp, ally.max_hp) < 0.8
					and (not injured or ratio(ally.hp, ally.max_hp) < ratio(injured.hp, injured.max_hp)) then
					injured = ally
				end
			end
		end
		if ticks >= next_heal then
			next_heal = ticks + 25
			if role.healing ~= "" and injured and cast(injured.id, role.healing) then return end
		end
		if ticks >= next_support then
			next_support = ticks + 100
			if #role.support > 0 then
				local skill = role.support[support_index]
				support_index = support_index % #role.support + 1
				if cast(allies[math.random(#allies)].id, skill) then return end
			end
		end
	end

	for i, buff in ipairs(role.buffs) do
		if ticks >= (next_buff[i] or stagger * 4) then
			next_buff[i] = ticks + buff.interval
			if cast(player.id, buff.skill) then return end
		end
	end

	local enemies = goro.enemies()
	local enemy
	for _, entry in ipairs(enemies) do
		if entry.id == target and inside(entry) and ticks - target_since < 100 then
			enemy = entry
			break
		end
	end
	if not enemy then
		target = nil
		-- Two attempts between fights prevent protected drops from monopolizing bots.
		local drop
		for _, entry in ipairs(goro.items()) do
			if inside(entry) and entry.distance <= 5
				and (not drop or entry.distance < drop.distance) then
				drop = entry
			end
		end
		if drop and loot_attempts < 2 then
			if goro.loot(drop.id) then pickups = pickups + 1 end
			loot_attempts = loot_attempts + 1
			next_action = ticks + 7
			return
		end

		local best_score = math.huge
		for _, entry in ipairs(enemies) do
			-- Poporings resist the Thief/Rogue's poison attack.
			if inside(entry) and not ((role.name == "Thief" or role.name == "Rogue") and entry.job == 1031) then
				local score = entry.distance + math.random() * 5
				if score < best_score then enemy, best_score = entry, score end
			end
		end
		if enemy then
			target, target_since = enemy.id, ticks
			loot_attempts = 0
		end
	end

	if enemy then
		local normal_attack = not role.ranged or role.name == "Archer" or role.name == "Hunter"
		if ticks >= next_skill and sp > 0.15 then
			local skill = skills[skill_index]
			-- Magnum Break is centered on the caster, so approach before casting it.
			if skill == "SM_MAGNUM" and enemy.distance > 2 then
				goro.walk(enemy.x, enemy.y)
				next_action = ticks + 4
				return
			end
			skill_index = skill_index % #skills + 1
			if cast(skill == "SM_MAGNUM" and player.id or enemy.id, skill) then
				-- Leave room for a normal attack between skills on weapon users.
				next_skill = next_action + (normal_attack and math.random(3, 6) or 0)
				return
			end
			next_skill = ticks + 10
		end
		if normal_attack then
			if goro.attack(enemy.id) then attacks = attacks + 1 end
			next_action = ticks + 10
		else
			-- Ranged classes stay at skill range instead of chasing into melee.
			next_action = ticks + 1
		end
	else
		goro.walk(center_x + math.random(-6, 6), center_y + math.random(-6, 6))
		next_action = ticks + math.random(15, 25)
	end
end
