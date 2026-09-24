#!/usr/bin/env bash
set -euo pipefail
GORO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$GORO_ROOT"
SDK=${ANDROID_SDK_ROOT:-${ANDROID_HOME:-$HOME/Android/Sdk}}
NDK=${ANDROID_NDK_HOME:-$(find "$SDK/ndk" -mindepth 1 -maxdepth 1 -type d | sort -V | tail -1)}
BUILD_TOOLS=${ANDROID_BUILD_TOOLS:-$(find "$SDK/build-tools" -mindepth 1 -maxdepth 1 -type d | sort -V | tail -1)}
ANDROID_JAR="$SDK/platforms/android-35/android.jar"
OUT="$GORO_ROOT/dist/android"
JAVAC=${JAVA_HOME:+$JAVA_HOME/bin/}javac
KEYTOOL=${JAVA_HOME:+$JAVA_HOME/bin/}keytool
for tool in "$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android29-clang" "$BUILD_TOOLS/aapt2" "$BUILD_TOOLS/d8" "$ANDROID_JAR"; do
    if [[ ! -f "$tool" ]]; then echo "Missing Android build dependency: $tool" >&2; exit 1; fi
done
mkdir -p "$OUT"/{lib/arm64-v8a,classes,java,res/drawable,dex,overlay}
export GOCACHE=${GOCACHE:-/tmp/goro-go-cache}
# The Android patches need dependency sources before go build can download them.
go mod download github.com/gogpu/gogpu github.com/ebitengine/oto/v3
GOGPU_DIR=$(go list -m -f '{{.Dir}}' github.com/gogpu/gogpu)
GOGPU_VERSION=$(go list -m -f '{{.Version}}' github.com/gogpu/gogpu)
if [[ "$GOGPU_VERSION" != v0.54.0 ]]; then
    echo "The Android platform overlay needs review for GoGPU $GOGPU_VERSION (expected v0.54.0)." >&2; exit 1
fi
# Go forbids overlays inside its immutable module cache. Work from a private
# copy and a temporary modfile; the project's desktop module graph is untouched.
if [[ ! -d "$OUT/gogpu-$GOGPU_VERSION" ]]; then
    cp -a "$GOGPU_DIR" "$OUT/gogpu-$GOGPU_VERSION"
fi
cp go.mod "$OUT/android.mod"
cp go.sum "$OUT/android.sum"
go mod edit -modfile "$OUT/android.mod" -replace "github.com/gogpu/gogpu=$OUT/gogpu-$GOGPU_VERSION"
python3 packaging/android/overlay.py "$OUT/gogpu-$GOGPU_VERSION" "$OUT/overlay"
# Oto's bundled Oboe predates the final NDK 30 AAudio declarations. Patch
# only a private copy when those declarations are present in the selected NDK.
AAUDIO_HEADER="$NDK/toolchains/llvm/prebuilt/linux-x86_64/sysroot/usr/include/aaudio/AAudio.h"
if rg -q 'typedef enum AAudio_FallbackMode' "$AAUDIO_HEADER"; then
    OTO_DIR=$(go list -m -f '{{.Dir}}' github.com/ebitengine/oto/v3)
    OTO_VERSION=$(go list -m -f '{{.Version}}' github.com/ebitengine/oto/v3)
    if [[ "$OTO_VERSION" != v3.5.0-alpha.8 ]]; then
        echo "Review the NDK 30 audio compatibility patch for Oto $OTO_VERSION." >&2; exit 1
    fi
    if [[ ! -d "$OUT/oto-$OTO_VERSION" ]]; then
        cp -a "$OTO_DIR" "$OUT/oto-$OTO_VERSION"
        chmod -R u+w "$OUT/oto-$OTO_VERSION"
    fi
    python3 packaging/android/patch-oboe.py "$OTO_DIR" "$OUT/oto-$OTO_VERSION"
    go mod edit -modfile "$OUT/android.mod" -replace "github.com/ebitengine/oto/v3=$OUT/oto-$OTO_VERSION"
fi
export CC="$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android29-clang"
export CXX="$NDK/toolchains/llvm/prebuilt/linux-x86_64/bin/aarch64-linux-android29-clang++"
GOOS=android GOARCH=arm64 CGO_ENABLED=1 go build \
    -modfile "$OUT/android.mod" -overlay "$OUT/overlay/overlay.json" -buildmode=c-shared \
    -ldflags='-s -w -extldflags=-Wl,-z,max-page-size=16384' \
    -o "$OUT/lib/arm64-v8a/libgoro.so" ./cmd/goro-android
cp internal/appicon/icon.png "$OUT/res/drawable/icon.png"
"$BUILD_TOOLS/aapt2" compile --dir "$OUT/res" -o "$OUT/resources.zip"
"$BUILD_TOOLS/aapt2" link -o "$OUT/resources.apk" -I "$ANDROID_JAR" \
    --manifest packaging/android/AndroidManifest.xml --java "$OUT/java" "$OUT/resources.zip"
mapfile -t JAVA_SOURCES < <(find packaging/android/java "$OUT/java" -name '*.java')
"$JAVAC" -encoding UTF-8 --release 8 -classpath "$ANDROID_JAR" -d "$OUT/classes" "${JAVA_SOURCES[@]}"
mapfile -t CLASSES < <(find "$OUT/classes" -name '*.class')
"$BUILD_TOOLS/d8" --min-api 29 --lib "$ANDROID_JAR" --output "$OUT/dex" "${CLASSES[@]}"
cp "$OUT/resources.apk" "$OUT/unsigned.apk"
python3 - "$OUT" <<'PY'
import pathlib, sys, zipfile
root = pathlib.Path(sys.argv[1])
with zipfile.ZipFile(root / 'unsigned.apk', 'a', compression=zipfile.ZIP_DEFLATED) as apk:
    apk.write(root / 'dex/classes.dex', 'classes.dex')
    apk.write(root / 'lib/arm64-v8a/libgoro.so', 'lib/arm64-v8a/libgoro.so')
PY
"$BUILD_TOOLS/zipalign" -f -P 16 4 "$OUT/unsigned.apk" "$OUT/aligned.apk"
if [[ ! -f "$OUT/debug.keystore" ]]; then
    "$KEYTOOL" -genkeypair -keystore "$OUT/debug.keystore" -storepass android -keypass android \
        -alias androiddebugkey -keyalg RSA -keysize 2048 -validity 10000 -dname 'CN=Android Debug,O=Goro,C=US'
fi
"$BUILD_TOOLS/apksigner" sign --ks "$OUT/debug.keystore" --ks-pass pass:android \
    --out "$OUT/goro-debug.apk" "$OUT/aligned.apk"
"$BUILD_TOOLS/apksigner" verify "$OUT/goro-debug.apk"
echo "Built $OUT/goro-debug.apk"
