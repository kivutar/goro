# Homunculus And Mercenary Support

Target client date: `20080910`.

Goro implements homunculus and mercenary companions as one companion feature with
separate session state, UI windows, AI runners, and packet paths. The behavior is
kept close to roBrowser where possible, with 2008 packetver compatibility kept
explicit because modern rAthena sends newer homunculus refresh packets by
default.

## Runtime Features

Homunculus support includes:

- right-click menu with info, feed, delete, and delete cleanup
- status window with name editing, HP/SP/EXP/hunger bars, stats table, and
  low-hunger color handling
- display HP/SP/hunger bars and name under the sprite
- level-up visual effect
- movement, attack, return-to-master, and skill-use commands
- skill window with staged skill upgrades, confirm button, drag-to-shortcut, and
  shortcut skill use
- Gravity-style `AI/AI.lua` default AI and `AI/USER_AI/AI.lua` custom AI toggled
  by `/hoai`

Mercenary support includes:

- right-click menu with info and skill windows
- status window with HP/SP/time/kills bars and combat stats
- display HP/SP bars under the sprite
- humanoid sprite composition with default weapon appearance for sword, spear,
  and bow mercenaries
- attack animation, weapon hit sounds, and weapon-aware skill effects
- movement, attack, return-to-master, action, and skill-use commands
- skill window using the same companion skill UI and shortcut path as
  homunculus skills
- Gravity-style `AI/AI_M.lua` default AI and `AI/USER_AI/AI_M.lua` custom AI
  toggled by `/merai`

`--force-user-ai=true` starts both companion AI runners in custom `USER_AI` mode.
Without that flag, Goro starts in default AI mode and switches only when `/hoai`
or `/merai` is used.

## Packet Coverage

Implemented 2008 companion packet paths:

- `0x022D`: homunculus command/menu actions
- `0x022E`: full homunculus property packet
- `0x022F`: homunculus skill list
- `0x0230`: homunculus skill update
- `0x0231`: homunculus rename
- `0x0232`: companion move
- `0x0233`: companion attack
- `0x0234`: companion return to master
- `0x0235`: homunculus food state
- `0x0239`: homunculus state
- `0x027D`: homunculus skill-up request
- `0x029B`: mercenary property packet
- `0x029C`: mercenary skill list
- `0x029D`: mercenary skill update
- `0x029E`: mercenary parameter update
- `0x029F`: mercenary action
- `0x02A2`: mercenary delete/state update

Goro also parses newer homunculus property and parameter packets (`0x07DB`,
`0x09F7`, `0x0B2F`, `0x0B76`, `0x0BA4`, `0x0BA5`) so nearby or patched servers
do not silently lose companion state.

## rAthena Compatibility

For `PACKETVER=20080910`, official-style clients use the full
`ZC_PROPERTY_HOMUN` packet (`0x022E`) for homunculus status refreshes.
`ZC_HO_PAR_CHANGE` (`0x07DB`) starts at `PACKETVER >= 20090610`, so a modern
rAthena tree must be patched to send a full `0x022E` refresh for old clients
when homunculus HP, SP, or EXP changes.

The required patch is documented in [rAthena setup](rathena-setup.md). Without
it, the homunculus status window can show stale HP/SP/EXP after the initial info
packet.

## Visual And Data Notes

Mercenary archers use roBrowser's monster attack projectile mapping rather than
player weapon-view detection. Goro imports that data in
`db.MonsterAttackProjectile`; currently only the implemented
`ef_arrow_projectile` primitive is rendered as the normal arrow shot effect.
Other imported projectile names are intentionally ignored until their effect
primitives exist.

Mercenary sword, spear, and bow appearance is derived client-side when the server
does not send an explicit weapon view id. This matches the 2008 mercenary jobs
well enough for rendering and keeps server packets compatible with the older
packetver.

## Current Limits

The feature has automated coverage for packet parsing/building, companion UI
state, AI selection, skill shortcuts, mercenary skill targeting, attack sounds,
and bow projectiles. Manual coverage has focused on Vanilmirth and the common
sword, spear, and bow mercenary scrolls. Less common custom AI scripts and every
individual mercenary/homunculus skill visual have not been exhaustively checked.
