# Map Effects And Weather TODO

Source of truth for this list: roBrowser parses RSW effect objects in
`Loaders/World.js`, feeds them through `Renderer/Map/Effects.js`, and starts
map-wide weather from `DB/Effects/WeatherEffect.js` through
`Renderer/ScreenEffectManager.js`.

## RSW-Placed Map Effects

These are effects placed directly in `.rsw` files. They should keep using the
world-effect renderer, but the effect IDs come from RSW data rather than skill
packets.

- [x] `EF_SMOKE` `44`: Prontera chimney smoke.
- [x] `EF_FIREFLY` `45`: faint floating white particles. Currently subtle; keep matching the reference.
- [x] `EF_TORCH` `47`: torch/fire flame effect.
- [x] `EF_BUBBLE` `109`: Bayalan underwater bubbles.
- [x] `EF_DRAGONSMOKE` `373`: house smoke variant.
- [x] `EF_BANJJAKII` `165`: Comodo fireworks ball sprite. This is distinct from map-wide fireworks weather.
- [x] `EF_MAPPILLAR` `231`: map light pillar animation. It is commented in the reference table, but maps that reference it now get a visible light pillar.
- [x] `EF_TORCH_RED` `690`, `EF_TORCH_GREEN` `691`, `EF_TORCH_PURPLE` `696`: colored torch variants. Reference marks them commented, but they are useful map effects.
- [x] `EF_MAP_GHOST` `692`: small ghost/aura bubbles.
- [x] `EF_GLOW1` `693`, `EF_GLOW2` `694`, `EF_GLOW4` `695`: translucent colored glow circles.
- [x] `EF_BUBBLE_DROP` `665`: little blue ball falling from sky.
- [x] `EF_RAINBOW` `410`: rainbow.

## Map-Wide Weather Effects

These are not RSW object effects. They are started from the map name. roBrowser
uses `Weather.effects` for weather and `Weather.sky` for airship clouds; the
original client starts the airship's `EF_CLOUD5` variant directly.

- [x] `xmas.rsw` -> `snow` -> `EF_SNOW` `162`: snow weather.
- [x] `comodo.rsw` -> `fireworks` -> `EF_POKJUK` `297`: fireworks weather.
- [x] `airplane.rsw`, `airplane_01.rsw` -> `EF_CLOUD5` `516`: original-client airship clouds, with faster one-way drift beneath the deck.
- [x] `einbroch.rsw` -> `cloud3` -> `EF_CLOUD3` `233`: industrial clouds/smoke.
- [x] `payon.rsw` -> `rain` -> `EF_RAIN` `161`: rain renderer exists, but Payon routing is disabled by default because roBrowser comments it out and it does not look natural on this target.

## Weather Systems Supported By roBrowser

roBrowser's screen effect manager also supports these modes, even when they are
not enabled in the default `Weather.effects` map. We should implement them once
we find maps or commands that need them.

- [x] `rain` -> `EF_RAIN` `161`.
- [x] `snow` -> `EF_SNOW` `162`.
- [x] `sakura` -> `EF_SAKURA` `163`.
- [x] `leaves` -> `EF_MAPLE` `333`.
- [x] `cloud` -> `EF_CLOUD` `229`.
- [x] `cloud2` -> `EF_CLOUD2` `230`.
- [x] `cloud3` -> `EF_CLOUD3` `233`.
- [x] `cloud4` -> `EF_CLOUD4` `515`.
- [x] `cloud5` -> `EF_CLOUD5` `516`.
- [x] `cloud6` -> `EF_CLOUD6` `592`.
- [x] `cloud7` -> `EF_CLOUD7` `697`.
- [x] `cloud8` -> `EF_CLOUD8` `698`.

## Sky And Cloud Color Overrides

roBrowser also has `Weather.sky` entries. These are not particles, but they
affect outdoor mood and should be treated as map rendering data.

- [x] Blue sky/cloud overrides: `airplane.rsw`, `airplane_01.rsw`, `gonryun.rsw`, `gon_dun02.rsw`, `himinn.rsw`, `ra_temsky.rsw`, `rwc01.rsw`, `sch_gld.rsw`, `valkyrie.rsw`, `yuno.rsw`.
- [x] Special sky colors: `5@tower.rsw`, `thana_boss.rsw`.

## Cleanup / Architecture

- [x] Keep RSW-placed effects and map-wide weather separate in code. This matches roBrowser and avoids pretending that all map visuals are skill effects.
- [x] Prefer table-driven world-effect components when an effect is a normal `EffectTable` entry.
- [x] Use dedicated weather systems for screen/map-wide effects that maintain particles over time, such as snow, rain, clouds, sakura, and fireworks.
- [x] When adding a new map effect, first check whether it is an RSW object effect, a `Weather.effects` entry, or a `Weather.sky` entry.
