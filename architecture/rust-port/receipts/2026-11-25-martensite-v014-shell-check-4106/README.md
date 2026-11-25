# Shell tolerance on the Martensite v0.14.0 tag

The v0.14.0 tag resolves to `b2a6a5a2d56f0714fabe51e91584cfa6dbbb29c8`.
The published 20 mm cube reproduction still widens the maximum vertex tolerance
for a 1 mm inward thick solid at the requested `1e-7 mm` tolerance.
[Martensite issue #554](https://github.com/Oblikovati/Martensite/issues/554)

| Shape | Maximum vertex tolerance in mm |
| --- | ---: |
| Source cube | `1.00000000000000022e-7` |
| Shell result | `1.99999999999999991e-7` |

The [probe source](shell-tolerance-probe.rs) ran in a disposable Cargo package
using path dependencies from a Git archive of the tagged tree. The tag's lock
file seeded the offline resolution. The [probe manifest](probe-Cargo.toml.txt)
uses `martensite-v014-source` as the sibling directory containing that archive.
The [result](result.txt) records the assertion failure.

Oblikovati's native runs use application pin `bef25f47548431c8f128f08f66a1f719bcee4982`.
Its [Ribbon Docking probe](../2026-11-25-ribbon-docking-current-3860/README.md)
records the refusal on the published application fixture. Issue #4106 tracks
Shell support in the application.
