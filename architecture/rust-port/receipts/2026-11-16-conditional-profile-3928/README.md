<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Conditional profile input mapping

The host adapter uses the region input converter to attribute every converted
line and vertex to the current conditional sketch root. The focused adapter
unit tests cover transitive authored coincidence, distinct points with equal
coordinates, construction exclusion, unselected lines, missing or invalid
bounds, and refusal of curved input. A separate test uses a current canonical
origin projection and its conditional root. `scoped.log` records the full
adapter library and `sketch_prism` test runs on this change.

A wider `profile_regions` run failed the curved extrusion face comparison.
The same case failed on the clean base commit `cb5cdc56b` in a separate Cargo
target; `base-curved-failure.log` retains the result. The observed face area
and centroid match the open Martensite surface integration issue #407. The
mapping is conditional evidence only; it does not resolve that kernel issue,
prove region admission or qualify extrusion. The full gate was not run.
