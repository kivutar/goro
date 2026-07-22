  Core Gameplay / UI

  - [x] Skill targeting: 0x007E UseSkillToPosMoreInfo
  - [x] Character deletion: 0x01FB CH_DELETE_CHAR for rAthena 20080910, with 0x006F/0x0070 replies
  - [x] Remove cart option: 0x012A RemoveOption
  - [x] Quit game: 0x018A QuitGame
  - [x] Less effects toggle: 0x021D LessEffect
  - [x] Hotkey save: 0x02BA Hotkey

  NPC Dialogs

  - [x] Number input: 0x0143 NpcAmountInput
  - [x] String input: 0x01D5 NpcStringInput

  Chat / Social

  - [x] Ignore whisper from player: 0x00CF PMIgnore
  - [x] Ignore all whispers: 0x00D0 PMIgnoreAll
  - [x] Create chat room: 0x00D5 CreateChatRoom
  - [x] Whisper send/receive: 0x0096, 0x0097
  - [x] Friend list/invite/reply/remove: 0x0201, 0x0202, 0x0203, 0x0207, 0x0208, 0x0209, 0x020A
  - [x] Player emotes: 0x00BF, 0x00C0
  - [ ] Battle chat: 0x02DB BattleChat

  Party

  - [x] Create/invite/reply/leave/settings/chat: 0x00F9, 0x00FC, 0x00FF, 0x0100, 0x0102, 0x0108
  - [x] Newer invite/reply packets: 0x02C4, 0x02C7
  - [x] Party invite accept/refuse config: 0x02C8, 0x02C9
  - [ ] Party booking register: 0x0802
  - [ ] Party booking delete: 0x0806

  Guild

  - [ ] Change member position info: 0x0161
  - [x] Create guild/result: 0x0165, 0x0167
  - [x] Invite/reply to guild: 0x0168, 0x0169, 0x016A, 0x016B
  - [x] Change guild notice: 0x016E
  - [ ] Guild alliance request/reply/delete/opposition: 0x0170, 0x0172, 0x0180, 0x0183
  - [ ] Guild message: 0x017E

  Items / Crafting / Equipment

  - [x] Cart/body/storage transfers: 0x0126, 0x0127, 0x0128, 0x0129
  - [x] Card composition list/insert: 0x017A, 0x017C
  - [x] Arrow Crafting material list/selection: 0x01AD, 0x01AE
  - [x] Show equipment/view equipment: 0x02D6, 0x02D7, 0x02D8
  - [ ] Blacksmith/alchemist crafting: 0x018E
  - [ ] Item repair: 0x01FD
  - [ ] Weapon refine: 0x0222
  - [ ] Cooking: 0x025B
  - [ ] Storage password: 0x023B
  - [ ] Cash shop NPC buy: 0x0288

  Rankings / PvP

  - [ ] PvP info: 0x020F
  - [ ] Blacksmith rank: 0x0217
  - [ ] Alchemist rank: 0x0218
  - [ ] Taekwon rank: 0x0225
  - [ ] Killer rank: 0x0237

  Pets

  - [x] Catch pet: 0x019E, 0x019F, 0x01A0
  - [x] Pet menu / status / feed / performance / back to egg / accessory: 0x01A1, 0x01A2, 0x01A3, 0x01A4
  - [x] Pet status window and rename: 0x01A5
  - [x] Select egg UI / hatch pet: 0x01A6, 0x01A7
  - [x] Pet emotion and pettalktable-backed talk: 0x01A9, 0x01AA
  - [x] Feeding emotion reactions using roBrowser's pet emotion table
  - [x] Familiarity-gated client-side talk triggers for feeding, hunting, danger, death, and level-up

  Homunculus / Mercenary

  - [x] Homunculus right-click menu, status window, feed/delete confirmations, and delete cleanup: 0x022D
  - [x] Change homunculus name from the status window: 0x0231
  - [x] Homunculus HP/SP/hunger bars, name placement, level-up effect, and low-hunger color parity
  - [x] Homunculus/mercenary move/attack/return to master: 0x0232, 0x0233, 0x0234
  - [x] Mercenary action: 0x029F
  - [x] Homunculus/mercenary property, parameter, and skill-list state: 0x022E, 0x022F, 0x0230, 0x0235, 0x0239, 0x027D, 0x029B, 0x029C, 0x029D, 0x029E, 0x02A2
  - [x] Compatibility parsers for newer homunculus property/param packets: 0x07DB, 0x09F7, 0x0B2F, 0x0B76, 0x0BA4, 0x0BA5
  - [x] Homunculus skill window, staged skill upgrades with confirm, and shortcut drag/use
  - [x] Mercenary right-click/status/skill windows, humanoid display, skill shortcuts, attack sounds, and archer projectiles
  - [x] Gravity-style AI.lua / AI_M.lua loader and Lua API shim
  - [x] rAthena compatibility note: 2008 clients need full 0x022E homunculus info refreshes instead of 0x07DB param-change packets

  Mail / Auction

  - [ ] Mail refresh/read/delete/get attachment/open/set attachment/send/return: 0x023F, 0x0241, 0x0243, 0x0244, 0x0246, 0x0247, 0x0248, 0x0273
  - [ ] Auction cancel registration/set item/register/cancel/bid/search/buy-sell/close: 0x024B, 0x024C, 0x024D, 0x024E, 0x024F, 0x0251, 0x025C, 0x025D

  Adoption / Family

  - [ ] Adoption request/reply: 0x01F9, 0x01F7

  Quest / Instances

  - [ ] Quest state ack: 0x02B6
  - [ ] Memorial dungeon command: 0x02CF
  - [ ] Progress bar ack/cancel: 0x02F1 progressbar

  Class-Specific / Skill Dialog Choices

  - [ ] Sage autospell selection: 0x01CE
  - [ ] Novice dori-dori: 0x01E7
  - [ ] Novice explosion spirits: 0x01ED
  - [ ] Star Gladiator feel save confirmation: 0x0254
  - [ ] Auto-revive: 0x0292

  GM / Admin

  - [ ] GM kick/kick all: 0x00CC, 0x00CE
  - [ ] GM item/monster: 0x013F
  - [ ] GM move map: 0x0140
  - [ ] GM no-chat: 0x0149
  - [ ] GM change map type: 0x0198
  - [ ] GM shift/recall variants: 0x01BA, 0x01BB, 0x01BC, 0x01BD
  - [ ] GM request account name: 0x01DF
  - [ ] GM remote command/check: 0x0212, 0x0213
