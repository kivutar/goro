#!/usr/bin/env python3
"""Add the Android host to the pinned GoGPU version without changing Go's cache."""
import json
import pathlib
import sys

source = pathlib.Path(__file__).resolve().parent / "gogpu"
module = pathlib.Path(sys.argv[1]).resolve()
output = pathlib.Path(sys.argv[2]).resolve()
output.mkdir(parents=True, exist_ok=True)
# Keep generated package fragments out of the root module's ./... traversal.
(output / "go.mod").write_text("module goro-android-overlay\n\ngo 1.26.4\n")
replace = {}
for original in (module / "internal/platform").glob("*_linux.go"):
    target = output / original.name
    target.write_text("//go:build !android\n\npackage platform\n")
    replace[str(original)] = str(target)
replace[str(module / "internal/platform/platform_android.go")] = str(source / "platform_android.go")
replace[str(module / "host_android.go")] = str(source / "host_android.go")
renderer = module / "renderer.go"
text = renderer.read_text()
anchor = "\t// Phase 4: Configure surface dimensions."
if text.count(anchor) != 1:
    raise SystemExit("GoGPU renderer changed; review Android surface format selection")
text = text.replace(anchor, """\tif err := r.selectAndroidSurfaceFormat(); err != nil {
\t\tr.Destroy()
\t\tr.ReleaseInstance()
\t\treturn nil, err
\t}

""" + anchor)
alpha = "alphaMode := resolveAlphaMode(caps, ws.transparent)"
if text.count(alpha) != 1:
    raise SystemExit("GoGPU renderer changed; review Android composite alpha selection")
text = text.replace(alpha, "alphaMode := resolveAndroidAlphaMode(caps)")
(output / "renderer.go").write_text(text)
replace[str(renderer)] = str(output / "renderer.go")
(output / "overlay.json").write_text(json.dumps({"Replace": replace}, indent=2) + "\n")
