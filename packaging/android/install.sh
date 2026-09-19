#!/usr/bin/env bash
set -euo pipefail
GORO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
APK="$GORO_ROOT/dist/android/goro-debug.apk"
DEVICE_DATA=/sdcard/Android/data/org.goro/files
if [[ $# -gt 1 || ${1:-} == --help ]]; then
    echo "Usage: $0 [client-data-directory]"
    echo 'Build first with packaging/android/build.sh. ANDROID_SERIAL selects a device.'
    exit 0
fi
if [[ ! -f "$APK" ]]; then echo "Build the APK first: packaging/android/build.sh" >&2; exit 1; fi
adb install --no-incremental -r "$APK"
if [[ $# == 1 ]]; then
    CLIENT_DATA=$(cd "$1" && pwd)
    assets=()
    for name in data BGM System AI Emblem skin; do
        if [[ -d "$CLIENT_DATA/$name" ]]; then assets+=("$name"); fi
    done
    shopt -s nullglob
    for asset in "$CLIENT_DATA"/*.grf "$CLIENT_DATA"/*.GRF "$CLIENT_DATA"/clientinfo.xml "$CLIENT_DATA"/sclientinfo.xml; do
        if [[ -f "$asset" ]]; then assets+=("${asset##*/}"); fi
    done
    if [[ ${#assets[@]} == 0 ]]; then echo "No client assets found in $CLIENT_DATA" >&2; exit 1; fi
    # Let Android establish the app-owned external directory first.
    adb shell am start -W -n org.goro/.GoroActivity
    # Stop before replacing data that may currently be read by the game.
    adb shell am force-stop org.goro
    adb shell mkdir -p "$DEVICE_DATA"
    # The sync service cannot mkdir in scoped storage on some Android 14
    # devices, even though the shell can. Stream directly, without a second
    # copy on either device or a change to storage permissions.
    tar -C "$CLIENT_DATA" -cf - "${assets[@]}" | adb shell "tar -xof - -C $DEVICE_DATA"
fi
adb shell am force-stop org.goro
adb shell am start -W -n org.goro/.GoroActivity
