# Local docking scenario snapshot

This bundle captures the docking feature, its supporting documents and SVG, and
its native `.opd` fixture from the live unversioned scenario directory. The
local Git commit is `028492efd6fad2a4fb740048e4810c7c6e29234a`. It is not an
upstream commit. The bundle SHA-256 is
`262bebad79d74369a267b1e05c4715731a2a5bca191f4ba1db6b5bd1e9fe3111`.

The snapshot tool checked the selected file hashes before and after the copy.
It excludes unrelated features, reference images, blockers, and non-native CAD
fixtures. The first full-source attempts detected unrelated files changing
while the scenario agent worked. The feature-scoped capture then passed.

This bundle does not replace the global pinned scenario snapshot or make a
receipt verified. The sealed diagnostic still requires integration with a
feature-scoped inventory, and build and runtime provenance remain unproven
under issue #3978.
