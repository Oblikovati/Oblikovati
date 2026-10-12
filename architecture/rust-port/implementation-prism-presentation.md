# Retained prism and kernel mesh presentation — partial K02/K04/R1V

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563) and
[#3630](https://github.com/Oblikovati/Oblikovati/issues/3630), under
[ADR-0007](../decisions/ADR-0007-martensite-integration.md) and
[ADR-0008](../decisions/ADR-0008-vulkan-renderer.md).

## Retained analytic source

The planar prism diagnostic now retains a private copied B-rep with copied face
roles and an owned analytic report. Copy history must map each source face to
exactly one copied face. The shared geometric checker and complete role coverage
check run on each copy. Native handles stay inside the Martensite adapter, and
the snapshot cannot move to another thread. The copy uses the same geometry and
mesh copy boundary as the retained box diagnostic.

Tests check that detached roots, faces, and surface geometry do not share native
identity with the source. Analytic reports are equal before and after copying.
Construction roles are diagnostic lineage, not persistent feature reference
keys. This step does not claim the full immutable operation result contract.

## Derived display geometry

Presentation uses the published kernel's `BRepMeshIncrementalMesh` on another
detached copy. Chord and angular approximation limits are explicit finite
positive inputs. They do not change modeling tolerances. Incomplete meshing or
nonzero defect flags produce a named error. Every face needs a nonempty kernel
triangulation. Kernel locations transform mesh nodes into working coordinates;
reversed face orientation reverses triangle winding. Internal and external
orientations are refused.

The owned presentation contains finite triangle coordinates and the construction
role of each face. The viewport converts those coordinates to its display
precision and refuses values that cannot be finite display floats. Existing
scene validation checks background and face colors. No application triangulator,
containment test, coordinate repair, or alternate renderer was added.

The retained source faces have no triangulation before or after presentation.
Tests recompute analytic values from that source after meshing; the cached report
alone cannot pass this check. All 27 cases in the independent OCCT prism corpus
also compare each derived face area with its analytic area. That comparison is a
test of display conversion for these planar faces, not a modeling predicate.

## Native Linux check

The native viewport test constructs a prism with an inner loop through the same
adapter, submits its kernel mesh, and views it from above. It checks green solid
pixels, blue background pixels through the hole, and exact agreement between
completed GPU capture and actual X11 drawable pixels. It also checks that the
analytic report stays unchanged and the kernel reports no Vulkan validation
errors. The local GPU is AMD Radeon 8060S Graphics with RADV STRIX_HALO.

```sh
cargo test --offline --locked -p oblikovati-martensite
cargo test --offline --locked -p oblikovati-viewport
CARGO_NET_OFFLINE=true make gate
```

All checks are local to this Linux host. This check is a diagnostic scene, not
an editable model feature or a foreground MCPBridge acceptance run. R1 and R1V
stay open for complete validity, tolerance budgets, persistent naming, feature
evaluation and replay, model-driven scene publication, picks, and shared Dear
ImGui composition. Required legacy Go kernel capabilities still belong in
Martensite. Windows and macOS remain deferred, with their pipelines unchanged.
