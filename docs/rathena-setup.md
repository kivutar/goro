# rAthena Setup

Goro currently targets the classic pre-renewal `2008-09-10aSakexe` flow.
Stock rAthena is close, but it needs a few compatibility patches for this
client profile.

## Pre-Patched Fork

The simplest option is to use the Goro-compatible rAthena fork:

```sh
git clone https://github.com/kivutar/rathena.git
```

This fork already has the compatibility patches below applied. Use the manual
patch section only when starting from upstream rAthena.

## Required Patches

Save this as `goro-rathena.patch` at the rAthena root and apply it with
`patch -p1 < goro-rathena.patch`.

```diff
diff --git a/src/char/char.cpp b/src/char/char.cpp
index 28f208023..5ba2582dd 100644
--- a/src/char/char.cpp
+++ b/src/char/char.cpp
@@ -1795,10 +1795,17 @@ int32 char_mmo_char_tobuf( CHARACTER_INFO& info, mmo_charstatus& p ){
 	info.virtue = p.karma;
 	info.honor = p.manner;
 	info.jobpoint = umin( p.status_point, INT16_MAX );
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+	info.hp = u32min( p.hp, INT16_MAX );
+	info.maxhp = u32min( p.max_hp, INT16_MAX );
+	info.sp = u32min( p.sp, INT16_MAX );
+	info.maxsp = u32min( p.max_sp, INT16_MAX );
+#else
 	info.hp = p.hp;
 	info.maxhp = p.max_hp;
 	info.sp = min( p.sp, INT16_MAX );
 	info.maxsp = min( p.max_sp, INT16_MAX );
+#endif
 	info.speed = DEFAULT_WALK_SPEED; // p.speed;
 	info.job = p.class_;
 	info.head = p.hair;
@@ -1828,9 +1835,14 @@ int32 char_mmo_char_tobuf( CHARACTER_INFO& info, mmo_charstatus& p ){
 	info.Int = (uint8)u16min( p.int_, UINT8_MAX );
 	info.Dex = (uint8)u16min( p.dex, UINT8_MAX );
 	info.Luk = (uint8)u16min( p.luk, UINT8_MAX );
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+	info.CharNum = p.slot;
+	info.hairColor = u16min( p.hair_color, INT16_MAX );
+#else
 	info.CharNum = p.slot;
 	info.hairColor = (uint8)u16min( p.hair_color, UINT8_MAX );
 	info.bIsChangedCharName = ( p.rename > 0 ) ? 0 : 1;
+#endif
 #if (PACKETVER >= 20100720 && PACKETVER <= 20100727) || PACKETVER >= 20100803
 	mapindex_getmapname_ext( p.last_point.map, info.mapName );
 #endif
diff --git a/src/char/char_clif.cpp b/src/char/char_clif.cpp
index 53ff6e075..a8737e7da 100644
--- a/src/char/char_clif.cpp
+++ b/src/char/char_clif.cpp
@@ -1478,6 +1478,10 @@ TIMER_FUNC(charblock_timer){
  * <GID>L <szExpireDate>20B (TAG_CHARACTER_BLOCK_INFO)
  */
 void chclif_block_character( int32 fd, char_session_data& sd){
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+	return;
+#endif
+
 	time_t now = time( nullptr );
 
 	PACKET_HC_BLOCK_CHARACTER* p = reinterpret_cast<PACKET_HC_BLOCK_CHARACTER*>( packet_buffer );
diff --git a/src/common/packets.hpp b/src/common/packets.hpp
index 24a7d4d68..6307da0fc 100644
--- a/src/common/packets.hpp
+++ b/src/common/packets.hpp
@@ -28,6 +28,10 @@ struct PACKET{
 	int16 packetLength;
 } __attribute__((packed));
 
+#if defined(PACKETVER_SAK_NUM) && PACKETVER_SAK_NUM > 0 && PACKETVER_SAK_NUM < 20090225
+	#define PACKETVER_SAK_LEGACY_CHARINFO
+#endif
+
 struct CHARACTER_INFO{
 	uint32 GID;
 #if PACKETVER >= 20170830
@@ -48,6 +52,12 @@ struct CHARACTER_INFO{
 	int32 virtue;
 	int32 honor;
 	int16 jobpoint;
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+	int16 hp;
+	int16 maxhp;
+	int16 sp;
+	int16 maxsp;
+#else
 #if PACKETVER_RE_NUM >= 20211103 || PACKETVER_MAIN_NUM >= 20220330
 	int64 hp;
 	int64 maxhp;
@@ -58,6 +68,7 @@ struct CHARACTER_INFO{
 	int32 maxhp;
 	int16 sp;
 	int16 maxsp;
+#endif
 #endif
 	int16 speed;
 	int16 job;
@@ -81,9 +92,14 @@ struct CHARACTER_INFO{
 	uint8 Int;
 	uint8 Dex;
 	uint8 Luk;
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+	int16 CharNum;
+	int16 hairColor;
+#else
 	uint8 CharNum;
 	uint8 hairColor;
 	int16 bIsChangedCharName;
+#endif
 #if (PACKETVER >= 20100720 && PACKETVER <= 20100727) || PACKETVER >= 20100803
 	char mapName[16];
 #endif
@@ -104,6 +120,10 @@ struct CHARACTER_INFO{
 #endif
 } __attribute__((packed));
 
+#ifdef PACKETVER_SAK_LEGACY_CHARINFO
+static_assert( sizeof( CHARACTER_INFO ) == 108, "Legacy Sakexe CHARACTER_INFO size mismatch" );
+#endif
+
 struct PACKET_CA_LOGIN{
 	int16 packetType;
 	uint32 version;
diff --git a/src/custom/defines_pre.hpp b/src/custom/defines_pre.hpp
index 93b1780dc..259066662 100644
--- a/src/custom/defines_pre.hpp
+++ b/src/custom/defines_pre.hpp
@@ -9,6 +9,15 @@
  * For detailed guidance on these check http://rathena.org/wiki/SRC/config/
  **/
 
+#ifndef PACKETVER
+	#define PACKETVER 20080910
+#endif
+#ifndef PACKETVER_SAK_NUM
+	#define PACKETVER_SAK_NUM 20080910
+#endif
+#ifndef PRERE
+	#define PRERE
+#endif
 
 
 #endif /* CONFIG_CUSTOM_DEFINES_PRE_HPP */
diff --git a/src/map/clif.cpp b/src/map/clif.cpp
index d3f4c71b6..fb3ae029a 100644
--- a/src/map/clif.cpp
+++ b/src/map/clif.cpp
@@ -1653,7 +1653,7 @@ static inline bool clif_npc_mayapurple( const block_list& bl ){
 }
 
 /// For the stupid cloth-dye bug. Resends the given view data to the area specified by bl.
-void clif_refresh_clothcolor( const block_list& bl, enum send_target target, block_list* tbl = nullptr ){
+void clif_refresh_clothcolor( const block_list& bl, enum send_target target, const block_list* tbl = nullptr ){
 // Unconfirmed when this was fixed, if you encounter any problems, feel free to report them
 #if PACKETVER < 20091103
 	const view_data* vd = status_get_viewdata( &bl );
@@ -1670,7 +1670,7 @@ void clif_refresh_clothcolor( const block_list& bl, enum send_target target, blo
 		tbl = &bl;
 	}
 
-	clif_sprite_change( tbl, bl.id, LOOK_CLOTHES_COLOR, vd->look[LOOK_CLOTHES_COLOR], 0, target );
+	clif_sprite_change( const_cast<block_list*>( tbl ), bl.id, LOOK_CLOTHES_COLOR, vd->look[LOOK_CLOTHES_COLOR], 0, target );
 #endif
 }
 
@@ -1767,10 +1767,10 @@ int32 clif_spawn( const block_list* bl, bool walking ){
 /// 0x7db <type>.W <value>.L (ZC_HO_PAR_CHANGE)
 /// 0xba5 <type>.W <value>.Q (ZC_HO_PAR_CHANGE2)
 void clif_homunculus_updatestatus( const map_session_data& sd, _sp type ) {
-#if PACKETVER >= 20090610
    if( !hom_is_active(sd.hd) )
        return;

+#if PACKETVER >= 20090610
    PACKET_ZC_HO_PAR_CHANGE p = {};

    p.packetType = HEADER_ZC_HO_PAR_CHANGE;
@@ -1817,6 +1817,8 @@ void clif_homunculus_updatestatus( const map_session_data& sd, _sp type ) {
    }

    clif_send(&p, sizeof(p), &sd, SELF);
+#else
+    clif_hominfo(&sd, sd.hd, 0);
 #endif
 }

@@ -1826 +1826 @@ void clif_hominfo( const map_session_data* sd, const homun_data *hd, int32 flag ){
-#if PACKETVER_MAIN_NUM >= 20101005 || PACKETVER_RE_NUM >= 20080827 || defined(PACKETVER_ZERO)
+#if PACKETVER_MAIN_NUM >= 20101005 || PACKETVER_RE_NUM >= 20080827 || PACKETVER_SAK_NUM >= 20080618 || defined(PACKETVER_ZERO)
diff --git a/src/map/clif_packetdb.hpp b/src/map/clif_packetdb.hpp
index 79cf8caea..45447e40b 100644
--- a/src/map/clif_packetdb.hpp
+++ b/src/map/clif_packetdb.hpp
@@ -1192,8 +1192,10 @@
 #endif
 
 // Renewal Clients
+#ifdef PACKETVER_RE_NUM
+
 // 2008-08-27aRagexeRE
-#if PACKETVER >= 20080827
+#if PACKETVER_RE_NUM >= 20080827
 	parseable_packet(0x0072,22,clif_parse_UseSkillToId,9,15,18);
 	packet(0x007c,44);
 	parseable_packet(0x007e,105,clif_parse_UseSkillToPosMoreInfo,10,14,18,23,25);
@@ -1217,7 +1219,7 @@
 #endif
 
 // 2008-09-10aRagexeRE
-#if PACKETVER >= 20080910
+#if PACKETVER_RE_NUM >= 20080910
 	parseable_packet(0x0436,19,clif_parse_WantToConnection,2,6,10,14,18);
 	parseable_packet(0x0437,7,clif_parse_ActionRequest,2,6);
 	parseable_packet(0x0438,10,clif_parse_UseSkillToId,2,4,6);
@@ -1225,25 +1227,27 @@
 #endif
 
 // 2008-11-12aRagexeRE
-#if PACKETVER >= 20081112
+#if PACKETVER_RE_NUM >= 20081112
 	packet(0x043f,8);
 #endif
 
 // 2008-12-17bRagexeRE
-#if PACKETVER >= 20081217
+#if PACKETVER_RE_NUM >= 20081217
 	packet(0x006d,114);
 #endif
 
 // 2009-01-21aRagexeRE
-#if PACKETVER >= 20090121
+#if PACKETVER_RE_NUM >= 20090121
 	packet(0x043f,25);
 #endif
 
 // 2009-05-20aRagexeRE
-#if PACKETVER >= 20090520
+#if PACKETVER_RE_NUM >= 20090520
 	parseable_packet( 0x0447, 2, clif_parse_blocking_playcancel, 0 );
 #endif
 
+#endif
+
 // 2009-06-03aRagexeRE
 #if PACKETVER >= 20090603
 	parseable_packet(0x07d7,8,clif_parse_PartyChangeOption,2,6,7);
diff --git a/src/map/packets_struct.hpp b/src/map/packets_struct.hpp
index 52d1555d9..1720f560c 100644
--- a/src/map/packets_struct.hpp
+++ b/src/map/packets_struct.hpp
@@ -2854 +2854 @@ struct PACKET_ZC_PROPERTY_HOMUN {
-#elif PACKETVER_MAIN_NUM >= 20101005 || PACKETVER_RE_NUM >= 20080827 || defined(PACKETVER_ZERO)
+#elif PACKETVER_MAIN_NUM >= 20101005 || PACKETVER_RE_NUM >= 20080827 || PACKETVER_SAK_NUM >= 20080618 || defined(PACKETVER_ZERO)
@@ -3975,7 +3975,8 @@ struct PACKET_ZC_USESKILL_ACK {
 	uint8 disposable;
 } __attribute__((packed));
 DEFINE_PACKET_HEADER(ZC_USESKILL_ACK, 0x07fb);
-#elif PACKETVER_MAIN_NUM >= 20090406 || PACKETVER_SAK_NUM >= 20080618 || PACKETVER_RE_NUM >= 20080827 || defined(PACKETVER_ZERO)
+// 2008-09-10aSakexe uses shuffled CZ skill packets but still expects the classic cast ACK.
+#elif PACKETVER_MAIN_NUM >= 20090406 || PACKETVER == 20080910 || PACKETVER_SAK_NUM >= 20080618 || PACKETVER_RE_NUM >= 20080827 || defined(PACKETVER_ZERO)
 struct PACKET_ZC_USESKILL_ACK {
 	int16 packetType;
 	uint32 srcId;
```

