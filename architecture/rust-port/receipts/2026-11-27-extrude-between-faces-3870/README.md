# Extrude a midplane circle between two faces

The published part has a 20 mm block and a 3 mm radius circle on its fixed
10 mm midplane. The circle's center in sketch coordinates is
`[-2, -1.25] cm`. On Martensite v0.14.2 at `d4635266`, selecting Sketch2
and starting Extrude stops at profile certification on both
[Wayland](wayland/junit.xml) and [X11](x11/junit.xml). The face picks have
yet to run on the published fixture.

The [standalone kernel probe](region_probe.rs) invokes
`martensite_tkbo::profile_extrusion::build_planar_regions` with the same
model-relative resolution. It reproduces
`Joins: PairUnavailable { sources: [RegionSourceId(2), RegionSourceId(4)] }`
for the 3 mm circle at the fixture placement. The 3 mm circle at the
world origin and the 2.5 mm controls certify with the expected analytic
area. See the [probe output](region-probe.txt).

Martensite needs to certify the one bounded region with its analytic area
at both placements. This reproduces the translated-circle defect tracked
in [Martensite #619](https://github.com/Oblikovati/Martensite/issues/619).
The published input is unchanged.
