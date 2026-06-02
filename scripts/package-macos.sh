#!/usr/bin/env bash
# Package the Oblikovati GUI head for macOS (UNSIGNED — no Apple cert yet, so users
# right-click → Open the first time to clear Gatekeeper).
#
# macOS has no native Vulkan, so we bundle the Vulkan loader + MoltenVK (Vulkan over
# Metal) and a launcher that points the loader at them. Homebrew-linked dylibs carry
# absolute install names, so we rewrite the head's references to @rpath and ship the
# libs alongside it.
#
#   Usage: scripts/package-macos.sh <head-binary> <version> <arch> <out-dir>
# Requires: brew packages glfw, vulkan-loader, molten-vk; install_name_tool (Xcode CLT).
set -euo pipefail

bin="$1"
version="$2"
arch="$3"
out="${4:-dist}"
mkdir -p "$out"
out="$(cd "$out" && pwd)"

stage="$(mktemp -d)/oblikovati"
mkdir -p "$stage/bin" "$stage/lib" "$stage/share/vulkan/icd.d"
cp "$bin" "$stage/bin/oblikovati-head"

brewp="$(brew --prefix)"
# Bundle the Vulkan loader, MoltenVK, and GLFW dylibs.
cp "$brewp"/lib/libvulkan.1*.dylib "$stage/lib/" 2>/dev/null || cp "$brewp"/lib/libvulkan*.dylib "$stage/lib/"
cp "$(brew --prefix molten-vk)/lib/libMoltenVK.dylib" "$stage/lib/"
cp "$(brew --prefix glfw)"/lib/libglfw*.dylib "$stage/lib/" 2>/dev/null || true

# ICD manifest pointing the loader at the bundled MoltenVK (path relative to the json).
cat >"$stage/share/vulkan/icd.d/MoltenVK_icd.json" <<'EOF'
{ "file_format_version": "1.0.0", "ICD": { "library_path": "../../../lib/libMoltenVK.dylib", "api_version": "1.2.0" } }
EOF

# Rewrite the head's absolute brew dylib references to @rpath and add an rpath to lib/.
fix_ref() {
	local dep; dep="$(otool -L "$stage/bin/oblikovati-head" | awk '/'"$1"'/{print $1; exit}')"
	[ -n "$dep" ] && install_name_tool -change "$dep" "@rpath/$(basename "$dep")" "$stage/bin/oblikovati-head" || true
}
fix_ref libvulkan
fix_ref libglfw
install_name_tool -add_rpath "@loader_path/../lib" "$stage/bin/oblikovati-head" 2>/dev/null || true

# Launcher: point the loader at the bundled ICD, then exec the head.
cat >"$stage/oblikovati-head" <<'EOF'
#!/usr/bin/env bash
here="$(cd "$(dirname "$0")" && pwd)"
export VK_ICD_FILENAMES="$here/share/vulkan/icd.d/MoltenVK_icd.json"
export DYLD_LIBRARY_PATH="$here/lib:${DYLD_LIBRARY_PATH:-}"
exec "$here/bin/oblikovati-head" "$@"
EOF
chmod +x "$stage/oblikovati-head"

tar -C "$(dirname "$stage")" -czf "$out/oblikovati-head-$version-darwin-$arch.tar.gz" oblikovati
echo "built $out/oblikovati-head-$version-darwin-$arch.tar.gz"
