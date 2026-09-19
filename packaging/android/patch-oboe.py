#!/usr/bin/env python3
"""Adapt Oto alpha.8's bundled Oboe declarations to NDK 30 (private copy only)."""
from pathlib import Path
import sys

source, destination = (Path(p) / "internal/oboe" for p in sys.argv[1:])
changes = {
    "oboe_aaudio_AAudioLoader_android.h": (
        "#if OBOE_USING_NDK && __NDK_MAJOR__ <= 30",
        "#if OBOE_USING_NDK && __NDK_MAJOR__ < 30",
    ),
    "oboe_aaudio_AAudioLoader_android.cpp": (
        "ASSERT_INT32(AAudio_DeviceType);",
        'static_assert(sizeof(AAudio_DeviceType) == sizeof(int32_t), "AAudio_DeviceType ABI");',
    ),
}
for filename, (old, new) in changes.items():
    text = (source / filename).read_text()
    if text.count(old) != 1:
        raise SystemExit(f"Oboe patch no longer matches {filename}")
    (destination / filename).write_text(text.replace(old, new))
