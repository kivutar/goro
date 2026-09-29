// Package scripts provides optional controls bundled in desktop and APK builds.
package scripts

import "embed"

//go:embed wasd.lua
var Builtin embed.FS
