# Conic extrema minimum flags prevent a visible ellipse from being picked

## Reproduce

SDK 0.11.0 commit: `4141137449d911884c9a3fd261f7913498548e90`.
The same rotated-ellipse pick failure also occurs with SDK 0.13.0 commit
`d051ff699982faba4ec2df3ee9a47e14f38dedbd`.

Run the raw SDK reproducer in this host worktree:

```sh
cargo run --locked -p oblikovati-martensite --example conic_extrema_flags
```

It emits each root, stored minimum flag, squared-distance second variation,
and expected geometric classification. It currently exits with six mismatches
among ten nondegenerate stationary roots. No window or host feature is involved.
The example uses only SDK conics, `ExtPElC2d`, and `elclib::d2`.

The public host regression is:

```sh
cargo test --locked -p oblikovati-martensite --test planar_pick \
  a_nearby_rotated_ellipse_pick_finds_the_interior_minimum
```

## Minimal ellipse and independent oracle

```text
center = [2.7574, -6.9211]
major axis = [0.6, 0.8], counterclockwise = true
radii = [0.75793, 0.38063], full native period
cursor = [2.9111111111111105, -6.288888888888888]
```

The host discards the actual minimum and returns the period boundary:
parameter 0, point `[3.212158, -6.314756]`, distance 0.3021561463000778.
A native mouse click on the visible ellipse consequently leaves Measure empty.

OCCT DRAW `2dproj` on this exact 2D ellipse reports distance
0.019024451883025126. The same ellipse lifted into Z=0 and queried by `proj`
reports parameter **0.68281751332396734**, with foot
`[2.918064003780093, -6.306597278227931]`. Its other stationary point has
parameter 3.2519998883927697 and distance 1.3838430950249225.

## Actual raw SDK flag cases

All values below were measured on SDK 0.11.0 by the reproducer, not inferred
from root order. Lengths are in the same working units as the input geometry.

| Conic and query | Root parameter | Distance | Stored minimum | Geometric minimum |
| --- | ---: | ---: | --- | --- |
| Circle r=1, query (2,0) | pi | 3 | true | false |
| Circle r=1, query (2,0) | 0 | 1 | true | true |
| Rotated ellipse above | 0.6828175133239673 | 0.019024451883025126 | false | true |
| Rotated ellipse above | 3.2519998883927697 | 1.3838430950249225 | false | false |
| Hyperbola a=2,b=1, query (3,0) | -0.6223625037147787 | 0.8944271909999161 | false | true |
| Same hyperbola | 0.6223625037147789 | 0.8944271909999157 | false | true |
| Same hyperbola | 0 | 1 | false | false |
| Parabola focal=1, query (3,0) | 2 | 2.8284271247461903 | false | true |
| Same parabola | -2 | 2.8284271247461903 | false | true |
| Same parabola | 0 | 3 | false | false |

Hyperbola bounds are [-2,2]; parabola bounds are [-4,4]. Both use origin (0,0)
and X axis (1,0). Every listed second variation is separated from zero.

## Root cause and required invariant

`crates/martensite-tkgeombase/src/extrema/ext_p_el_c2d.rs` marks both circle
extrema as minima, while ellipse, hyperbola, and parabola roots get false.
The ellipse comment explicitly preserves OCCT's old `NoSol == 0` condition,
which never holds in its one-based root loop. Generic
`extrema/point_curve/ext_pc/mod.rs` copies those flags into `ExtPC2d`.

OCCT's `2dproj` does not filter these flags. The host's general analytic-pick
pipeline does filter `ExtPC2d::is_min` before its stationarity checks, so it
loses every ellipse minimum. The existing axis-aligned ellipse test did not
expose the defect because its correct minimum also happened to be the period
boundary.

Classify every root against its geometry in one general mechanism. At a
stationary parameter u, half the squared-distance second derivative is
`C'(u) dot C'(u) + (C(u)-Q) dot C''(u)`. Certify the branch rather than assigning
a physical meaning from its root index or conic kind. Preserve positions,
parameters, distances, root completeness, and existing valid classifications.
Handle a degenerate classification explicitly; do not add an ellipse-only
host fallback, move the pointer fixture, or select the first root.

Logs: `/tmp/oblikovati-conic-extrema-flags.log`,
`/tmp/oblikovati-ellipse-pick-red.log`,
`/tmp/oblikovati-ellipse-pick-occt.log`, and
`/tmp/oblikovati-ellipse-pick-occt-parameter.log`.
