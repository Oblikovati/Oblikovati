#!/usr/bin/env bash
# Stage one architecture's macOS head build for the universal-bundle assembly.
#
# The universal .app is lipo'd from both runners' outputs, so each runner uploads its
# binary plus the exact dylibs it links against (Vulkan loader + GLFW) and MoltenVK
# (dlopened by the loader, not a link-time dep). Symlinks are dereferenced and copied
# under canonical names so the two arches' files line up by name for `lipo -create`.
#
#   Usage: scripts/macos-stage.sh <head-binary> <out-staging-dir>
# Requires: brew with glfw, vulkan-loader, molten-vk.
set -euo pipefail

bin="$1"
out="$2"
mkdir -p "$out/bin" "$out/frameworks"

cp "$bin" "$out/bin/oblikovati-head"

# -L dereferences the brew version symlinks (libvulkan.1.dylib -> libvulkan.1.4.350.dylib)
# so the staged file keeps the canonical name the binary links against.
cp -L "$(brew --prefix vulkan-loader)/lib/libvulkan.1.dylib" "$out/frameworks/"
cp -L "$(brew --prefix molten-vk)/lib/libMoltenVK.dylib" "$out/frameworks/"
cp -L "$(brew --prefix glfw)"/lib/libglfw.3.dylib "$out/frameworks/"

echo "staged $(uname -m) head + dylibs in $out"
ls -1 "$out/frameworks"