## Build

After patching, regenerate the makefiles and build the server:

```sh
./configure --enable-packetver=20080910 --enable-prere
make clean
make server
```

`--enable-packetver=20080910` adds `-DPACKETVER=20080910`, and
`--enable-prere` adds `-DPRERE`. rAthena does not expose a configure switch for
`PACKETVER_SAK_NUM`, so keep the `#define PACKETVER_SAK_NUM 20080910` patch in
`src/custom/defines_pre.hpp`.

The homunculus hunk is required for 2008 Sakray clients. `ZC_HO_PAR_CHANGE`
(`0x07DB`) starts at `PACKETVER >= 20090610`; older clients expect the server to
refresh the full `ZC_PROPERTY_HOMUN` (`0x022E`) packet when homunculus HP, SP, or
EXP changes. Without this, the homunculus status window can stay stuck on stale
values after the first info request.

## Required Config

In `conf/char_athena.conf`:

```conf
char_del_delay: 0
char_del_option: 1
pincode_enabled: no
char_move_enabled: no
allowed_job_flag: 1
```

`char_del_delay: 0` is important: delayed character deletion is for newer
clients, while the 2008 client deletes directly after the email check.

## MariaDB Import

For a fresh local install, create the main and log databases:

```sql
CREATE DATABASE rathena;
CREATE DATABASE log;
```

