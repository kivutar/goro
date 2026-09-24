# Android development build

The Android host runs the existing game and Vulkan renderer in a landscape
SurfaceView at 1280×720. It targets arm64 devices with Android 10 (API 29) or
newer and Vulkan support. The package ID is `org.goro`; local builds default
to a debug APK.

## Build and install

On Linux, install Go (the version in `go.mod`), JDK 17+, Python 3, ripgrep,
Android SDK platform 35, build-tools 35 or newer, and an Android NDK with an
`aarch64-linux-android29-clang` compiler. Then run from the repository root:

```sh
packaging/android/build.sh
packaging/android/install.sh /path/to/OldRO
```

The build defaults to `~/Android/Sdk`. Set `ANDROID_SDK_ROOT` (or
`ANDROID_HOME`), `ANDROID_NDK_HOME`, `ANDROID_BUILD_TOOLS`, and `JAVA_HOME` to
override tool locations. `ANDROID_BUILD_TOOLS` is a directory, not a version.
Set `ANDROID_SERIAL` when multiple devices are attached.

Output: `dist/android/goro-debug.apk`. The debug signing key is generated at
`dist/android/debug.keystore`; retain it to update an existing installation.
The scripts do not require Gradle or gomobile.

`install.sh` installs the APK, optionally copies the supplied client assets, then
launches Goro. Omit the data argument on subsequent installs to retain the
device's assets and settings. It never copies the desktop `goro.ini`.

## CI and release APKs

CI builds an arm64 debug APK on pushes and pull requests. Download the
`goro-android-arm64-debug` artifact from the workflow run. These builds use
temporary debug keys and cannot update an installation signed with another
key. CI checks compilation and packaging, not gameplay on an Android device.

Creating a GitHub release also builds `goro-android-arm64.apk` and attaches it
alongside the desktop binaries. Release APKs disable debugging, use the release
tag as their version name, and use the Release workflow run number as their
version code. CI and releases share the pinned SDK/NDK setup in
`.github/actions/setup-android/action.yml`.

Before the first release, configure these repository Actions secrets:

- `ANDROID_KEYSTORE_BASE64`: the base64-encoded release keystore.
- `ANDROID_KEYSTORE_PASSWORD`: the password for both the keystore and its key.

Create a keystore once:

```sh
keytool -genkeypair -keystore goro-release.p12 -storetype PKCS12 \
    -alias goro -keyalg RSA -keysize 2048 -validity 10000
```

Keep the keystore and password backed up outside the repository: future APKs
must use the same signing key to update existing installations and retain data.
The release job fails if either secret is missing; it never generates a release
key. Only the APK is uploaded.

For a local release build, set `ANDROID_BUILD_TYPE=release`, `ANDROID_KEYSTORE`
to the keystore path, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_VERSION_NAME`, and
a positive `ANDROID_VERSION_CODE`, then run `packaging/android/build.sh`.
`install.sh` continues to install the local debug APK; release APKs can be
installed with `adb install -r dist/android/goro-android-arm64.apk`.

## Client data and servers

The original client assets are separate from the APK. The app reads its GRF
archives, loose `data/` files, `BGM/`, and optional `System/`, `AI/`, `Emblem/`,
and `skin/` directories from:

```text
/sdcard/Android/data/org.goro/files/
```

Place an Android-specific `goro.ini` there if needed. Settings saved in game
also go there. App-specific external storage needs no storage permission and
is removed when the app is uninstalled. Installing with `adb install -r`
retains it.

Point `data/clientinfo.xml` at a server reachable from the device. For a server
running on the USB-connected development computer and advertising localhost,
forward its login, character, and map ports:

```sh
adb reverse tcp:6900 tcp:6900
adb reverse tcp:6121 tcp:6121
adb reverse tcp:5121 tcp:5121
```

## Controls and current limitations

- Tap/drag: left click / held movement. Two-finger drag: right drag for camera
  rotation. Pinch: mouse wheel for zoom.
- **Keyboard** opens the Android IME for the currently focused game field.
  Physical keyboards use the shared desktop key mapping.
- Gamepad A/B: left/right click at the pointer; left stick moves the pointer;
  Start sends Escape; Select opens the keyboard.
- Leaving the Activity stops the game and releases the Vulkan surface before
  Android destroys its native window. Returning starts at login again; the
  current server session is not preserved across backgrounding.
- Native clipboard and file dialogs are not implemented. The desktop UI has
  not been redesigned for touch, and this is not a Play Store release build.

Inspect native and game logs with:

```sh
adb logcat -s Goro AndroidRuntime
```

## Dependency integration

GoGPU v0.54.0 has no Android window host. `overlay.py` excludes its Linux
window implementation and injects the two small files in `gogpu/`. This uses
a private dependency copy and temporary modfile under `dist/android`; neither
the module cache nor the desktop dependency graph is changed. Review the
overlay when upgrading GoGPU; the build checks the pinned version.

The overlay also selects the swapchain format and composite alpha mode from
the surface capabilities. On the AIR X these are `RGBA8Unorm` and `Inherit`;
GoGPU's desktop defaults, `BGRA8Unorm` and `Opaque`, are not supported there.

Oto's Android backend is enabled with CGO. NDK 30's AAudio declarations overlap
with compatibility declarations in Oto v3.5.0-alpha.8's bundled Oboe. When the
selected NDK contains those declarations, `patch-oboe.py` updates their guard
and checks the device-type ABI by size in a private Oto copy. Older NDKs keep
the original sources. The desktop build remains pure Go with `nofakecgo`.

## Device verification

Verified on a MANGMI AIR X running Android 14 with an Adreno 610: APK install,
asset loading, Vulkan presentation of the server/login screens, and Activity
background/resume with surface recreation in the same process. The desktop
test suite and build also pass. This does not establish full gameplay or
compatibility with other Android GPUs.
