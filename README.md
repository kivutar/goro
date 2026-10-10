# goro

`goro` is an open Ragnarok Online client recreation implemented in Go.

The runtime uses GoGPU/wgpu for the window and presentation path, with a modern
GPU pipeline and Vulkan support. Desktop builds use pure Go without CGO and can
be statically compiled. The Android development build adds a small native
Activity bridge and uses the NDK for audio and packaging.

This project wouldn't be possible without the existence of other open source clients
like ROBrowser Legacy and Open Midgard and their reverse engineering efforts.

![goro screenshot](https://github.com/kivutar/goro/releases/download/v0.0.1/goro-20260716-164507.png)

Visit the [project website](https://kivutar.github.io/goro/) or see Goro in
action on this [YouTube playlist](https://www.youtube.com/watch?v=5qldvYi9v-U&list=PLQhSdCGUOBwc).

We also have an active [Discord](https://discord.gg/5fXmjXJCwa) to provide live support and news.

## Project Goals

- Faithfully reimplement the original Ragnarok Online client.
- Focus on the pre-renewal 2008 experience first.
- Stay pure Go, without CGO, so cross-compilation and deployment stay simple on
  many platforms.
- Aim for simple, readable, hackable codebase.
- Use a modern GPU pipeline through GoGPU, including Vulkan and Wayland support.
- Deliver good performance, including support for high-refresh-rate displays.
- Provide a modernized, neat themeable UI built with `gogpu/ui`.
- Keep the engine reusable for creating new MMORPGs.
- Become a drop-in replacement for `Ragexe` and `Sakexe`.

### Stretch goals:

- Provide GRF tooling.
- Provide map, sprite, and model viewers.
- Support more Ragnarok Online client versions.
- Support optional anti-cheat and security features.

## Build and Run

```sh
CGO_ENABLED=0 go build -tags nofakecgo .
./goro
```

For an Android arm64 development APK and USB installation, see the
[Android build instructions](packaging/android/README.md).

### Keyboard and gamepad controls

Enable **Keyboard & joypad controls** in Settings to use the bundled WASD script.
The checkbox takes effect immediately and remembers your choice across restarts.
You can also enable it from the command line:

```sh
./goro --data-dir /path/to/OldRO --script builtin:wasd
```

You can also use `--script scripts/wasd.lua` to edit the bindings in Lua. Android
uses the bundled script by default when no other script is configured.

You can also enable it from chat with `/script wasd`. Use `/script none` to stop
scripting, or `/script` to list bundled scripts. Chat selection lasts for the
current run, including map changes.

| Control | Action |
| --- | --- |
| Left stick / D-pad | Move relative to camera (eight directions) |
| West face button (Xbox X / PlayStation Square) | Hold to loot |
| Right stick | Move the pointer |
| L2 + right stick | Rotate / tilt camera (where the map allows it) |
| R2 + right stick up/down | Zoom in/out (where the map allows it) |
| R2 + South / East / West / North | Hotbar slots 1 / 2 / 3 / 4 in the active row |
| R2 + D-pad Up / Right / Down / Left | Hotbar slots 5 / 6 / 7 / 8 in the active row |
| South | Confirm armed skill; otherwise hold to attack selected or nearest enemy; UI / pointer left click when unclaimed |
| East | Cancel skill / clear target; otherwise right click |
| Start / Menu | Escape menu |
| Right / left shoulder | Next / previous enemy, or eligible target for an armed skill |
| D-pad up/down, South/East in NPC dialogs | Select choice, confirm / cancel |
| Left stick click | Use the armed skill on the highlighted target |
| Select / Back on Android | Open the keyboard |

Keyboard WASD moves, Space loots, and F attacks the selected enemy or the nearest
one within eight cells. Tab / Shift+Tab cycle targets like the shoulders; F1–F9
activate hotbar slots; Enter confirms a skill target and Escape cancels targeting.

Both devices support target-first and skill-first play. A skill shortcut casts
immediately on an eligible selected target. Otherwise it highlights the nearest
eligible target; cycle to another or confirm. Enable `/noshift` to target monsters
with Heal. Ground skills use the pointer and left click / South. Self skills and
items activate immediately.

Gameplay controls pause while chat or a form has keyboard focus. UI pointer
clicks and NPC dialogs take priority over controller attacks. The first detected
controller stays selected until it disconnects; connecting and disconnecting
controllers does not require a restart.

Backends: Windows XInput, Linux evdev, macOS GameController, and Android
InputDevice. Windows requires an XInput-compatible controller or driver; Linux
requires read access to the controller's `/dev/input/event*` node and a driver
using the kernel's standard gamepad layout. macOS supports controllers exposed
with an extended gamepad profile by the system framework. Older nonstandard
controller mappings may need a driver or mapping fix.

### Configuration

Configuration precedence, from highest to lowest:

1. Command-line flags such as `--vsync=false`.
2. The file explicitly supplied with `--config <path>`.
3. `goro.ini` in the directory explicitly supplied with `--data-dir`, if it exists.
4. `./goro.ini` in the working directory, if it exists.
5. Built-in defaults.

Goro only searches the locations above; it does not automatically read or write
config files in system or per-user directories such as `/etc` or `$XDG_CONFIG_HOME`.
A missing explicit `--config` file is an error.

In-game settings, remembered login IDs, and chat shortcuts are saved to the
`--config` file when supplied, otherwise to `--data-dir/goro.ini` when `--data-dir`
is supplied, otherwise to `./goro.ini`. A missing data-directory or working-directory
file is created when saving preferences.

Example `goro.ini`:

```ini
data_dir = /home/kivutar/Téléchargements/OldRO

[window]
width = 1280
height = 720
fullscreen = false

[packet]
client_date = 20080910
profile = 23

[audio]
bgm = true
bgm_volume = 0.55

[render]
graphics_api = vulkan
vsync = true
anisotropy = 8
smooth_sprites = true
msaa = false
bloom = false
ssao = false

[network]
trace = false
```

Anisotropic filtering sharpens terrain and model textures viewed at an angle.
It defaults to 8x and can be changed live in Settings, or with `--anisotropy`:
`0` (off), `2`, `4`, `8`, or `16`. It falls back to trilinear filtering on
unsupported adapters. The current DX12 backend also uses trilinear filtering.

"Smooth world sprites" in Settings switches characters, monsters, NPCs, and
dropped items between smooth (default) and crisp nearest-neighbor sampling.
It applies immediately and is saved as `smooth_sprites`. Use
`--smooth-sprites=false` to start with crisp sprites. UI, shadows, effects,
and the cached sprite composition keep their existing filtering.

"Anti-aliasing 4x (Restart)" in Settings enables 4x MSAA for the world,
including terrain and model edges. UI and text stay at native resolution.
It defaults off to avoid the extra GPU memory and rendering cost, and is saved
as `msaa`. Restart after changing it, or launch with `--msaa` to try it.

"Bloom" in Settings adds a subtle glow around bright world pixels. It defaults
off, applies immediately, and is saved as `bloom`; `--bloom` enables it at launch.
It works with or without MSAA. The blur runs at quarter resolution and is
composited before the UI, keeping text and windows sharp. This is a brightness
filter on the existing scene, so bright scenery and sprites can glow too.

"Ambient occlusion" in Settings adds subtle contact shading to terrain and
models. It defaults off and applies immediately; `--ssao` enables it at launch.
It uses a separate geometry depth pass and half-resolution occlusion, works
with MSAA and bloom, and is applied before water, effects, sprites, and UI.
This is an optional visual enhancement, not an original-client effect; it adds
GPU work and can darken areas already shaded by the map's baked lighting.

Command-line options override the ini file:

```sh
CGO_ENABLED=0 go run -tags nofakecgo . --data-dir ~/kRO --fullscreen
CGO_ENABLED=0 go run -tags nofakecgo . --config ./goro.ini --bgm=false --graphics-api vulkan
```

Useful options:

```sh
--net-trace
--packet-client-date 20211103 # only when rAthena is rebuilt for that packetver
--fullscreen
--bgm=false
--bgm-volume 0.35
--no-audio # disable BGM and SFX output entirely (useful for profiling)
--graphics-api gles # fallback if Vulkan is unavailable
--vsync=false # unlock fps
--username <username> # prefill the username in login window
--password <password> # same for password
--autologin=true # perform server connection and login on startup
--server-slot 2 # select the third clientinfo.xml login server during autologin (default 0)
--char-server-slot 0 # select the first character server returned after login (default 0)
--char-slot 0 # select character slot 0 after autologin
--force-user-ai=true # start homunculus and mercenary in USER_AI custom mode
--script <path> # run an optional Lua control script during login and in game
```

## Getting Started

These are tutorials on how to setup a development environment.

- [Server setup](docs/rathena-setup.md)
- [Client setup](docs/client-setup.md)
- [Homunculus and mercenary support](docs/companions-20080910.md)
- [Lua bot scripting](docs/bot-scripting.md)

Runtime data is discovered from, in order:

- `--data-dir`
- `data_dir` in `goro.ini`
- current working directory

The resource manager currently looks for loose files such as:

- `data/clientinfo.xml`
- `data/sclientinfo.xml`
- `clientinfo.xml`
- `sclientinfo.xml`
- `System/clientinfo.xml`
- `System/sclientinfo.xml`

GRF archives are selected through `DATA.INI` in the data folder, with lower
numeric entries taking priority. Without it, Goro loads existing `fdata.grf`,
`rdata.grf`, `sdata.grf`, and `data.grf` in that order. Event and custom archives
must be listed explicitly. See [GRF archive configuration](docs/client-setup.md#grf-archives).

## Current Scope

Currently implemented (not a claim of complete reference-client parity):

 * Login
 * Character selection
 * Character creation
 * Character deletion
 * Maps display
   * Water
   * Map sounds
   * Lightmaps
   * RSW fog with reference-client near/far behavior
   * Animated models
   * Weather effects
   * Map-specific effects such as Yuno clouds, pillars, and fireworks
   * Indoors
   * Granny 3D NPC models
   * Black-covered map loading with first-frame prewarming and fade-in
 * Camera
   * Smooth character following and zoom
   * Rotation and bounded outdoor tilt
   * Indoor and map-authored viewpoint locks
   * Outdoor zoom and rotation restoration after locked maps
 * Battle and Gameplay
   * Enemies
   * Classic PvP map targeting and rank counter
   * [War of Emperium (2008 FE/SE client behavior)](docs/woe-20080910.md)
   * Path finding
   * Continuous held-click walking
   * Drops
   * Playable characters animation chain
   * Attack-ready stance
   * Jobs
     * Novice
     * Super Novice
     * First jobs
       * Swordman
       * Magician
       * Archer
       * Acolyte
       * Thief
       * Merchant
     * Second jobs
       * Knight and Crusader
       * Wizard and Sage
       * Hunter, Bard, and Dancer
       * Priest and Monk
       * Blacksmith and Alchemist
       * Assassin and Rogue
     * Transcendent jobs
       * Lord Knight and Paladin
       * High Wizard and Professor
       * Sniper, Clown, and Gypsy
       * High Priest and Champion
       * Whitesmith and Creator
       * Assassin Cross and Stalker
     * Expanded jobs, including baby variants
       * Taekwon, Star Gladiator, and Soul Linker
       * Gunslinger and Ninja
   * Skill effects
     * Ground skill units and cast markers
     * Song and dance effects
   * Skill casting
   * Walk cancellation
   * Casting cancellation
   * Cursor snap
   * Noshift
   * Noctrl
   * Item drops
   * Item identification
   * Card composition
   * Trading
   * Vending
   * [Legacy mail with item and Zeny attachments](docs/legacy-mail-20080910.md)
   * Show equipment
   * Alchemist crafting
   * Blacksmith repair and weapon refinement
   * Guilds
     * Creation and invitations
     * Member and position management
     * Leaving, member expulsion, and guild disbanding
     * Guild skills
     * Notices and expulsion history
     * Emblem selection
   * Pets
     * Capture slot machine
     * Egg hatching
     * Feeding
     * Status window and rename
     * Accessory equip/unequip
     * Performance actions
     * Emotes and talk bubbles
     * Feeding emotion reactions
     * Familiarity-gated client-side talk triggers
   * Companions
     * Homunculi
       * Status, skills, feeding, renaming, and deletion
       * Movement, combat, and skill commands
       * Default and custom USER_AI support
     * Mercenaries
       * Status and skill management
       * Movement, combat, and skill commands
       * Default and custom USER_AI support
     * Falcons
   * Friends
   * Parties
   * Whispers
   * Chat rooms
   * Character presentation
     * Item-specific weapon sprites
     * Mounts
     * Wedding sprites
     * Level 99 aura
 * UI
   * Overlapping windows with focus-to-front ordering and stable dragging
   * Basic information
   * Button bar
   * Multi-row shortcuts bar with classic key bindings
   * Classic Battle Mode (`/bm`), direct typing, and F12 shortcut-bar switching
   * Console
   * Minimap with player, NPC, party, and guild markers
   * Classic world map with player/party locations and minimap previews
   * Quest journal with descriptions, hunt progress, and time limits
   * Items with vertical category tabs
   * Equipment
   * Option
     * Settings
   * Friends
   * Party & party settings
   * Guild management
   * Chat rooms
   * Stats
   * Skills with class-level vertical tabs
   * Homunculus and mercenary status and skill windows
   * Emote window
   * Cart Storage
   * Kafra Storage with vertical category tabs
   * Teleport skill modal
   * Warp skill modal
   * Cart appearance modal
   * Trade window
   * Vending windows
   * Legacy mailbox, read-mail window, and multiline composer
   * Card composition and full card illustration windows
   * Show-equipment window
   * Alchemist crafting window
   * Blacksmith repair and refinement windows
   * Item pickup notifications
   * Item and skill tooltips
   * Status icons with roBrowser-sourced metadata
 * Emotes
 * Overlay text
   * FPS meter
   * Character names and HP/SP bars
   * Speech bubbles
 * Optional Lua character scripting
   * Player, enemy, nearby-player, companion, floor-item, and inventory state
   * Walking, stopping, attacking, looting, item use, chat, and targeted skills
   * Layout-independent physical keyboard input and text input
   * Lua-defined keyboard controls for movement, combat, looting, and skill target cycling
 * Tools
   * GRF packing and extraction
