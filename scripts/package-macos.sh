#!/usr/bin/env bash
# Assemble the Oblikovati GUI head into a self-contained macOS .app bundle.
#
# macOS has no system Vulkan, so the bundle ships its own loader + MoltenVK (Vulkan
# over Metal) + GLFW. The result runs on a clean Mac with NO Vulkan SDK and NO
# environment setup:
#   - the binary finds its dylibs via a baked-in @rpath (@executable_path/../Frameworks),
#     not DYLD_* (which the hardened runtime that notarization requires would strip);
#   - GLFW finds the loader itself via the bundle's Frameworks dir (Cocoa bundle search);
#   - the app points the loader at the bundled MoltenVK ICD in-process at startup
#     (head/internal/native/icd_darwin.go), so no launcher and no env var are needed.
#
# This script only assembles + relinks the bundle; codesigning + notarization is a
# separate step (scripts/macos-sign.sh) so an unsigned bundle is still locally runnable.
#
#   Usage: scripts/package-macos.sh <head-binary> <frameworks-dir> <version> <out-dir>
#     <head-binary>     the oblikovati-head executable (single-arch or universal)
#     <frameworks-dir>  dir holding libvulkan.1.dylib, libMoltenVK.dylib, libglfw.*.dylib
#                       (matching arch(es) of <head-binary>)
#     <version>         release version string
#     <out-dir>         where Oblikovati.app is written
# Requires: install_name_tool, otool (Xcode Command Line Tools).
set -euo pipefail

bin="$1"
frameworks="$2"
version="$3"
out="${4:-dist}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

app="$out/Oblikovati.app"
rm -rf "$app"
macos="$app/Contents/MacOS"
fw="$app/Contents/Frameworks"
icddir="$app/Contents/Resources/vulkan/icd.d"
mkdir -p "$macos" "$fw" "$icddir"

cp "$bin" "$macos/oblikovati-head"
chmod +x "$macos/oblikovati-head"
cp "$frameworks"/*.dylib "$fw/"

# Info.plist — CFBundleExecutable wires the launch; the identifier + versions are what
# notarization and Gatekeeper record. NSHighResolutionCapable gives a crisp viewport.
cat >"$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key><string>oblikovati-head</string>
	<key>CFBundleIdentifier</key><string>com.oblikovati.head</string>
	<key>CFBundleName</key><string>Oblikovati</string>
	<key>CFBundleDisplayName</key><string>Oblikovati</string>
	<key>CFBundleIconFile</key><string>Oblikovati</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleShortVersionString</key><string>${version}</string>
	<key>CFBundleVersion</key><string>${version}</string>
	<key>LSMinimumSystemVersion</key><string>11.0</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSSupportsAutomaticGraphicsSwitching</key><true/>
</dict>
</plist>
EOF

# ICD manifest the in-process shim points the loader at. library_path is relative to the
# json: Resources/vulkan/icd.d/ -> ../../../Frameworks/libMoltenVK.dylib.
cat >"$icddir/MoltenVK_icd.json" <<'EOF'
{ "file_format_version": "1.0.0", "ICD": { "library_path": "../../../Frameworks/libMoltenVK.dylib", "api_version": "1.2.0" } }
EOF

# Relink for the bundle: give every bundled dylib an @rpath id, repoint the binary's
# linked deps that we bundle to @rpath, and add the Frameworks rpath. MoltenVK is not a
# link-time dep of the binary (the loader dlopens it via the ICD), so it needs only an id.
for f in "$fw"/*.dylib; do
	install_name_tool -id "@rpath/$(basename "$f")" "$f"
done
for dep in $(otool -L "$macos/oblikovati-head" | awk 'NR>1{print $1}'); do
	base="$(basename "$dep")"
	[ -f "$fw/$base" ] && install_name_tool -change "$dep" "@rpath/$base" "$macos/oblikovati-head"
done
otool -l "$macos/oblikovati-head" | grep -q "@executable_path/../Frameworks" \
	|| install_name_tool -add_rpath "@executable_path/../Frameworks" "$macos/oblikovati-head"

# App/dock icon: render the brand mark from the source SVG (head/cmd/genappicon) into the
# Apple .icns the Info.plist's CFBundleIconFile points at. genappicon needs Go + the head
# module (the CI job sets these up via setup-go + the api-contract action); iconutil is in
# the Xcode CLT. Best-effort: a local assembly without them still yields a runnable bundle.
repo="$(cd "$(dirname "$0")/.." && pwd)"
if command -v go >/dev/null && command -v iconutil >/dev/null; then
	iconset="$(mktemp -d)/Oblikovati.iconset"
	mkdir -p "$iconset"
	# size + canonical iconset member name (Retina @2x members are the next size up).
	while read -r size name; do
		go -C "$repo/head" run ./cmd/genappicon -format png -size "$size" -out "$iconset/$name.png"
	done <<'SIZES'
16 icon_16x16
32 icon_16x16@2x
32 icon_32x32
64 icon_32x32@2x
128 icon_128x128
256 icon_128x128@2x
256 icon_256x256
512 icon_256x256@2x
512 icon_512x512
1024 icon_512x512@2x
SIZES
	iconutil -c icns "$iconset" -o "$app/Contents/Resources/Oblikovati.icns"
	echo "icon: wrote $app/Contents/Resources/Oblikovati.icns"
else
	echo "package-macos: go or iconutil missing — skipping the app icon (bundle still runnable)"
fi

echo "assembled $app"
