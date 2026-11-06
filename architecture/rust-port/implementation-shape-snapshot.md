# Retained kernel shape isolation — partial K01/R1

Tracking: [#3563](https://github.com/Oblikovati/Oblikovati/issues/3563).
The required boundary remains [ADR-0007](../decisions/ADR-0007-martensite-integration.md).
This document records a release-probe ownership check. R1 remains open.

## Detached ownership

The application adapter now retains the box release probe as an opaque
`ReleaseProbeSnapshot`. It uses the published Martensite v0.1.2 source at
`19108f6650cdfca7050febc759d91bdfec001769`. Kernel algorithms remain in Martensite.
The active sibling kernel checkout is unchanged.

A native `Shape` clone shares mutable TShape and geometry handles. The snapshot
therefore uses `BRepBuilderAPICopy` with geometry and mesh copying enabled, then
checks the copied shape with the existing geometric checker. It exposes only
owned analytic properties and an explicit detached-copy operation. It exposes
no native shape handle or mutation callback. A non-Send marker retains owner
thread affinity until the required cross-thread contract is proved.

The old scalar `probe_box_release` entry point now queries this same snapshot
path. It does not create another evaluation engine. The adapter refuses null
inputs, kernel construction/validation errors, and nonfinite analytic volume or
centroid output. Volume integration explicitly disables the mesh path.

## Isolation and independent comparison

The tests detach all six face surfaces and all twelve unique edge curves. They
mutate the original surfaces and curves and check that the copy's values remain
unchanged. Separate tests mutate source face tolerance and a vertex position,
then verify isolation of copied attributes and points. Root and subshape handle
inequality is used only to check aliasing. Pointer values are not model IDs.

Location, reversed orientation, unique topology counts, signed analytic volume,
and centroid are compared against six OCCT 8.0.0 cases. They use three working
scales and both orientations, with a nonzero translation. The retained C++
generator is `crates/martensite/tests/snapshot/occt_oracle.cpp`; its CSV is
`occt-8.0.0.csv` in the same directory. The installed OCCT library reports 8.0.0.
That version is not a source-commit certification of the installed binaries.

The local generator command is:

```sh
g++ -std=c++17 -I../occt-install/include/opencascade \
  crates/martensite/tests/snapshot/occt_oracle.cpp \
  -L../occt-install/lib -Wl,-rpath-link,../occt-install/lib -lTKPrim -lTKTopAlgo -lTKBRep \
  -lTKGeomBase -lTKG3d -lTKMath -lTKernel \
  -o /tmp/oblikovati-snapshot-occt-oracle
LD_LIBRARY_PATH=../occt-install/lib /tmp/oblikovati-snapshot-occt-oracle
cargo test --locked -p oblikovati-martensite
make gate
```

Rust tests hard-assert the retained oracle rows, topology counts, and relative
numeric bounds. Existing invalid-input and below-resolution cases still fail
before retention. An extreme parameter case verifies the kernel's own named
construction refusal. Synthetic nonfinite property outputs separately verify
the result gate. All checks remain on this Linux host.

## Remaining requirements

This is a diagnostic box snapshot. It is not a production feature-result
contract. It does not prove immutable ownership for every supported geometry,
cached mesh isolation, general profiles with holes, per-face operation result
certification, complete validity levels, measured tolerance budgets, or
persistent provenance/naming. These remain K01-K05 and R1 requirements. No R1
milestone or kernel capability row is closed here.

The snapshot is not a body-feature recipe and is not displayed by the head.
Model feature evaluation and viewport publication still need the full result
contract. Required Go-kernel behavior outside OCCT still belongs in Martensite;
this copy boundary cannot replace those extensions.
