# 2008 Packet Coverage

Target client date: `20080910`.

Sources:
- Goro-compatible rAthena `src/map/clif_packetdb.hpp`, preprocessed with
  `PACKETVER=20080910` and `PACKETVER_SAK_NUM=20080910`.
- rAthena `src/map/packets.hpp`, `src/map/packets_struct.hpp`, and `src/common/packets.hpp` for header aliases.
- Goro `network/*.go` for currently referenced packet IDs, excluding tests.

This is the final effective rAthena map packet table. rAthena defines historical remaps by calling `packetdb_addpacket` more than once for the same opcode; the last definition wins, so this document records the winning definition per opcode.

Status meaning:
- `implemented`: the opcode is referenced by Goro network code and wired to an active client parser/builder flow.
- `referenced`: the opcode appears in Goro network code. This is opcode-level coverage, not a promise that every field/layout variant is complete.
- `missing`: rAthena accepts this client packet for 20080910 and Goro does not reference its opcode yet.
- `untracked`: server-to-client packet in rAthena packet DB that Goro does not reference yet; many are optional until the feature exists.

## Summary

- Raw rAthena map packet declarations after preprocessing: `886`
- Effective unique map opcodes: `603`
- Overwritten historical/remap declarations: `283`
- Client-to-map packets accepted by rAthena: `177`
- Effective map opcodes referenced by Goro: `232`
- Client-to-map accepted packets referenced by Goro: `104` / `177`
- Unresolved packet aliases in this generated pass: `0`

## Homunculus Compatibility Notes

For `20080910`, homunculus status refreshes use the full `ZC_PROPERTY_HOMUN`
packet (`0x022E`). `ZC_HO_PAR_CHANGE` (`0x07DB`) is a later packet, starting at
`PACKETVER >= 20090610`, so it is not part of the effective 2008 rAthena map
packet table. Goro still parses the later homunculus property and param-change
variants so it can interoperate with nearby or patched servers.

Current upstream rAthena and the Goro-compatible rAthena branch make
`clif_homunculus_updatestatus` send a full `0x022E` refresh for
`PACKETVER < 20090610`. Older server checkouts without that fallback can display
stale HP, SP, or EXP values after the initial homunculus info packet.

`ZC_NOTIFY_EXP` (`0x07F6`) is the later client-side EXP gain/loss notification
packet. eAthena and Hercules both gate `clif_displayexp()` behind
`PACKETVER >= 20091027`, and Hercules' 2008 Sakray packet table does not include
`0x07F6`. Current upstream rAthena keeps the same behavior by not sending that
packet to the 2008 Sakray profile; `@showexp` remains the server text-message
path for green EXP percentage lines.

See [homunculus and mercenary support](companions-20080910.md) for the current
companion UI, AI, skill, and visual coverage.

| Opcode | Direction | Status | Scope | Goro refs |
|---:|---|---|---|---|
| `0x022E` | S->C | implemented | 2008 full homunculus property and legacy refresh path | companion_packets.go, packet.go |
| `0x07DB` | S->C | compatibility parser | 2009+ incremental homunculus param change | companion_packets.go, packet.go |
| `0x09F7` | S->C | compatibility parser | later homunculus property layout with 32-bit HP | companion_packets.go, packet.go |
| `0x0B2F` | S->C | compatibility parser | later homunculus property layout without equip item id | companion_packets.go, packet.go |
| `0x0B76` | S->C | compatibility parser | later homunculus property layout with 32-bit HP/SP | companion_packets.go, packet.go |
| `0x0BA4` | S->C | compatibility parser | later homunculus property layout with 64-bit EXP | companion_packets.go, packet.go |
| `0x0BA5` | S->C | compatibility parser | later 64-bit homunculus param change | companion_packets.go, packet.go |

## Server-Driven Display Notes

Goro handles the 2008 ShowDigit clock (`0x01B1`), Convex Mirror boss report
(`0x0293`), and NPC progress lifecycle (`0x02F0`-`0x02F2`). It also handles
the fixed Gospel/Full Strip skill notice (`0x0215`); rAthena sends that packet
to 2008 clients even though it is not declared in the effective map parser
table used to generate the full table below.

## High Priority Gaps

| Opcode | Direction | rAthena symbol | Length | Handler | Priority |
|---:|---|---|---:|---|---|
| `0x0149` | C->S | `0x0149` | `9` | `clif_parse_GMReqNoChat` | P2 |
| `0x0170` | C->S | `0x0170` | `14` | `clif_parse_GuildRequestAlliance` | P2 |
| `0x0172` | C->S | `0x0172` | `10` | `clif_parse_GuildReplyAlliance` | P2 |
| `0x017E` | C->S | `0x017e` | `-1` | `clif_parse_GuildMessage` | P2 |
| `0x0180` | C->S | `0x0180` | `6` | `clif_parse_GuildOpposition` | P2 |
| `0x0183` | C->S | `0x0183` | `10` | `clif_parse_GuildDelAlliance` | P2 |
| `0x02CF` | C->S | `0x02cf` | `6` | `clif_parse_MemorialDungeonCommand` | P2 |
| `0x02DB` | C->S | `0x02db` | `-1` | `clif_parse_BattleChat` | P2 |
| `0x0802` | C->S | `0x0802` | `18` | `clif_parse_PartyBookingRegisterReq` | P2 |
| `0x0806` | C->S | `0x0806` | `4` | `clif_parse_PartyBookingDeleteReq` | P2 |

## Login And Char Server Packets

This section is from rAthena common packet headers. It is not a parser DB, but it is useful for login/char coverage.

For rAthena `PACKETVER=20080910`, character deletion uses `0x01FB`
(`CH_DELETE_CHAR`) with a 50-byte email/key field. The older `0x0068` shape is
kept only for pre-20040419 compatibility. When testing this against local
rAthena, `conf/char_athena.conf` must use `char_del_delay: 0`; delayed deletion
is a 2010-08-03+ client flow and direct 2008 deletion otherwise fails after the
email check.

| Opcode | Header | Goro refs |
|---:|---|---|
| `0x0064` | `CA_LOGIN` | login_packets.go |
| `0x0066` | `CH_SELECT_CHAR` | login_packets.go |
| `0x0067` | `CH_MAKE_CHAR` | login_packets.go |
| `0x0068` | `CH_DELETE_CHAR` | login_packets.go (pre-20040419 only) |
| `0x0069` | `AC_ACCEPT_LOGIN` | login_responses.go, packet.go |
| `0x006A` | `AC_REFUSE_LOGIN` | packet.go |
| `0x006B` | `HC_ACCEPT_ENTER` | login_responses.go, packet.go |
| `0x006C` | `HC_REFUSE_ENTER` | packet.go |
| `0x006D` | `HC_ACCEPT_MAKECHAR` | login_responses.go, packet.go |
| `0x006E` | `HC_REFUSE_MAKECHAR` | login_responses.go, packet.go |
| `0x006F` | `HC_ACCEPT_DELETECHAR` | login_responses.go, packet.go |
| `0x0070` | `HC_REFUSE_DELETECHAR` | login_responses.go, packet.go |
| `0x0071` | `HC_NOTIFY_ZONESVR` | login_responses.go, packet.go |
| `0x0081` | `SC_NOTIFY_BAN` | disconnect_packets.go, packet.go |
| `0x0187` | `PING` | packet.go |
| `0x01DB` | `CA_REQ_HASH` | - |
| `0x01DC` | `AC_ACK_HASH` | packet.go |
| `0x01DD` | `CA_LOGIN2` | - |
| `0x01FA` | `CA_LOGIN3` | - |
| `0x01FB` | `CH_DELETE_CHAR` | login_packets.go |
| `0x0200` | `CA_CONNECT_INFO_CHANGED` | - |
| `0x0204` | `CA_EXE_HASHCHECK` | - |
| `0x020D` | `HC_BLOCK_CHARACTER` | - |
| `0x0277` | `CA_LOGIN_PCBANG` | - |
| `0x027C` | `CA_LOGIN4` | - |
| `0x028D` | `CH_REQ_IS_VALID_CHARNAME` | - |
| `0x028E` | `HC_ACK_IS_VALID_CHARNAME` | - |
| `0x028F` | `CH_REQ_CHANGE_CHARNAME` | - |
| `0x0290` | `HC_ACK_CHANGE_CHARNAME` | - |
| `0x02B0` | `CA_LOGIN_CHANNEL` | packet.go |
| `0x0825` | `CA_SSO_LOGIN_REQ` | - |
| `0x0827` | `CH_DELETE_CHAR3_RESERVED` | - |
| `0x0828` | `HC_DELETE_CHAR3_RESERVED` | - |
| `0x0829` | `CH_DELETE_CHAR3` | - |
| `0x082A` | `HC_DELETE_CHAR3` | - |
| `0x082B` | `CH_DELETE_CHAR3_CANCEL` | - |
| `0x082C` | `HC_DELETE_CHAR3_CANCEL` | - |
| `0x082D` | `HC_ACCEPT_ENTER2` | - |
| `0x083E` | `AC_REFUSE_LOGIN` | - |
| `0x0840` | `HC_NOTIFY_ACCESSIBLE_MAPNAME` | - |
| `0x0841` | `CH_SELECT_ACCESSIBLE_MAPNAME` | - |
| `0x08B8` | `CH_SECOND_PASSWD_ACK` | - |
| `0x08B9` | `HC_SECOND_PASSWD_LOGIN` | - |
| `0x08BA` | `CH_MAKE_SECOND_PASSWD` | - |
| `0x08BE` | `CH_EDIT_SECOND_PASSWD` | - |
| `0x08C5` | `CH_AVAILABLE_SECOND_PASSWD` | - |
| `0x08D4` | `CH_REQ_CHANGE_CHARACTER_SLOT` | - |
| `0x08D5` | `HC_ACK_CHANGE_CHARACTER_SLOT` | - |
| `0x08FC` | `CH_REQ_CHANGE_CHARNAME` | - |
| `0x08FD` | `HC_ACK_CHANGE_CHARNAME` | - |
| `0x0970` | `CH_MAKE_CHAR` | - |
| `0x099D` | `HC_ACK_CHARINFO_PER_PAGE` | - |
| `0x09A0` | `HC_CHARLIST_NOTIFY` | - |
| `0x09A1` | `CH_CHARLIST_REQ` | - |
| `0x0A39` | `CH_MAKE_CHAR` | - |
| `0x0AC4` | `AC_ACCEPT_LOGIN` | - |
| `0x0AC5` | `HC_NOTIFY_ZONESVR` | - |
| `0x0B6F` | `HC_ACCEPT_MAKECHAR` | - |
| `0x0B70` | `HC_ACK_CHANGE_CHARACTER_SLOT` | - |
| `0x0B72` | `HC_ACK_CHARINFO_PER_PAGE` | - |

