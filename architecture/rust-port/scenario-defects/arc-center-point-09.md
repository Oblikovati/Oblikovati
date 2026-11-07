# Arc center point 09: inconsistent recorded measurements

## Source

The source is
`/mnt/vininas-projects/MCAD-Scenarios/features/sketch/arc-center-point/scenario.feature`,
scenario `arc-center-point-09 Measure the restored curve instead of its middle point`.
Its SHA-256 at the test run was:

```text
e91609c2b73a7190934cc553891fe27fba08837f1e90286c802e7861fbd5142a
```

The two steps at source lines 52 and 53 require radius 4.759 mm, diameter
9.518 mm, length 7.037 mm, and sweep 81.72 degrees for the same circular arc.

## Invariant

A circular arc has length `L = r * theta`, with theta in radians. The reported
radius and length give:

```text
7.037 / 4.759 * 180 / pi = 84.72166430627449 degrees
```

Even when each displayed millimeter value has rounding uncertainty of 0.0005 mm,
the possible sweep is 84.70674494039369 to 84.73658680746387 degrees. The required
81.72 degrees is outside this interval.

## Native result

The focused native run executed all ten current center-point arc scenarios.
Nine passed. Scenario 09 alone failed, at the sweep assertion. The Measure panel
showed radius 4.759 mm, diameter 9.518 mm, length 7.037 mm, and sweep 84.72 degrees.
Center X 0.000, Y -7.023, Z -83.568 mm matched the source.

The test fixture uses the recorded radius and length to construct its circular
arc through the real tool. Thus its sweep agrees with the circular invariant.
There is no independent recording in this lane that can decide which source
number was transcribed incorrectly. Correct the source from its recording;
do not change the measurement code to return mutually inconsistent values.

The test log is `/tmp/oblikovati-sketch-center-arc-baseline.log`. Native screenshots
are under this worktree's `target/benchmarks/application/`.

## Implementation references

The Go reference uses one analytic circular recipe and one measured curve:
`Oblikovati/app/sketch_variant_tools.go:133` and
`Oblikovati/model/sketch/recipe_curve.go`. The Rust host uses its existing
center-point arc placement and Measure paths; no special case was added for
this scenario. The failing assertion remains active.
