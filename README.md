# goro

`goro` is a Go Ragnarok Online client foundation.

The runtime uses GoGPU/wgpu for the window and presentation path, with a modern
GPU pipeline and Vulkan support. Build and run with `CGO_ENABLED=0` and
`-tags nofakecgo`.

## Goals

- Faithfully reimplement the Ragnarok Online client.
- Focus on the pre-renewal 2008 experience first.
- Stay pure Go, without cgo, so cross-compilation and deployment stay simple on
  many platforms.
- Use a modern GPU pipeline through GoGPU, including Vulkan and Wayland support.
- Deliver good performance, including support for high-refresh-rate displays.
- Provide a modernized, themeable UI built with `gogpu/ui`.
- Keep the engine reusable for creating new MMORPGs.
- Become a drop-in replacement for `ragexe` and `sakexe`.

Stretch goals:

- Provide GRF tooling.
- Provide map, sprite, and model viewers.
- Support optional Lua scripting for autoplay and automation experiments.
- Support more Ragnarok Online client versions.
- Support optional anti-cheat and security features.

## Run

```sh
CGO_ENABLED=0 go run -tags nofakecgo .
```

Configuration is loaded from `goro.ini` in the current directory when the file
exists. Pass another file with `--config`:

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

[network]
trace = false
```

Command-line options override the ini file:

```sh
CGO_ENABLED=0 go run -tags nofakecgo . --data-dir /home/kivutar/Téléchargements/OldRO --fullscreen
CGO_ENABLED=0 go run -tags nofakecgo . --config ./oldro.ini --bgm=false --graphics-api gles
```

Useful options:

```sh
CGO_ENABLED=0 go run -tags nofakecgo . --net-trace
CGO_ENABLED=0 go run -tags nofakecgo . --packet-client-date 20211103 # only when rAthena is rebuilt for that packetver
CGO_ENABLED=0 go run -tags nofakecgo . --fullscreen
CGO_ENABLED=0 go run -tags nofakecgo . --bgm=false
CGO_ENABLED=0 go run -tags nofakecgo . --bgm-volume 0.35
CGO_ENABLED=0 go run -tags nofakecgo . --graphics-api gles # fallback if Vulkan is unavailable
```

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

## Current Scope

This first pass establishes the same broad subsystem boundaries used by
OpenMidgard:

- `config` startup configuration
- `res` runtime data discovery and `clientinfo.xml` parsing
- `network` TCP connection and RO packet framing
- `session` account/character/session state
- `world` map and actor state
- `game` login/server selection and world rendering
- `render` GoGPU backend
- `input` per-frame input snapshot

It is not yet a complete RO implementation. The next substantial steps are GRF
loading, packet serializers for account/char/map login, and map asset parsers.