Import the base schemas:

```sh
mysql rathena < sql-files/main.sql
mysql rathena < sql-files/web.sql
mysql rathena < sql-files/roulette_default_data.sql
mysql log < sql-files/logs.sql
```

For pre-renewal SQL item/mob tables, also import:

```sh
mysql rathena < sql-files/item_db.sql
mysql rathena < sql-files/item_db_equip.sql
mysql rathena < sql-files/item_db_etc.sql
mysql rathena < sql-files/item_db_usable.sql
mysql rathena < sql-files/item_db2.sql
mysql rathena < sql-files/mob_db.sql
mysql rathena < sql-files/mob_db2.sql
mysql rathena < sql-files/mob_skill_db.sql
mysql rathena < sql-files/mob_skill_db2.sql
```

Use the same database names in `conf/inter_athena.conf`:

```conf
map_server_db: rathena
log_db_db: log
```

## Local Development Notes

Point rAthena at the same 2008-era data used by Goro:

```conf
// conf/grf-files.txt
grf: /home/kivutar/Téléchargements/OldRO/data.grf
data_dir: /home/kivutar/Téléchargements/OldRO/
```

If your map cache does not match the GRF, rebuild it:

```sh
make tools
./mapcache \
  -grf conf/grf-files.txt \
  -list db/map_index.txt \
  -cache db/pre-re/map_cache.dat \
  -rebuild
```

For local-only testing, keep all server IPs on localhost:

```conf
// conf/char_athena.conf
login_ip: 127.0.0.1
char_ip: 127.0.0.1

// conf/map_athena.conf
char_ip: 127.0.0.1
map_ip: 127.0.0.1
```

For LAN testing, use the host IP that clients can reach:

```conf
// example: rAthena host is 192.168.1.169
char_ip: 192.168.1.169
map_ip: 192.168.1.169
```

Some useful convenience settings for testing:

```conf
// conf/login_athena.conf
new_account: yes

// conf/char_athena.conf
start_zeny: 100000
```

To make a test account a GM:

```sql
UPDATE login SET group_id = 99 WHERE userid = 'Kivutar';
```

Starter zeny, test NPCs, and GM accounts are optional. They are not client
compatibility requirements.
