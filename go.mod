module github.com/kivutar/goro

go 1.26.4

require (
	github.com/charmbracelet/log v1.0.0
	github.com/ebitengine/oto/v3 v3.5.0-alpha.8
	github.com/go-fonts/dejavu v0.3.4
	github.com/gogpu/gogpu v0.44.6
	github.com/gogpu/gpucontext v0.21.1
	github.com/gogpu/gputypes v0.5.1
	github.com/gogpu/naga v0.17.15
	github.com/gogpu/ui v0.1.36
	github.com/gogpu/wgpu v0.30.19
	github.com/yuin/gopher-lua v1.1.2
	golang.org/x/image v0.43.0
	golang.org/x/text v0.38.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/go-logfmt/logfmt v0.6.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sync v0.21.0 // indirect
)

require (
	github.com/coregx/signals v0.1.0 // indirect
	github.com/go-text/typesetting v0.3.4 // indirect
	github.com/go-webgpu/goffi v0.6.0 // indirect
	github.com/go-webgpu/webgpu v0.5.3 // indirect
	github.com/godexture/codec-mp3 v0.0.0
	github.com/godexture/core v0.0.0
	github.com/godexture/format-mp3 v0.0.0
	github.com/godexture/metadata-id3 v0.0.0 // indirect
	github.com/godexture/sdk v0.0.0
	github.com/gogpu/gg v0.48.16
	github.com/jfreymuth/pulse v0.1.1 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/godexture/codec-mp3 => github.com/godexture/godec/plugins/codec-mp3 v0.0.0-20260621142744-bd77e78cfab1

replace github.com/godexture/format-mp3 => github.com/godexture/godec/plugins/format-mp3 v0.0.0-20260621142744-bd77e78cfab1

replace github.com/godexture/core => github.com/godexture/godec/core v0.0.0-20260621142744-bd77e78cfab1

replace github.com/godexture/sdk => github.com/godexture/godec/pkg v0.0.0-20260621142744-bd77e78cfab1

replace github.com/godexture/metadata-id3 => github.com/godexture/godec/plugins/metadata-id3 v0.0.0-20260621142744-bd77e78cfab1
