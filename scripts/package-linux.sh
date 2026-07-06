#!/usr/bin/env bash
# Package the Oblikovati GUI head as a Linux AppImage.
#
# The head links GLFW + the Vulkan loader (libvulkan.so.1); linuxdeploy bundles those
# plus the X11/Wayland deps from the binary's DT_NEEDED entries. The GPU's Vulkan ICD
# (driver) is NOT bundled — it is provided by the user's system at runtime, which is
# the standard arrangement for Vulkan AppImages.
#
#   Usage: scripts/package-linux.sh <head-binary> <version> <out-dir>
# Requires on PATH: linuxdeploy, imagemagick (convert), and APPIMAGE_EXTRACT_AND_RUN=1
# on CI runners without FUSE.
set -euo pipefail

bin="$1"
version="$2"
out="${3:-dist}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

work="$(mktemp -d)"
appdir="$work/Oblikovati.AppDir"
mkdir -p "$appdir/usr/bin"
cp "$bin" "$appdir/usr/bin/oblikovati-head"
chmod +x "$appdir/usr/bin/oblikovati-head"

# App icon: render the brand mark from the source SVG (head/cmd/genappicon) so the
# AppImage + .desktop entry carry the real Oblikovati icon. linuxdeploy requires an icon
# file; 256px is the standard AppImage app-icon size.
repo="$(cd "$(dirname "$0")/.." && pwd)"
icon="$work/oblikovati.png"
go -C "$repo/head" run ./cmd/genappicon -format png -size 256 -out "$icon"

cat >"$work/oblikovati.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Name=Oblikovati
Exec=oblikovati-head
Icon=oblikovati
Categories=Graphics;Engineering;
Terminal=false
EOF

export VERSION="$version"
( cd "$work" && linuxdeploy \
	--appdir "$appdir" \
	--executable "$appdir/usr/bin/oblikovati-head" \
	--desktop-file "$work/oblikovati.desktop" \
	--icon-file "$icon" \
	--output appimage )

mv "$work"/*.AppImage "$out/oblikovati-$version-linux-amd64.AppImage"
echo "built $out/oblikovati-$version-linux-amd64.AppImage"
