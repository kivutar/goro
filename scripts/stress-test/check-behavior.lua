local roster = dofile('scripts/stress-test/roster.lua')
local original_getenv, original_print = os.getenv, print
for index, role in ipairs(roster) do
 os.getenv = function(key) if key=='GORO_STRESS_INDEX' then return tostring(index) end return original_getenv(key) end
 print = function() end
 local player = {id=100, hp=1000,max_hp=1000,sp=1000,max_sp=1000,x=216,y=255,dead=false}
 local injured = {id=200,hp=100,max_hp=1000,x=217,y=255,distance=1,party_member=true,dead=false}
 local outsider = {id=201,hp=1,max_hp=1000,x=217,y=255,distance=1,party_member=false,dead=false}
 local dead = {id=202,hp=0,max_hp=1000,x=217,y=255,distance=1,party_member=true,dead=true}
 local seen = {}
 local buffs = {}
 for _,buff in ipairs(role.buffs) do buffs[buff.skill]=true end
 local attacks = {}
 for _,skill in ipairs(role.skills) do attacks[skill]=true end
 goro = {
  player=function() return player end,
  players=function() return {outsider,dead,injured} end,
  enemies=function() return {{id=300,x=217,y=255,distance=1,job=1013}} end,
  items=function() return {{id=400,x=217,y=255,distance=1}} end,
  inventory=function() return {} end,
  skill=function(id,skill)
   assert(id~=201 and id~=202, 'targeted outsider or dead party member: '..skill)
   if buffs[skill] or skill=='SM_MAGNUM' then assert(id==100, 'self skill target')
   elseif attacks[skill] then assert(id==300, 'attack target')
   elseif skill==role.healing then assert(id==200, 'heal must select injured party member');injured.hp=1000
   else assert(id==100 or id==200, 'party support target') end
   seen[skill]=true
   return true
  end,
  attack=function(id) assert(id==300);return true end,
  loot=function(id) assert(id==400);return true end,
  walk=function() return true end,
  stop=function() return true end,
  message=function() return true end,
  revive=function() error('unexpected revive') end,
 }
 dofile('scripts/stress-test/combat.lua')
 for _=1,2400 do tick() end
 for _,skill in ipairs(role.skills) do assert(seen[skill], role.name..' never used '..skill) end
 for _,skill in ipairs(role.support) do assert(seen[skill], role.name..' never used '..skill) end
 for _,buff in ipairs(role.buffs) do assert(seen[buff.skill], role.name..' never used '..buff.skill) end
 if role.healing~='' then assert(seen[role.healing], role.name..' never healed') end
 original_print(role.name..': attack rotation, self buffs and party support passed')
end
os.getenv,print=original_getenv,original_print