## Full Effective 20080910 Map Packet DB

| Opcode | Direction | Status | Symbol | Length | rAthena handler | Goro refs |
|---:|---|---|---|---:|---|---|
| `0x0064` | S->C | referenced | `0x0064` | `55` | `-` | login_packets.go |
| `0x0065` | S->C | referenced | `0x0065` | `17` | `-` | login_packets.go |
| `0x0066` | S->C | referenced | `0x0066` | `3` | `-` | login_packets.go |
| `0x0067` | S->C | referenced | `0x0067` | `37` | `-` | login_packets.go |
| `0x0068` | S->C | untracked | `0x0068` | `46` | `-` | - |
| `0x0069` | S->C | referenced | `0x0069` | `-1` | `-` | login_responses.go, packet.go |
| `0x006A` | S->C | referenced | `0x006a` | `23` | `-` | packet.go |
| `0x006B` | S->C | referenced | `0x006b` | `-1` | `-` | login_responses.go, packet.go |
| `0x006C` | S->C | referenced | `0x006c` | `3` | `-` | packet.go |
| `0x006D` | S->C | referenced | `0x006d` | `110` | `-` | login_responses.go, packet.go |
| `0x006E` | S->C | referenced | `0x006e` | `3` | `-` | login_responses.go, packet.go |
| `0x006F` | S->C | referenced | `0x006f` | `2` | `-` | packet.go |
| `0x0070` | S->C | referenced | `0x0070` | `3` | `-` | packet.go |
| `0x0071` | S->C | referenced | `0x0071` | `28` | `-` | login_responses.go, packet.go |
| `0x0072` | C->S | alias-covered | `0x0072` | `25` | `clif_parse_UseSkillToId` | - |
| `0x0075` | S->C | untracked | `0x0075` | `-1` | `-` | - |
| `0x0076` | S->C | untracked | `0x0076` | `9` | `-` | - |
| `0x0077` | S->C | referenced | `0x0077` | `5` | `-` | packet.go |
| `0x0079` | S->C | referenced | `0x0079` | `53` | `-` | actor_packets.go, packet.go |
| `0x007A` | S->C | referenced | `0x007a` | `58` | `-` | actor_packets.go, packet.go |
| `0x007B` | S->C | referenced | `0x007b` | `60` | `-` | actor_packets.go, packet.go |
| `0x007C` | S->C | referenced | `0x007c` | `42` | `-` | actor_packets.go, packet.go |
| `0x007D` | C->S | referenced | `0x007d` | `2` | `clif_parse_LoadEndAck` | login_packets.go |
| `0x007E` | C->S | implemented | `0x007e` | `102` | `clif_parse_UseSkillToPosMoreInfo` | skill_packets.go |
| `0x0082` | S->C | untracked | `0x0082` | `2` | `-` | - |
| `0x0083` | S->C | untracked | `0x0083` | `2` | `-` | - |
| `0x0084` | S->C | untracked | `0x0084` | `2` | `-` | - |
| `0x0085` | C->S | referenced | `0x0085` | `11` | `clif_parse_ChangeDir` | login_packets.go |
| `0x0089` | C->S | referenced | `0x0089` | `8` | `clif_parse_TickSend` | login_packets.go |
| `0x008B` | S->C | referenced | `0x008b` | `2` | `-` | packet.go |
| `0x008C` | C->S | referenced | `0x008c` | `11` | `clif_parse_GetCharNameRequest` | chat_packets.go, login_packets.go |
| `0x008D` | S->C | referenced | `0x008d` | `-1` | `-` | chat_packets.go, packet.go |
| `0x008E` | S->C | referenced | `0x008e` | `-1` | `-` | chat_packets.go, packet.go |
| `0x0090` | C->S | referenced | `HEADER_CZ_CONTACTNPC` | `sizeof( PACKET_CZ_CONTACTNPC )` | `clif_parse_NpcClicked` | npc_packets.go |
| `0x0093` | S->C | untracked | `0x0093` | `2` | `-` | - |
| `0x0094` | C->S | referenced | `0x0094` | `14` | `clif_parse_MoveToKafra` | item_packets.go, login_packets.go |
| `0x0096` | C->S | referenced | `0x0096` | `-1` | `clif_parse_WisMessage` | chat_packets.go |
| `0x0099` | C->S | referenced | `HEADER_CZ_BROADCAST` | `-1` | `clif_parse_Broadcast` | packet.go |
| `0x009B` | C->S | referenced | `0x009b` | `26` | `clif_parse_WantToConnection` | login_packets.go, packet.go |
| `0x009E` | S->C | referenced | `0x009e` | `17` | `-` | item_packets.go, packet.go |
| `0x009F` | C->S | referenced | `0x009f` | `14` | `clif_parse_UseItem` | item_packets.go |
| `0x00A2` | C->S | referenced | `0x00a2` | `15` | `clif_parse_SolveCharName` | item_packets.go |
| `0x00A7` | C->S | referenced | `0x00a7` | `8` | `clif_parse_WalkToXY` | item_packets.go, login_packets.go, packet.go |
| `0x00A8` | S->C | referenced | `useItemAckType` | `sizeof( struct PACKET_ZC_USE_ITEM_ACK )` | `-` | item_packets.go, packet.go |
| `0x00A9` | C->S | referenced | `HEADER_CZ_REQ_WEAR_EQUIP` | `sizeof( PACKET_CZ_REQ_WEAR_EQUIP )` | `clif_parse_EquipItem` | item_packets.go, packet.go |
| `0x00AB` | C->S | referenced | `0x00ab` | `4` | `clif_parse_UnequipItem` | item_packets.go |
| `0x00AE` | S->C | untracked | `0x00ae` | `-1` | `-` | - |
| `0x00B2` | C->S | referenced | `0x00b2` | `3` | `clif_parse_Restart` | login_packets.go, packet.go |
| `0x00B8` | C->S | referenced | `0x00b8` | `7` | `clif_parse_NpcSelectMenu` | npc_packets.go |
| `0x00B9` | C->S | referenced | `0x00b9` | `6` | `clif_parse_NpcNextClicked` | npc_packets.go |
| `0x00BA` | S->C | referenced | `0x00ba` | `2` | `-` | packet.go |
| `0x00BB` | C->S | referenced | `0x00bb` | `5` | `clif_parse_StatusUp` | packet.go, status_packets.go |
| `0x00BF` | C->S | referenced | `HEADER_CZ_REQ_EMOTION` | `sizeof( PACKET_CZ_REQ_EMOTION )` | `clif_parse_Emotion` (player emote) | emotion_packets.go, packet.go |
| `0x00C1` | C->S | referenced | `0x00c1` | `2` | `clif_parse_HowManyConnections` | packet.go |
| `0x00C3` | S->C | referenced | `0x00c3` | `8` | `-` | actor_packets.go, packet.go |
| `0x00C5` | C->S | referenced | `HEADER_CZ_ACK_SELECT_DEALTYPE` | `sizeof( PACKET_CZ_ACK_SELECT_DEALTYPE )` | `clif_parse_NpcBuySellSelected` | item_packets.go, packet.go |
| `0x00C6` | S->C | referenced | `0x00c6` | `-1` | `-` | item_packets.go, packet.go |
| `0x00C8` | C->S | referenced | `0x00c8` | `-1` | `clif_parse_NpcBuyListSend` | item_packets.go, packet.go |
| `0x00C9` | C->S | referenced | `HEADER_CZ_PC_SELL_ITEMLIST` | `-1` | `clif_parse_NpcSellListSend` | item_packets.go, packet.go |
| `0x00CA` | S->C | referenced | `0x00ca` | `3` | `-` | item_packets.go, packet.go |
| `0x00CB` | S->C | referenced | `0x00cb` | `3` | `-` | item_packets.go, packet.go |
| `0x00CC` | C->S | missing | `0x00cc` | `6` | `clif_parse_GMKick` | - |
| `0x00CD` | S->C | untracked | `0x00cd` | `3` | `-` | - |
| `0x00CE` | C->S | missing | `0x00ce` | `2` | `clif_parse_GMKickAll` | - |
| `0x00CF` | C->S | implemented | `0x00cf` | `27` | `clif_parse_PMIgnore` | chat_packets.go |
| `0x00D0` | C->S | implemented | `HEADER_CZ_SETTING_WHISPER_STATE` | `sizeof( PACKET_CZ_SETTING_WHISPER_STATE )` | `clif_parse_PMIgnoreAll` | chat_packets.go |
| `0x00D3` | C->S | referenced | `0x00d3` | `2` | `clif_parse_PMIgnoreList` | packet.go |
| `0x00D5` | C->S | implemented | `HEADER_CZ_CREATE_CHATROOM` | `-1` | `clif_parse_CreateChatRoom` | chat_packets.go |
| `0x00D9` | C->S | implemented | `HEADER_CZ_REQ_ENTER_ROOM` | `sizeof( PACKET_CZ_REQ_ENTER_ROOM )` | `clif_parse_ChatAddMember` | chat_packets.go |
| `0x00DE` | C->S | referenced | `HEADER_CZ_CHANGE_CHATROOM` | `-1` | `clif_parse_ChatRoomStatusChange` | packet.go |
| `0x00E0` | C->S | referenced | `0x00e0` | `30` | `clif_parse_ChangeChatOwner` | packet.go |
| `0x00E2` | C->S | referenced | `0x00e2` | `26` | `clif_parse_KickFromChat` | packet.go |
| `0x00E3` | C->S | implemented | `0x00e3` | `2` | `clif_parse_ChatLeave` | chat_packets.go |
| `0x00E4` | C->S | referenced | `0x00e4` | `6` | `clif_parse_TradeRequest` | trade_packets.go |
| `0x00E5` | S->C | referenced | `0x00e5` | `26` | `-` | trade_packets.go |
| `0x00E6` | C->S | referenced | `0x00e6` | `3` | `clif_parse_TradeAck` | trade_packets.go |
| `0x00E8` | C->S | referenced | `HEADER_CZ_ADD_EXCHANGE_ITEM` | `sizeof( PACKET_CZ_ADD_EXCHANGE_ITEM )` | `clif_parse_TradeAddItem` | trade_packets.go |
| `0x00EA` | S->C | referenced | `0x00ea` | `5` | `-` | trade_packets.go |
| `0x00EB` | C->S | referenced | `0x00eb` | `2` | `clif_parse_TradeOk` | trade_packets.go |
| `0x00ED` | C->S | referenced | `0x00ed` | `2` | `clif_parse_TradeCancel` | trade_packets.go |
| `0x00EF` | C->S | referenced | `0x00ef` | `2` | `clif_parse_TradeCommit` | trade_packets.go |
| `0x00F3` | C->S | referenced | `0x00f3` | `-1` | `clif_parse_GlobalMessage` | chat_packets.go, item_packets.go, login_packets.go, packet.go |
| `0x00F5` | C->S | referenced | `0x00f5` | `8` | `clif_parse_TakeItem` | item_packets.go, packet.go |
| `0x00F7` | C->S | referenced | `0x00f7` | `22` | `clif_parse_MoveFromKafra` | item_packets.go, packet.go |
| `0x00F9` | C->S | referenced | `HEADER_CZ_MAKE_GROUP` | `sizeof( PACKET_CZ_MAKE_GROUP )` | `clif_parse_CreateParty` | party_packets.go |
| `0x00FB` | S->C | referenced | `0x00fb` | `-1` | `-` | party_packets.go |
| `0x00FC` | C->S | referenced | `HEADER_CZ_REQ_JOIN_GROUP` | `sizeof( PACKET_CZ_REQ_JOIN_GROUP )` | `clif_parse_PartyInvite` | party_packets.go |
| `0x00FD` | S->C | referenced | `0x00fd` | `27` | `-` | party_packets.go |
| `0x00FF` | C->S | referenced | `HEADER_CZ_JOIN_GROUP` | `sizeof( PACKET_CZ_JOIN_GROUP )` | `clif_parse_ReplyPartyInvite` | party_packets.go |
| `0x0100` | C->S | referenced | `HEADER_CZ_REQ_LEAVE_GROUP` | `sizeof( PACKET_CZ_REQ_LEAVE_GROUP )` | `clif_parse_LeaveParty` | party_packets.go |
| `0x0101` | S->C | referenced | `0x0101` | `6` | `-` | party_packets.go |
| `0x0102` | C->S | referenced | `0x0102` | `6` | `clif_parse_PartyChangeOption` | party_packets.go |
| `0x0103` | C->S | implemented | `HEADER_CZ_REQ_EXPEL_GROUP_MEMBER` | `sizeof( PACKET_CZ_REQ_EXPEL_GROUP_MEMBER )` | `clif_parse_RemovePartyMember` | party_packets.go |
| `0x0104` | S->C | referenced | `0x0104` | `79` | `-` | party_packets.go |
| `0x0108` | C->S | referenced | `0x0108` | `-1` | `clif_parse_PartyMessage` | party_packets.go |
| `0x0109` | S->C | referenced | `0x0109` | `-1` | `-` | party_packets.go |
| `0x0112` | C->S | referenced | `0x0112` | `4` | `clif_parse_SkillUp` | skill_packets.go |
| `0x0113` | C->S | referenced | `0x0113` | `22` | `clif_parse_UseSkillToPos` | item_packets.go, packet.go, skill_packets.go |
| `0x0114` | S->C | referenced | `0x0114` | `31` | `-` | actor_packets.go, packet.go |
| `0x0115` | S->C | referenced | `0x0115` | `35` | `-` | packet.go |
| `0x0116` | C->S | referenced | `0x0116` | `10` | `clif_parse_DropItem` | item_packets.go, packet.go, skill_packets.go |
| `0x0118` | C->S | referenced | `0x0118` | `2` | `clif_parse_StopAttack` | packet.go |
| `0x0119` | S->C | referenced | `0x0119` | `13` | `-` | actor_packets.go, packet.go |
| `0x011B` | C->S | referenced | `HEADER_CZ_SELECT_WARPPOINT` | `sizeof( PACKET_CZ_SELECT_WARPPOINT )` | `clif_parse_UseSkillMap` | skill_packets.go |
| `0x011D` | C->S | referenced | `0x011d` | `2` | `clif_parse_RequestMemo` | packet.go, skill_packets.go |
| `0x011F` | S->C | referenced | `0x011f` | `16` | `-` | packet.go, skill_packets.go |
| `0x0126` | C->S | referenced | `HEADER_CZ_MOVE_ITEM_FROM_BODY_TO_CART` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_BODY_TO_CART )` | `clif_parse_PutItemToCart` | item_packets.go |
| `0x0127` | C->S | referenced | `HEADER_CZ_MOVE_ITEM_FROM_CART_TO_BODY` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_CART_TO_BODY )` | `clif_parse_GetItemFromCart` | item_packets.go |
| `0x0128` | C->S | referenced | `HEADER_CZ_MOVE_ITEM_FROM_STORE_TO_CART` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_STORE_TO_CART )` | `clif_parse_MoveFromKafraToCart` | item_packets.go |
| `0x0129` | C->S | referenced | `HEADER_CZ_MOVE_ITEM_FROM_CART_TO_STORE` | `sizeof( PACKET_CZ_MOVE_ITEM_FROM_CART_TO_STORE )` | `clif_parse_MoveToKafraFromCart` | item_packets.go |
| `0x012A` | C->S | implemented | `0x012a` | `2` | `clif_parse_RemoveOption` | equipment_packets.go |
| `0x012E` | C->S | referenced | `0x012e` | `2` | `clif_parse_CloseVending` | vending_packets.go |
| `0x012F` | C->S | referenced | `0x012f` | `-1` | `clif_parse_OpenVending` | vending_packets.go |
| `0x0130` | C->S | referenced | `0x0130` | `6` | `clif_parse_VendingListReq` | vending_packets.go |
| `0x0134` | C->S | referenced | `HEADER_CZ_PC_PURCHASE_ITEMLIST_FROMMC` | `-1` | `clif_parse_PurchaseReq` | vending_packets.go |
| `0x0138` | S->C | untracked | `0x0138` | `3` | `-` | - |
| `0x013F` | C->S | missing | `0x013f` | `26` | `clif_parse_GM_Item_Monster` | - |
| `0x0140` | C->S | missing | `HEADER_CZ_MOVETO_MAP` | `sizeof( PACKET_CZ_MOVETO_MAP )` | `clif_parse_MapMove` | - |
| `0x0143` | C->S | implemented | `HEADER_CZ_INPUT_EDITDLG` | `sizeof( PACKET_CZ_INPUT_EDITDLG )` | `clif_parse_NpcAmountInput` | npc_packets.go |
| `0x0145` | S->C | referenced | `0x0145` | `19` | `-` | packet.go |
| `0x0146` | C->S | referenced | `HEADER_CZ_CLOSE_DIALOG` | `sizeof( PACKET_CZ_CLOSE_DIALOG )` | `clif_parse_NpcCloseClicked` | npc_packets.go |
| `0x0147` | S->C | referenced | `0x0147` | `39` | `-` | packet.go, skill_packets.go |
| `0x0149` | C->S | missing | `0x0149` | `9` | `clif_parse_GMReqNoChat` | - |
| `0x014A` | S->C | referenced | `0x014a` | `6` | `-` | packet.go |
| `0x014D` | C->S | implemented | `0x014d` | `2` | `clif_parse_GuildCheckMaster` | guild_packets.go |
| `0x014E` | S->C | implemented | `HEADER_ZC_ACK_GUILD_MENUINTERFACE` | `6` | `-` | guild_packets.go, packet.go |
| `0x014F` | C->S | referenced | `0x014f` | `6` | `clif_parse_GuildRequestInfo` | guild_packets.go, packet.go |
| `0x0150` | S->C | referenced | `0x0150` | `110` | `-` | guild_packets.go, packet.go |
| `0x0151` | C->S | referenced | `0x0151` | `6` | `clif_parse_GuildRequestEmblem` | guild_packets.go, packet.go |
| `0x0152` | S->C | referenced | `0x0152` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0153` | C->S | referenced | `0x0153` | `-1` | `clif_parse_GuildChangeEmblem` | guild_packets.go, packet.go |
| `0x0154` | S->C | referenced | `0x0154` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0155` | C->S | referenced | `HEADER_CZ_REQ_CHANGE_MEMBERPOS` | `-1` | `clif_parse_GuildChangeMemberPosition` | guild_packets.go, packet.go |
| `0x0157` | S->C | referenced | `0x0157` | `6` | `-` | packet.go |
| `0x0158` | S->C | referenced | `0x0158` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0159` | C->S | implemented | `HEADER_CZ_REQ_LEAVE_GUILD` | `sizeof( PACKET_CZ_REQ_LEAVE_GUILD )` | `clif_parse_GuildLeave` | guild_packets.go |
| `0x015A` | S->C | implemented | `HEADER_ZC_ACK_LEAVE_GUILD` | `66` | `-` | guild_packets.go |
| `0x015B` | C->S | implemented | `HEADER_CZ_REQ_BAN_GUILD` | `sizeof( PACKET_CZ_REQ_BAN_GUILD )` | `clif_parse_GuildExpulsion` | guild_packets.go |
| `0x015C` | S->C | implemented | `HEADER_ZC_ACK_BAN_GUILD` | `90` | `-` | guild_packets.go |
| `0x015D` | C->S | implemented | `HEADER_CZ_REQ_DISORGANIZE_GUILD` | `sizeof( PACKET_CZ_REQ_DISORGANIZE_GUILD )` | `clif_parse_GuildBreak` | guild_packets.go, console.go |
| `0x015E` | S->C | implemented | `HEADER_ZC_ACK_DISORGANIZE_GUILD_RESULT` | `6` | `-` | guild_packets.go |
| `0x015F` | S->C | referenced | `0x015f` | `42` | `-` | packet.go |
| `0x0161` | C->S | implemented | `0x0161` | `-1` | `clif_parse_GuildChangePositionInfo` | guild_packets.go |
| `0x0164` | S->C | untracked | `0x0164` | `-1` | `-` | - |
| `0x0165` | C->S | referenced | `0x0165` | `30` | `clif_parse_CreateGuild` | guild_packets.go |
| `0x0166` | S->C | referenced | `0x0166` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0167` | S->C | referenced | `0x0167` | `3` | `-` | guild_packets.go |
| `0x0168` | C->S | referenced | `HEADER_CZ_REQ_JOIN_GUILD` | `sizeof( PACKET_CZ_REQ_JOIN_GUILD )` | `clif_parse_GuildInvite` | guild_packets.go |
| `0x0169` | S->C | referenced | `0x0169` | `3` | `-` | guild_packets.go |
| `0x016A` | S->C | referenced | `0x016a` | `30` | `-` | guild_packets.go |
| `0x016B` | C->S | referenced | `HEADER_CZ_JOIN_GUILD` | `sizeof( PACKET_CZ_JOIN_GUILD )` | `clif_parse_GuildReplyInvite` | guild_packets.go |
| `0x016C` | S->C | referenced | `0x016c` | `43` | `-` | guild_packets.go, packet.go |
| `0x016E` | C->S | referenced | `0x016e` | `186` | `clif_parse_GuildChangeNotice` | guild_packets.go |
| `0x0170` | C->S | missing | `0x0170` | `14` | `clif_parse_GuildRequestAlliance` | - |
| `0x0172` | C->S | missing | `0x0172` | `10` | `clif_parse_GuildReplyAlliance` | - |
| `0x0175` | S->C | referenced | `0x0175` | `6` | `-` | packet.go |
| `0x0176` | S->C | implemented | `0x0176` | `106` | `-` | guild_packets.go, packet.go |
| `0x0177` | S->C | referenced | `0x0177` | `-1` | `-` | item_packets.go, packet.go |
| `0x0178` | C->S | referenced | `HEADER_CZ_REQ_ITEMIDENTIFY` | `sizeof( PACKET_CZ_REQ_ITEMIDENTIFY )` | `clif_parse_ItemIdentify` | item_packets.go |
| `0x017A` | C->S | referenced | `HEADER_CZ_REQ_ITEMCOMPOSITION_LIST` | `sizeof( PACKET_CZ_REQ_ITEMCOMPOSITION_LIST )` | `clif_parse_UseCard` | item_packets.go |
| `0x017B` | S->C | referenced | `0x017b` | `-1` | `-` | packet.go |
| `0x017C` | C->S | referenced | `HEADER_CZ_REQ_ITEMCOMPOSITION` | `sizeof( PACKET_CZ_REQ_ITEMCOMPOSITION )` | `clif_parse_InsertCard` | item_packets.go |
| `0x017E` | C->S | referenced | `0x017e` | `-1` | `clif_parse_GuildMessage` | guild_packets.go |
| `0x017F` | S->C | implemented | `HEADER_ZC_GUILD_CHAT` | `-1` | `-` | guild_packets.go, packet.go |
| `0x0180` | C->S | missing | `0x0180` | `6` | `clif_parse_GuildOpposition` | - |
| `0x0182` | S->C | untracked | `0x0182` | `106` | `-` | - |
| `0x0183` | C->S | missing | `0x0183` | `10` | `clif_parse_GuildDelAlliance` | - |
| `0x0185` | S->C | untracked | `0x0185` | `34` | `-` | - |
| `0x0187` | S->C | referenced | `0x0187` | `6` | `-` | packet.go |
| `0x018A` | C->S | implemented | `0x018a` | `4` | `clif_parse_QuitGame` | login_packets.go |
| `0x018B` | S->C | referenced | `0x018b` | `4` | `-` | packet.go, restart_packets.go |
| `0x018D` | S->C | implemented | `HEADER_ZC_MAKABLEITEMLIST` | `-1` | `-` | item_packets.go, packet.go |
| `0x018E` | C->S | implemented | `HEADER_CZ_REQMAKINGITEM` | `sizeof( struct PACKET_CZ_REQMAKINGITEM )` | `clif_parse_ProduceMix` | item_packets.go |
| `0x018F` | S->C | implemented | `HEADER_ZC_ACK_REQMAKINGITEM` | `6` | `-` | item_packets.go, packet.go |
| `0x0190` | C->S | referenced | `0x0190` | `19` | `clif_parse_ActionRequest` | item_packets.go, login_packets.go |
| `0x0193` | C->S | referenced | `0x0193` | `2` | `clif_parse_CloseKafra` | item_packets.go |
| `0x0196` | S->C | referenced | `0x0196` | `9` | `-` | packet.go, status_packets.go |
| `0x0197` | C->S | referenced | `HEADER_CZ_RESET` | `sizeof( PACKET_CZ_RESET )` | `clif_parse_ResetChar` | packet.go |
| `0x0198` | C->S | missing | `0x0198` | `8` | `clif_parse_GMChangeMapType` | - |
| `0x0199` | S->C | implemented | `0x0199` | `4` | `-` | packet.go, pvp_packets.go |
| `0x019A` | S->C | implemented | `0x019a` | `14` | `-` | packet.go, pvp_packets.go |
| `0x019C` | C->S | referenced | `HEADER_CZ_LOCALBROADCAST` | `-1` | `clif_parse_LocalBroadcast` | packet.go |
| `0x019D` | C->S | referenced | `0x019d` | `6` | `clif_parse_GMHide` | packet.go |
| `0x019E` | S->C | implemented | `0x019e` | `2` | `-` | pet_packets.go |
| `0x019F` | C->S | implemented | `0x019f` | `6` | `clif_parse_CatchPet` | pet_packets.go |
| `0x01A0` | S->C | implemented | `0x01a0` | `3` | `-` | pet_packets.go |
| `0x01A1` | C->S | implemented | `0x01a1` | `3` | `clif_parse_PetMenu` | pet_packets.go |
| `0x01A2` | S->C | implemented | `0x01a2` | `37` | `-` | pet_packets.go |
| `0x01A3` | S->C | implemented | `0x01a3` | `5` | `-` | pet_packets.go |
| `0x01A4` | S->C | implemented | `0x01a4` | `11` | `-` | pet_packets.go |
| `0x01A5` | C->S | implemented | `0x01a5` | `26` | `clif_parse_ChangePetName` | pet_packets.go |
| `0x01A6` | S->C | implemented | `0x01a6` | `-1` | `-` | pet_packets.go |
| `0x01A7` | C->S | implemented | `0x01a7` | `4` | `clif_parse_SelectEgg` | pet_packets.go |
| `0x01A8` | S->C | referenced | `0x01a8` | `4` | `-` | packet.go |
| `0x01A9` | C->S | implemented | `0x01a9` | `6` | `clif_parse_SendEmotion` (pet emotion) | pet_packets.go |
| `0x01AA` | S->C | implemented | `0x01aa` | `10` | `-` | pet_packets.go |
| `0x01AC` | S->C | referenced | `0x01ac` | `6` | `-` | skill_packets.go |
| `0x01AD` | S->C | referenced | `0x01ad` | `-1` | `-` | item_packets.go, packet.go |
| `0x01AE` | C->S | implemented | `HEADER_CZ_REQ_MAKINGARROW` | `sizeof( PACKET_CZ_REQ_MAKINGARROW )` | `clif_parse_SelectArrow` | item_packets.go |
| `0x01AF` | C->S | referenced | `0x01af` | `4` | `clif_parse_ChangeCart` | skill_packets.go |
| `0x01B0` | S->C | untracked | `0x01b0` | `11` | `-` | - |
| `0x01B1` | S->C | implemented | `0x01b1` | `7` | `-` | packet.go, server_info_packets.go |
| `0x01B2` | C->S | referenced | `0x01b2` | `-1` | `clif_parse_OpenVending` | vending_packets.go |
| `0x01B5` | S->C | referenced | `0x01b5` | `18` | `-` | packet.go |
| `0x01B6` | S->C | referenced | `0x01b6` | `114` | `-` | guild_packets.go, packet.go |
| `0x01B7` | S->C | untracked | `0x01b7` | `6` | `-` | - |
| `0x01B8` | S->C | untracked | `0x01b8` | `3` | `-` | - |
| `0x01BA` | C->S | missing | `0x01ba` | `26` | `clif_parse_GMShift` | - |
| `0x01BB` | C->S | missing | `0x01bb` | `26` | `clif_parse_GMShift` | - |
| `0x01BC` | C->S | missing | `0x01bc` | `26` | `clif_parse_GMRecall` | - |
| `0x01BD` | C->S | missing | `0x01bd` | `26` | `clif_parse_GMRecall` | - |
| `0x01BE` | S->C | untracked | `0x01be` | `2` | `-` | - |
| `0x01BF` | S->C | untracked | `0x01bf` | `3` | `-` | - |
| `0x01C0` | S->C | untracked | `0x01c0` | `2` | `-` | - |
| `0x01C1` | S->C | untracked | `0x01c1` | `14` | `-` | - |
| `0x01C2` | S->C | untracked | `0x01c2` | `10` | `-` | - |
| `0x01C3` | S->C | untracked | `0x01c3` | `-1` | `-` | - |
| `0x01C6` | S->C | untracked | `0x01c6` | `4` | `-` | - |
| `0x01C7` | S->C | untracked | `0x01c7` | `2` | `-` | - |
| `0x01CA` | S->C | referenced | `0x01ca` | `3` | `-` | packet.go |
| `0x01CB` | S->C | untracked | `0x01cb` | `9` | `-` | - |
| `0x01CC` | S->C | untracked | `0x01cc` | `9` | `-` | - |
| `0x01CD` | S->C | implemented | `0x01cd` | `30` | `-` | skill_packets.go, packet.go |
| `0x01CE` | C->S | implemented | `HEADER_CZ_SELECTAUTOSPELL` | `sizeof( PACKET_CZ_SELECTAUTOSPELL )` | `clif_parse_AutoSpell` | skill_packets.go |
| `0x01CF` | S->C | referenced | `0x01cf` | `28` | `-` | packet.go |
| `0x01D0` | S->C | referenced | `0x01d0` | `8` | `-` | packet.go |
| `0x01D5` | C->S | implemented | `HEADER_CZ_INPUT_EDITDLGSTR` | `-1` | `clif_parse_NpcStringInput` | npc_packets.go |
| `0x01D7` | S->C | referenced | `0x01d7` | `11` | `-` | actor_packets.go, packet.go |
| `0x01D8` | S->C | referenced | `0x01d8` | `54` | `-` | actor_packets.go, packet.go |
| `0x01D9` | S->C | referenced | `0x01d9` | `53` | `-` | actor_packets.go, packet.go |
| `0x01DA` | S->C | referenced | `0x01da` | `60` | `-` | actor_packets.go, packet.go |
| `0x01DB` | S->C | untracked | `0x01db` | `2` | `-` | - |
| `0x01DC` | S->C | referenced | `0x01dc` | `-1` | `-` | packet.go |
| `0x01DD` | S->C | untracked | `0x01dd` | `47` | `-` | - |
| `0x01DE` | S->C | referenced | `0x01de` | `33` | `-` | actor_packets.go, packet.go |
| `0x01DF` | C->S | missing | `0x01df` | `6` | `clif_parse_GMReqAccountName` | - |
| `0x01E0` | S->C | untracked | `0x01e0` | `30` | `-` | - |
| `0x01E1` | S->C | untracked | `0x01e1` | `8` | `-` | - |
| `0x01E2` | S->C | untracked | `0x01e2` | `34` | `-` | - |
| `0x01E3` | S->C | untracked | `0x01e3` | `14` | `-` | - |
| `0x01E4` | S->C | untracked | `0x01e4` | `2` | `-` | - |
| `0x01E5` | S->C | untracked | `0x01e5` | `6` | `-` | - |
| `0x01E6` | S->C | untracked | `0x01e6` | `26` | `-` | - |
| `0x01E7` | C->S | implemented | `0x01e7` | `2` | `clif_parse_NoviceDoriDori` | novice_packets.go |
| `0x01E8` | C->S | implemented | `HEADER_CZ_MAKE_GROUP2` | `sizeof( PACKET_CZ_MAKE_GROUP2 )` | `clif_parse_CreateParty2` | party_packets.go |
| `0x01EC` | S->C | untracked | `0x01ec` | `26` | `-` | - |
| `0x01ED` | C->S | missing | `0x01ed` | `2` | `clif_parse_NoviceExplosionSpirits` | - |
| `0x01F0` | S->C | untracked | `0x01f0` | `-1` | `-` | - |
| `0x01F1` | S->C | referenced | `0x01f1` | `-1` | `-` | packet.go |
| `0x01F3` | S->C | referenced | `0x01f3` | `10` | `-` | packet.go |
| `0x01F6` | S->C | implemented | `0x01f6` | `34` | `-` | adoption_packets.go, packet.go |
| `0x01F7` | C->S | implemented | `HEADER_CZ_JOIN_BABY` | `sizeof( PACKET_CZ_JOIN_BABY )` | `clif_parse_Adopt_reply` | adoption_packets.go |
| `0x01F8` | S->C | implemented | `0x01f8` | `2` | `-` | adoption_packets.go, packet.go |
| `0x01F9` | C->S | implemented | `0x01f9` | `6` | `clif_parse_Adopt_request` | adoption_packets.go |
| `0x01FA` | S->C | untracked | `0x01fa` | `48` | `-` | - |
| `0x01FB` | S->C | untracked | `0x01fb` | `56` | `-` | - |
| `0x01FC` | S->C | implemented | `HEADER_ZC_REPAIRITEMLIST` | `-1` | `-` | item_packets.go, packet.go |
| `0x01FD` | C->S | implemented | `HEADER_CZ_REQ_ITEMREPAIR1` | `sizeof( struct PACKET_CZ_REQ_ITEMREPAIR1 )` | `clif_parse_RepairItem` | item_packets.go |
| `0x01FE` | S->C | implemented | `HEADER_ZC_ACK_ITEMREPAIR` | `5` | `-` | item_packets.go, packet.go |
| `0x0200` | S->C | untracked | `0x0200` | `26` | `-` | - |
| `0x0201` | S->C | referenced | `0x0201` | `-1` | `-` | friend_packets.go, packet.go |
| `0x0202` | C->S | referenced | `0x0202` | `26` | `clif_parse_FriendsListAdd` | friend_packets.go |
| `0x0203` | C->S | referenced | `0x0203` | `10` | `clif_parse_FriendsListRemove` | friend_packets.go |
| `0x0204` | S->C | untracked | `0x0204` | `18` | `-` | - |
| `0x0207` | S->C | referenced | `0x0207` | `34` | `-` | friend_packets.go, packet.go |
| `0x0208` | C->S | referenced | `0x0208` | `14` | `clif_parse_FriendsListReply` | friend_packets.go |
| `0x0209` | S->C | referenced | `0x0209` | `36` | `-` | friend_packets.go, packet.go |
| `0x020A` | S->C | referenced | `0x020a` | `10` | `-` | friend_packets.go, packet.go |
| `0x020D` | S->C | untracked | `0x020d` | `-1` | `-` | - |
| `0x020E` | S->C | implemented | `0x020e` | `32` | `-` | taekwon_packets.go, packet.go |
| `0x020F` | C->S | implemented | `0x020f` | `10` | `clif_parse_PVPInfo` | pvp_packets.go |
| `0x0210` | S->C | implemented | `0x0210` | `22` | `-` | pvp_packets.go |
| `0x0212` | C->S | missing | `0x0212` | `26` | `clif_parse_GMRc` | - |
| `0x0213` | C->S | missing | `0x0213` | `26` | `clif_parse_Check` | - |
| `0x0214` | S->C | referenced | `0x0214` | `42` | `-` | packet.go |
| `0x0216` | S->C | implemented | `0x0216` | `6` | `-` | adoption_packets.go, packet.go |
| `0x0217` | C->S | implemented | `HEADER_CZ_BLACKSMITH_RANK` | `sizeof( PACKET_CZ_BLACKSMITH_RANK )` | `clif_parse_ranklist_blacksmith` | ranking_packets.go |
| `0x0218` | C->S | implemented | `HEADER_CZ_ALCHEMIST_RANK` | `sizeof( PACKET_CZ_ALCHEMIST_RANK )` | `clif_parse_ranklist_alchemist` | ranking_packets.go |
| `0x0219` | S->C | implemented | `HEADER_ZC_BLACKSMITH_RANK` | `282` | `-` | ranking_packets.go, packet.go |
| `0x021A` | S->C | implemented | `HEADER_ZC_ALCHEMIST_RANK` | `282` | `-` | ranking_packets.go, packet.go |
| `0x021B` | S->C | implemented | `HEADER_ZC_BLACKSMITH_POINT` | `10` | `-` | ranking_packets.go, packet.go |
| `0x021C` | S->C | implemented | `HEADER_ZC_ALCHEMIST_POINT` | `10` | `-` | ranking_packets.go, packet.go |
| `0x021D` | C->S | implemented | `HEADER_CZ_LESSEFFECT` | `sizeof( PACKET_CZ_LESSEFFECT )` | `clif_parse_LessEffect` | effect_packets.go |
| `0x021E` | S->C | implemented | `0x021e` | `6` | `-` | effect_packets.go, packet.go |
| `0x021F` | S->C | untracked | `0x021f` | `66` | `-` | - |
| `0x0220` | S->C | referenced | `0x0220` | `10` | `-` | packet.go |
| `0x0221` | S->C | implemented | `HEADER_ZC_NOTIFY_WEAPONITEMLIST` | `-1` | `-` | item_packets.go, packet.go |
| `0x0222` | C->S | implemented | `0x0222` | `6` | `clif_parse_WeaponRefine` | item_packets.go |
| `0x0223` | S->C | implemented | `HEADER_ZC_ACK_WEAPONREFINE` | `8` | `-` | item_packets.go, packet.go |
| `0x0224` | S->C | implemented | `0x0224` | `10` | `-` | ranking_packets.go, packet.go |
| `0x0225` | C->S | implemented | `HEADER_CZ_TAEKWON_RANK` | `sizeof( PACKET_CZ_TAEKWON_RANK )` | `clif_parse_ranklist_taekwon` | ranking_packets.go |
| `0x0226` | S->C | implemented | `0x0226` | `282` | `-` | ranking_packets.go, packet.go |
| `0x0227` | S->C | untracked | `0x0227` | `18` | `-` | - |
| `0x0228` | S->C | untracked | `0x0228` | `18` | `-` | - |
| `0x0229` | S->C | referenced | `0x0229` | `15` | `-` | actor_packets.go, packet.go |
| `0x022A` | S->C | referenced | `0x022a` | `58` | `-` | packet.go |
| `0x022B` | S->C | referenced | `0x022b` | `57` | `-` | packet.go |
| `0x022C` | S->C | referenced | `0x022c` | `65` | `-` | actor_packets.go, packet.go |
| `0x022D` | C->S | implemented | `0x022d` | `5` | `clif_parse_HomMenu` | companion_packets.go, client.go |
| `0x022E` | S->C | implemented | `0x022e` | `71` | `-` | companion_packets.go, packet.go |
| `0x022F` | S->C | implemented | `0x022f` | `5` | `-` | companion_packets.go, packet.go |
| `0x0230` | S->C | implemented | `0x0230` | `12` | `-` | companion_packets.go, packet.go |
| `0x0231` | C->S | implemented | `0x0231` | `26` | `clif_parse_ChangeHomunculusName` | companion_packets.go, client.go |
| `0x0232` | C->S | implemented | `HEADER_CZ_REQUEST_MOVENPC` | `sizeof( PACKET_CZ_REQUEST_MOVENPC )` | `clif_parse_HomMoveTo` | companion_packets.go, client.go |
| `0x0233` | C->S | implemented | `0x0233` | `11` | `clif_parse_HomAttack` | companion_packets.go, client.go |
| `0x0234` | C->S | implemented | `0x0234` | `6` | `clif_parse_HomMoveToMaster` | companion_packets.go, client.go |
| `0x0235` | S->C | implemented | `0x0235` | `-1` | `-` | companion_packets.go, packet.go |
| `0x0237` | C->S | missing | `HEADER_CZ_KILLER_RANK` | `sizeof( PACKET_CZ_KILLER_RANK )` | `clif_parse_ranklist_killer` | - |
| `0x0239` | S->C | implemented | `0x0239` | `11` | `-` | companion_packets.go, packet.go |
| `0x023A` | S->C | implemented | `0x023a` | `4` | `-` | storage_password_packets.go, packet.go |
| `0x023B` | C->S | implemented | `0x023b` | `36` | `clif_parse_StoragePassword` | storage_password_packets.go |
| `0x023C` | S->C | implemented | `0x023c` | `6` | `-` | storage_password_packets.go, packet.go |
| `0x023D` | S->C | untracked | `0x023d` | `-1` | `-` | - |
| `0x023E` | S->C | untracked | `0x023e` | `8` | `-` | - |
| `0x023F` | C->S | missing | `0x023f` | `2` | `clif_parse_Mail_refreshinbox` | - |
| `0x0240` | S->C | untracked | `0x0240` | `-1` | `-` | - |
| `0x0241` | C->S | missing | `0x0241` | `6` | `clif_parse_Mail_read` | - |
| `0x0242` | S->C | untracked | `0x0242` | `-1` | `-` | - |
| `0x0243` | C->S | missing | `0x0243` | `6` | `clif_parse_Mail_delete` | - |
| `0x0244` | C->S | missing | `0x0244` | `6` | `clif_parse_Mail_getattach` | - |
| `0x0245` | S->C | untracked | `0x0245` | `3` | `-` | - |
| `0x0246` | C->S | missing | `0x0246` | `4` | `clif_parse_Mail_winopen` | - |
| `0x0247` | C->S | missing | `0x0247` | `8` | `clif_parse_Mail_setattach` | - |
| `0x0248` | C->S | missing | `0x0248` | `-1` | `clif_parse_Mail_send` | - |
| `0x0249` | S->C | untracked | `0x0249` | `3` | `-` | - |
| `0x024A` | S->C | untracked | `0x024a` | `70` | `-` | - |
| `0x024B` | C->S | missing | `0x024b` | `4` | `clif_parse_Auction_cancelreg` | - |
| `0x024C` | C->S | missing | `0x024c` | `8` | `clif_parse_Auction_setitem` | - |
| `0x024D` | C->S | missing | `HEADER_CZ_AUCTION_ADD` | `sizeof( PACKET_CZ_AUCTION_ADD )` | `clif_parse_Auction_register` | - |
| `0x024E` | C->S | missing | `0x024e` | `6` | `clif_parse_Auction_cancel` | - |
| `0x024F` | C->S | missing | `HEADER_CZ_AUCTION_BUY` | `sizeof( PACKET_CZ_AUCTION_BUY )` | `clif_parse_Auction_bid` | - |
| `0x0250` | S->C | untracked | `0x0250` | `3` | `-` | - |
| `0x0251` | C->S | missing | `HEADER_CZ_AUCTION_ITEM_SEARCH` | `sizeof( PACKET_CZ_AUCTION_ITEM_SEARCH )` | `clif_parse_Auction_search` | - |
| `0x0252` | S->C | untracked | `0x0252` | `-1` | `-` | - |
| `0x0253` | S->C | implemented | `0x0253` | `3` | `-` | taekwon_packets.go, packet.go |
| `0x0254` | C->S | implemented | `0x0254` | `3` | `clif_parse_FeelSaveOk` | taekwon_packets.go |
| `0x0255` | S->C | untracked | `0x0255` | `5` | `-` | - |
| `0x0256` | S->C | untracked | `0x0256` | `5` | `-` | - |
| `0x0257` | S->C | untracked | `0x0257` | `8` | `-` | - |
| `0x0258` | S->C | untracked | `0x0258` | `2` | `-` | - |
| `0x0259` | S->C | untracked | `0x0259` | `3` | `-` | - |
| `0x025A` | S->C | referenced | `HEADER_ZC_MAKINGITEM_LIST` | `-1` | `-` | item_packets.go, packet.go |
| `0x025B` | C->S | missing | `HEADER_CZ_REQ_MAKINGITEM` | `sizeof( struct PACKET_CZ_REQ_MAKINGITEM )` | `clif_parse_Cooking` | - |
| `0x025C` | C->S | missing | `0x025c` | `4` | `clif_parse_Auction_buysell` | - |
| `0x025D` | C->S | missing | `0x025d` | `6` | `clif_parse_Auction_close` | - |
| `0x025E` | S->C | untracked | `0x025e` | `4` | `-` | - |
| `0x025F` | S->C | untracked | `0x025f` | `6` | `-` | - |
| `0x0260` | S->C | untracked | `0x0260` | `6` | `-` | - |
| `0x0261` | S->C | untracked | `0x0261` | `11` | `-` | - |
| `0x0262` | S->C | untracked | `0x0262` | `11` | `-` | - |
| `0x0263` | S->C | untracked | `0x0263` | `11` | `-` | - |
| `0x0264` | S->C | untracked | `0x0264` | `20` | `-` | - |
| `0x0265` | S->C | untracked | `0x0265` | `20` | `-` | - |
| `0x0266` | S->C | untracked | `0x0266` | `30` | `-` | - |
| `0x0267` | S->C | untracked | `0x0267` | `4` | `-` | - |
| `0x0268` | S->C | untracked | `0x0268` | `4` | `-` | - |
| `0x0269` | S->C | untracked | `0x0269` | `4` | `-` | - |
| `0x026A` | S->C | untracked | `0x026a` | `4` | `-` | - |
| `0x026B` | S->C | untracked | `0x026b` | `4` | `-` | - |
| `0x026C` | S->C | untracked | `0x026c` | `4` | `-` | - |
| `0x026D` | S->C | untracked | `0x026d` | `4` | `-` | - |
| `0x026F` | S->C | untracked | `0x026f` | `2` | `-` | - |
| `0x0270` | S->C | untracked | `0x0270` | `2` | `-` | - |
| `0x0271` | S->C | untracked | `0x0271` | `40` | `-` | - |
| `0x0272` | S->C | untracked | `0x0272` | `44` | `-` | - |
| `0x0273` | C->S | missing | `0x0273` | `30` | `clif_parse_Mail_return` | - |
| `0x0274` | S->C | untracked | `0x0274` | `8` | `-` | - |
| `0x0277` | S->C | untracked | `0x0277` | `84` | `-` | - |
| `0x0278` | S->C | untracked | `0x0278` | `2` | `-` | - |
| `0x0279` | S->C | untracked | `0x0279` | `2` | `-` | - |
| `0x027A` | S->C | untracked | `0x027a` | `-1` | `-` | - |
| `0x027B` | S->C | untracked | `0x027b` | `14` | `-` | - |
| `0x027C` | S->C | untracked | `0x027c` | `60` | `-` | - |
| `0x027D` | S->C | implemented | `0x027d` | `62` | `-` | companion_packets.go, packet.go |
| `0x027E` | S->C | untracked | `0x027e` | `-1` | `-` | - |
| `0x027F` | S->C | untracked | `0x027f` | `8` | `-` | - |
| `0x0280` | S->C | untracked | `0x0280` | `12` | `-` | - |
| `0x0281` | S->C | untracked | `0x0281` | `4` | `-` | - |
| `0x0282` | S->C | untracked | `0x0282` | `284` | `-` | - |
| `0x0283` | S->C | referenced | `0x0283` | `6` | `-` | packet.go |
| `0x0284` | S->C | untracked | `0x0284` | `14` | `-` | - |
| `0x0285` | S->C | untracked | `0x0285` | `6` | `-` | - |
| `0x0286` | S->C | untracked | `0x0286` | `4` | `-` | - |
| `0x0287` | S->C | untracked | `0x0287` | `-1` | `-` | - |
| `0x0288` | C->S | missing | `0x0288` | `10` | `clif_parse_npccashshop_buy` | - |
| `0x0289` | S->C | untracked | `0x0289` | `12` | `-` | - |
| `0x028B` | S->C | untracked | `0x028b` | `-1` | `-` | - |
| `0x028C` | S->C | untracked | `0x028c` | `46` | `-` | - |
| `0x028D` | S->C | untracked | `0x028d` | `34` | `-` | - |
| `0x028E` | S->C | untracked | `0x028e` | `4` | `-` | - |
| `0x028F` | S->C | untracked | `0x028f` | `6` | `-` | - |
| `0x0290` | S->C | untracked | `0x0290` | `4` | `-` | - |
| `0x0292` | C->S | implemented | `0x0292` | `2` | `clif_parse_AutoRevive` | revive_packets.go |
| `0x0293` | S->C | implemented | `0x0293` | `70` | `-` | packet.go, server_info_packets.go |
| `0x0294` | S->C | untracked | `0x0294` | `10` | `-` | - |
| `0x029B` | S->C | implemented | `0x029b` | `80` | `-` | companion_packets.go, packet.go |
| `0x029C` | S->C | implemented | `0x029c` | `66` | `-` | companion_packets.go, packet.go |
| `0x029D` | S->C | implemented | `0x029d` | `-1` | `-` | companion_packets.go, packet.go |
| `0x029E` | S->C | implemented | `0x029e` | `11` | `-` | companion_packets.go, packet.go |
| `0x029F` | C->S | implemented | `0x029f` | `3` | `clif_parse_mercenary_action` | companion_packets.go, client.go |
| `0x02A0` | S->C | untracked | `0x02a0` | `-1` | `-` | - |
| `0x02A1` | S->C | untracked | `0x02a1` | `-1` | `-` | - |
| `0x02A2` | S->C | implemented | `0x02a2` | `8` | `-` | companion_packets.go, packet.go |
| `0x02A3` | S->C | untracked | `0x02a3` | `-1` | `-` | - |
| `0x02A4` | S->C | untracked | `0x02a4` | `-1` | `-` | - |
| `0x02A5` | S->C | untracked | `0x02a5` | `8` | `-` | - |
| `0x02A6` | S->C | untracked | `0x02a6` | `22` | `-` | - |
| `0x02A7` | S->C | untracked | `0x02a7` | `22` | `-` | - |
| `0x02A8` | S->C | untracked | `0x02a8` | `162` | `-` | - |
| `0x02A9` | S->C | untracked | `0x02a9` | `58` | `-` | - |
| `0x02AA` | S->C | untracked | `0x02aa` | `4` | `-` | - |
| `0x02AB` | S->C | untracked | `0x02ab` | `36` | `-` | - |
| `0x02AC` | S->C | untracked | `0x02ac` | `6` | `-` | - |
| `0x02AD` | S->C | untracked | `0x02ad` | `8` | `-` | - |
| `0x02B0` | S->C | referenced | `0x02b0` | `85` | `-` | packet.go |
| `0x02B1` | S->C | untracked | `0x02b1` | `-1` | `-` | - |
| `0x02B2` | S->C | untracked | `0x02b2` | `-1` | `-` | - |
| `0x02B3` | S->C | untracked | `0x02b3` | `107` | `-` | - |
| `0x02B4` | S->C | untracked | `0x02b4` | `6` | `-` | - |
| `0x02B5` | S->C | untracked | `0x02b5` | `-1` | `-` | - |
| `0x02B6` | C->S | missing | `HEADER_CZ_ACTIVE_QUEST` | `sizeof( PACKET_CZ_ACTIVE_QUEST )` | `clif_parse_questStateAck` | - |
| `0x02B7` | S->C | untracked | `0x02b7` | `7` | `-` | - |
| `0x02B9` | S->C | implemented | `0x02b9` | `191` | `-` | hotkey_packets.go, packet.go |
| `0x02BA` | C->S | implemented | `0x02ba` | `11` | `clif_parse_Hotkey` | hotkey_packets.go |
| `0x02BC` | S->C | untracked | `0x02bc` | `6` | `-` | - |
| `0x02BF` | S->C | untracked | `0x02bf` | `-1` | `-` | - |
| `0x02C0` | S->C | untracked | `0x02c0` | `-1` | `-` | - |
| `0x02C1` | S->C | implemented | `0x02c1` | `-1` | `-` | chat_packets.go, packet.go |
| `0x02C2` | S->C | untracked | `0x02c2` | `-1` | `-` | - |
| `0x02C4` | C->S | implemented | `HEADER_CZ_PARTY_JOIN_REQ` | `sizeof( PACKET_CZ_PARTY_JOIN_REQ )` | `clif_parse_PartyInvite2` | party_packets.go |
| `0x02C5` | S->C | implemented | `0x02c5` | `30` | `-` | party_packets.go |
| `0x02C6` | S->C | implemented | `0x02c6` | `30` | `-` | party_packets.go |
| `0x02C7` | C->S | implemented | `HEADER_CZ_PARTY_JOIN_REQ_ACK` | `sizeof( PACKET_CZ_PARTY_JOIN_REQ_ACK )` | `clif_parse_ReplyPartyInvite2` | party_packets.go |
| `0x02C8` | C->S | implemented | `HEADER_CZ_PARTY_CONFIG` | `sizeof( PACKET_CZ_PARTY_CONFIG )` | `clif_parse_PartyTick` | party_packets.go |
| `0x02CA` | S->C | untracked | `0x02ca` | `3` | `-` | - |
| `0x02CB` | S->C | untracked | `0x02cb` | `65` | `-` | - |
| `0x02CC` | S->C | untracked | `0x02cc` | `4` | `-` | - |
| `0x02CD` | S->C | untracked | `0x02cd` | `71` | `-` | - |
| `0x02CE` | S->C | untracked | `0x02ce` | `10` | `-` | - |
| `0x02CF` | C->S | missing | `0x02cf` | `6` | `clif_parse_MemorialDungeonCommand` | - |
| `0x02D0` | S->C | referenced | `inventorylistequipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D1` | S->C | referenced | `storageListEquipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D2` | S->C | referenced | `cartlistequipType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02D5` | S->C | referenced | `0x02d5` | `2` | `-` | disconnect_packets.go, packet.go |
| `0x02D6` | C->S | referenced | `0x02d6` | `6` | `clif_parse_ViewPlayerEquip` | equipment_packets.go, packet.go |
| `0x02D7` | S->C | referenced | `0x02d7` | `-1` | `-` | equipment_packets.go, packet.go |
| `0x02D8` | C->S | referenced | `0x02d8` | `10` | `clif_parse_configuration` | equipment_packets.go, packet.go |
| `0x02D9` | S->C | referenced | `0x02d9` | `10` | `-` | packet.go |
| `0x02DB` | C->S | missing | `0x02db` | `-1` | `clif_parse_BattleChat` | - |
| `0x02DC` | S->C | untracked | `0x02dc` | `-1` | `-` | - |
| `0x02DD` | S->C | referenced | `0x02dd` | `32` | `-` | packet.go |
| `0x02DE` | S->C | untracked | `0x02de` | `6` | `-` | - |
| `0x02DF` | S->C | untracked | `0x02df` | `36` | `-` | - |
| `0x02E0` | S->C | untracked | `0x02e0` | `34` | `-` | - |
| `0x02E2` | S->C | untracked | `0x02e2` | `14` | `-` | - |
| `0x02E3` | S->C | untracked | `0x02e3` | `25` | `-` | - |
| `0x02E4` | S->C | untracked | `0x02e4` | `8` | `-` | - |
| `0x02E5` | S->C | untracked | `0x02e5` | `8` | `-` | - |
| `0x02E6` | S->C | untracked | `0x02e6` | `6` | `-` | - |
| `0x02E7` | S->C | untracked | `0x02e7` | `-1` | `-` | - |
| `0x02E8` | S->C | referenced | `inventorylistnormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02E9` | S->C | referenced | `cartlistnormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02EA` | S->C | referenced | `storageListNormalType` | `-1` | `-` | item_packets.go, packet.go |
| `0x02EC` | S->C | referenced | `0x02ec` | `67` | `-` | actor_packets.go, packet.go |
| `0x02ED` | S->C | referenced | `0x02ed` | `59` | `-` | actor_packets.go, packet.go |
| `0x02EE` | S->C | referenced | `0x02ee` | `60` | `-` | actor_packets.go, packet.go |
| `0x02EF` | S->C | referenced | `0x02ef` | `8` | `-` | packet.go |
| `0x02F0` | S->C | implemented | `0x02f0` | `10` | `-` | packet.go, server_info_packets.go |
| `0x02F1` | C->S | implemented | `0x02f1` | `2` | `clif_parse_progressbar` | server_info_packets.go |
| `0x02F2` | S->C | implemented | `0x02f2` | `2` | `-` | packet.go, server_info_packets.go |
| `0x02F3` | S->C | untracked | `0x02f3` | `-1` | `-` | - |
| `0x02F4` | S->C | untracked | `0x02f4` | `-1` | `-` | - |
| `0x02F5` | S->C | untracked | `0x02f5` | `-1` | `-` | - |
| `0x02F6` | S->C | untracked | `0x02f6` | `-1` | `-` | - |
| `0x02F7` | S->C | untracked | `0x02f7` | `-1` | `-` | - |
| `0x02F8` | S->C | untracked | `0x02f8` | `-1` | `-` | - |
| `0x02F9` | S->C | untracked | `0x02f9` | `-1` | `-` | - |
| `0x02FA` | S->C | untracked | `0x02fa` | `-1` | `-` | - |
| `0x02FB` | S->C | untracked | `0x02fb` | `-1` | `-` | - |
| `0x02FC` | S->C | untracked | `0x02fc` | `-1` | `-` | - |
| `0x02FD` | S->C | untracked | `0x02fd` | `-1` | `-` | - |
| `0x02FE` | S->C | untracked | `0x02fe` | `-1` | `-` | - |
| `0x02FF` | S->C | untracked | `0x02ff` | `-1` | `-` | - |
| `0x0300` | S->C | untracked | `0x0300` | `-1` | `-` | - |
| `0x0301` | S->C | untracked | `0x0301` | `-1` | `-` | - |
| `0x0302` | S->C | untracked | `0x0302` | `-1` | `-` | - |
| `0x0303` | S->C | untracked | `0x0303` | `-1` | `-` | - |
| `0x0304` | S->C | untracked | `0x0304` | `-1` | `-` | - |
| `0x0305` | S->C | untracked | `0x0305` | `-1` | `-` | - |
| `0x0306` | S->C | untracked | `0x0306` | `-1` | `-` | - |
| `0x0307` | S->C | untracked | `0x0307` | `-1` | `-` | - |
| `0x0308` | S->C | untracked | `0x0308` | `-1` | `-` | - |
| `0x0309` | S->C | untracked | `0x0309` | `-1` | `-` | - |
| `0x030A` | S->C | untracked | `0x030a` | `-1` | `-` | - |
| `0x030B` | S->C | untracked | `0x030b` | `-1` | `-` | - |
| `0x030C` | S->C | untracked | `0x030c` | `-1` | `-` | - |
| `0x030D` | S->C | untracked | `0x030d` | `-1` | `-` | - |
| `0x030E` | S->C | untracked | `0x030e` | `-1` | `-` | - |
| `0x030F` | S->C | untracked | `0x030f` | `-1` | `-` | - |
| `0x0310` | S->C | untracked | `0x0310` | `-1` | `-` | - |
| `0x0311` | S->C | untracked | `0x0311` | `-1` | `-` | - |
| `0x0312` | S->C | untracked | `0x0312` | `-1` | `-` | - |
| `0x0313` | S->C | untracked | `0x0313` | `-1` | `-` | - |
| `0x0314` | S->C | untracked | `0x0314` | `-1` | `-` | - |
| `0x0315` | S->C | untracked | `0x0315` | `-1` | `-` | - |
| `0x0316` | S->C | untracked | `0x0316` | `-1` | `-` | - |
| `0x0317` | S->C | untracked | `0x0317` | `-1` | `-` | - |
| `0x0318` | S->C | untracked | `0x0318` | `-1` | `-` | - |
| `0x0319` | S->C | untracked | `0x0319` | `-1` | `-` | - |
| `0x031A` | S->C | untracked | `0x031a` | `-1` | `-` | - |
| `0x031B` | S->C | untracked | `0x031b` | `-1` | `-` | - |
| `0x031C` | S->C | untracked | `0x031c` | `-1` | `-` | - |
| `0x031D` | S->C | untracked | `0x031d` | `-1` | `-` | - |
| `0x031E` | S->C | untracked | `0x031e` | `-1` | `-` | - |
| `0x031F` | S->C | untracked | `0x031f` | `-1` | `-` | - |
| `0x0320` | S->C | untracked | `0x0320` | `-1` | `-` | - |
| `0x0321` | S->C | untracked | `0x0321` | `-1` | `-` | - |
| `0x0322` | S->C | untracked | `0x0322` | `-1` | `-` | - |
| `0x0323` | S->C | untracked | `0x0323` | `-1` | `-` | - |
| `0x0324` | S->C | untracked | `0x0324` | `-1` | `-` | - |
| `0x0325` | S->C | untracked | `0x0325` | `-1` | `-` | - |
| `0x0326` | S->C | untracked | `0x0326` | `-1` | `-` | - |
| `0x0327` | S->C | untracked | `0x0327` | `-1` | `-` | - |
| `0x0328` | S->C | untracked | `0x0328` | `-1` | `-` | - |
| `0x0329` | S->C | untracked | `0x0329` | `-1` | `-` | - |
| `0x032A` | S->C | untracked | `0x032a` | `-1` | `-` | - |
| `0x032B` | S->C | untracked | `0x032b` | `-1` | `-` | - |
| `0x032C` | S->C | untracked | `0x032c` | `-1` | `-` | - |
| `0x032D` | S->C | untracked | `0x032d` | `-1` | `-` | - |
| `0x032E` | S->C | untracked | `0x032e` | `-1` | `-` | - |
| `0x032F` | S->C | untracked | `0x032f` | `-1` | `-` | - |
| `0x0330` | S->C | untracked | `0x0330` | `-1` | `-` | - |
| `0x0331` | S->C | untracked | `0x0331` | `-1` | `-` | - |
| `0x0332` | S->C | untracked | `0x0332` | `-1` | `-` | - |
| `0x0333` | S->C | untracked | `0x0333` | `-1` | `-` | - |
| `0x0334` | S->C | untracked | `0x0334` | `-1` | `-` | - |
| `0x0335` | S->C | untracked | `0x0335` | `-1` | `-` | - |
| `0x0336` | S->C | untracked | `0x0336` | `-1` | `-` | - |
| `0x0337` | S->C | untracked | `0x0337` | `-1` | `-` | - |
| `0x0338` | S->C | untracked | `0x0338` | `-1` | `-` | - |
| `0x0339` | S->C | untracked | `0x0339` | `-1` | `-` | - |
| `0x033A` | S->C | untracked | `0x033a` | `-1` | `-` | - |
| `0x033B` | S->C | untracked | `0x033b` | `-1` | `-` | - |
| `0x033C` | S->C | untracked | `0x033c` | `-1` | `-` | - |
| `0x033D` | S->C | untracked | `0x033d` | `-1` | `-` | - |
| `0x033E` | S->C | untracked | `0x033e` | `-1` | `-` | - |
| `0x033F` | S->C | untracked | `0x033f` | `-1` | `-` | - |
| `0x0340` | S->C | untracked | `0x0340` | `-1` | `-` | - |
| `0x0341` | S->C | untracked | `0x0341` | `-1` | `-` | - |
| `0x0342` | S->C | untracked | `0x0342` | `-1` | `-` | - |
| `0x0343` | S->C | untracked | `0x0343` | `-1` | `-` | - |
| `0x0344` | S->C | untracked | `0x0344` | `-1` | `-` | - |
| `0x0345` | S->C | untracked | `0x0345` | `-1` | `-` | - |
| `0x0346` | S->C | untracked | `0x0346` | `-1` | `-` | - |
| `0x0347` | S->C | untracked | `0x0347` | `-1` | `-` | - |
| `0x0348` | S->C | untracked | `0x0348` | `-1` | `-` | - |
| `0x0349` | S->C | untracked | `0x0349` | `-1` | `-` | - |
| `0x034A` | S->C | untracked | `0x034a` | `-1` | `-` | - |
| `0x034B` | S->C | untracked | `0x034b` | `-1` | `-` | - |
| `0x034C` | S->C | untracked | `0x034c` | `-1` | `-` | - |
| `0x034D` | S->C | untracked | `0x034d` | `-1` | `-` | - |
| `0x034E` | S->C | untracked | `0x034e` | `-1` | `-` | - |
| `0x034F` | S->C | untracked | `0x034f` | `-1` | `-` | - |
| `0x0350` | S->C | untracked | `0x0350` | `-1` | `-` | - |
| `0x0351` | S->C | untracked | `0x0351` | `-1` | `-` | - |
| `0x0352` | S->C | untracked | `0x0352` | `-1` | `-` | - |
| `0x0353` | S->C | untracked | `0x0353` | `-1` | `-` | - |
| `0x0354` | S->C | untracked | `0x0354` | `-1` | `-` | - |
| `0x0355` | S->C | untracked | `0x0355` | `-1` | `-` | - |
| `0x0356` | S->C | untracked | `0x0356` | `-1` | `-` | - |
| `0x0357` | S->C | untracked | `0x0357` | `-1` | `-` | - |
| `0x0358` | S->C | untracked | `0x0358` | `-1` | `-` | - |
| `0x0359` | S->C | untracked | `0x0359` | `-1` | `-` | - |
| `0x035A` | S->C | untracked | `0x035a` | `-1` | `-` | - |
| `0x035B` | S->C | untracked | `0x035b` | `-1` | `-` | - |
| `0x035C` | S->C | untracked | `0x035c` | `2` | `-` | - |
| `0x035D` | S->C | untracked | `0x035d` | `-1` | `-` | - |
| `0x035E` | S->C | untracked | `0x035e` | `2` | `-` | - |
| `0x035F` | S->C | referenced | `0x035f` | `-1` | `-` | login_packets.go |
| `0x0389` | S->C | untracked | `0x0389` | `-1` | `-` | - |
| `0x040C` | S->C | untracked | `0x040c` | `-1` | `-` | - |
| `0x040D` | S->C | untracked | `0x040d` | `-1` | `-` | - |
| `0x040E` | S->C | untracked | `0x040e` | `-1` | `-` | - |
| `0x040F` | S->C | untracked | `0x040f` | `-1` | `-` | - |
| `0x0410` | S->C | untracked | `0x0410` | `-1` | `-` | - |
| `0x0411` | S->C | untracked | `0x0411` | `-1` | `-` | - |
| `0x0412` | S->C | untracked | `0x0412` | `-1` | `-` | - |
| `0x0413` | S->C | untracked | `0x0413` | `-1` | `-` | - |
| `0x0414` | S->C | untracked | `0x0414` | `-1` | `-` | - |
| `0x0415` | S->C | untracked | `0x0415` | `-1` | `-` | - |
| `0x0416` | S->C | untracked | `0x0416` | `-1` | `-` | - |
| `0x0417` | S->C | untracked | `0x0417` | `-1` | `-` | - |
| `0x0418` | S->C | untracked | `0x0418` | `-1` | `-` | - |
| `0x0419` | S->C | untracked | `0x0419` | `-1` | `-` | - |
| `0x041A` | S->C | untracked | `0x041a` | `-1` | `-` | - |
| `0x041B` | S->C | untracked | `0x041b` | `-1` | `-` | - |
| `0x041C` | S->C | untracked | `0x041c` | `-1` | `-` | - |
| `0x041D` | S->C | untracked | `0x041d` | `-1` | `-` | - |
| `0x041E` | S->C | untracked | `0x041e` | `-1` | `-` | - |
| `0x041F` | S->C | untracked | `0x041f` | `-1` | `-` | - |
| `0x0420` | S->C | untracked | `0x0420` | `-1` | `-` | - |
| `0x0421` | S->C | untracked | `0x0421` | `-1` | `-` | - |
| `0x0422` | S->C | untracked | `0x0422` | `-1` | `-` | - |
| `0x0423` | S->C | untracked | `0x0423` | `-1` | `-` | - |
| `0x0424` | S->C | untracked | `0x0424` | `-1` | `-` | - |
| `0x0425` | S->C | untracked | `0x0425` | `-1` | `-` | - |
| `0x0426` | S->C | untracked | `0x0426` | `-1` | `-` | - |
| `0x0427` | S->C | untracked | `0x0427` | `-1` | `-` | - |
| `0x0428` | S->C | untracked | `0x0428` | `-1` | `-` | - |
| `0x0429` | S->C | untracked | `0x0429` | `-1` | `-` | - |
| `0x042A` | S->C | untracked | `0x042a` | `-1` | `-` | - |
| `0x042B` | S->C | untracked | `0x042b` | `-1` | `-` | - |
| `0x042C` | S->C | untracked | `0x042c` | `-1` | `-` | - |
| `0x042D` | S->C | untracked | `0x042d` | `-1` | `-` | - |
| `0x042E` | S->C | untracked | `0x042e` | `-1` | `-` | - |
| `0x042F` | S->C | untracked | `0x042f` | `-1` | `-` | - |
| `0x0430` | S->C | untracked | `0x0430` | `-1` | `-` | - |
| `0x0431` | S->C | untracked | `0x0431` | `-1` | `-` | - |
| `0x0432` | S->C | untracked | `0x0432` | `-1` | `-` | - |
| `0x0433` | S->C | untracked | `0x0433` | `-1` | `-` | - |
| `0x0434` | S->C | untracked | `0x0434` | `-1` | `-` | - |
| `0x0435` | S->C | untracked | `0x0435` | `-1` | `-` | - |
| `0x0436` | C->S | referenced | `0x0436` | `19` | `clif_parse_WantToConnection` | login_packets.go |
| `0x0437` | C->S | referenced | `0x0437` | `7` | `clif_parse_ActionRequest` | login_packets.go |
| `0x0438` | C->S | referenced | `0x0438` | `10` | `clif_parse_UseSkillToId` | skill_packets.go |
| `0x0439` | C->S | referenced | `0x0439` | `8` | `clif_parse_UseItem` | item_packets.go |
| `0x0802` | C->S | missing | `0x0802` | `18` | `clif_parse_PartyBookingRegisterReq` | - |
| `0x0803` | S->C | untracked | `0x0803` | `4` | `-` | - |
| `0x0804` | S->C | untracked | `0x0804` | `8` | `-` | - |
| `0x0805` | S->C | untracked | `0x0805` | `-1` | `-` | - |
| `0x0806` | C->S | missing | `0x0806` | `4` | `clif_parse_PartyBookingDeleteReq` | - |
| `0x0808` | S->C | untracked | `0x0808` | `4` | `-` | - |
| `0x08B3` | S->C | untracked | `0x8b3` | `-1` | `-` | - |

## Maintenance Notes

- Regenerate this when `PACKETVER`, rAthena packet shuffles, or Goro `network/` packet builders/parsers change.
- Treat `referenced` as opcode-level coverage only. Field layouts still need tests per feature.
- Prefer the preprocessed rAthena packet DB over hand-entered shuffle tables when adding 2008 client packets.
