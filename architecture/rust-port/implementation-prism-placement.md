# Model-plane prism integration — partial K01/K02/R1V

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563) and
[#3630](https://github.com/Oblikovati/Oblikovati/issues/3630), under
[ADR-0007](../decisions/ADR-0007-martensite-integration.md) and
[ADR-0008](../decisions/ADR-0008-vulkan-renderer.md).

## One construction path

`retain_prism_on_plane` accepts the model's validated `SketchPlane` and the
existing ordered line profile. Each point uses the model's `to_model` mapping.
Displacement components use the plane's X, Y and derived normal axes. The
adapter checks finite transformed displacement before kernel construction.
The existing XY entry point calls this same path with `SketchPlane::xy()`.
Profile, prism, validity, construction lineage, copy, and meshing operations
remain owned by the published Martensite release.

The adapter consumes the model frame without a second frame DTO or validation
policy. It does not silently normalize axes accepted by the model's metric
policy. A displacement has no origin; its conversion never subtracts two
placed points. That subtraction would lose a small displacement at a large
origin. Coordinate overflow and native collapsed segments remain named errors,
rather than a reason to move or repair the input.

This integrates the authored plane value. It does not select a sketch, resolve
a work-plane reference, approve numerical readiness, extract a closed profile,
or publish a body feature. Those tasks still need their document owners and
the complete feature result contract.

## Independent geometry and mesh checks

The retained OCCT 8.0.0 generator now has a `--placed` mode. Its original mode
reproduces the existing 216-row XY corpus byte for byte. The new mode generates
1,080 face rows for 135 prisms across five frames: translated XY, translated YZ,
tilted, reversed normal, and small axis deviations accepted by the model.
Each frame covers rectangular, holed and concave profiles, three working
scales, and positive, negative and skew displacement.

Rust checks each analytic face area, centroid and loop count, plus body volume
and centroid, against that independent corpus. Each placed case also checks
the derived mesh area for every construction face. A separate test requires
the mesh to retain a vertex calculated from the authored axes, and tests
detached report equality. The default and explicit XY paths must return equal
reports and equal presentation coordinates.

Tests also cover displacement precision at a large origin, transformed-vector
overflow, placed-point overflow, and native loss of distinct input segments.
These errors do not produce a diagnostic body.

```sh
g++ -std=c++17 -I../occt-install/include/opencascade \
  crates/martensite/tests/prism/occt_oracle.cpp \
  -L../occt-install/lib -Wl,-rpath-link,../occt-install/lib \
  -lTKPrim -lTKTopAlgo -lTKBRep -lTKGeomBase -lTKG3d -lTKMath -lTKernel \
  -o /tmp/oblikovati-prism-plane-oracle
LD_LIBRARY_PATH=../occt-install/lib /tmp/oblikovati-prism-plane-oracle --placed
cargo test --offline --locked -p oblikovati-martensite
cargo test --offline --locked -p oblikovati-viewport --test native
CARGO_NET_OFFLINE=true make gate
```

## Native Linux display

The native viewport check replaces the diagnostic scene on one owned window
for XY, XZ, YZ, tilted translated, and flipped tilted frames. Each camera uses
the same model plane. Green solid pixels and blue pixels through the hole must
match the actual X11 drawable and the completed GPU capture. Analytic reports
remain unchanged, and kernel Vulkan validation must report no errors.

All checks stay on this Linux host. This adds no application geometry fallback,
no kernel source change, and no Windows or macOS test. R1, R1V and R3 remain
open for curved profiles, general feature evaluation, persistent references,
full validity and tolerance contracts, editable parts, and native selection.

The next [checked sketch consumer](implementation-sketch-prism-consumer.md)
connects selected authored line boundaries and numerical readiness to this path.
It also extends the native diagnostic scene to a parameter edit and recompute.
