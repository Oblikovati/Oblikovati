<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Go sketch line-role fixture

`generate.go` ran from Go application commit
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. It created a sketch with four
lines covering every combination of the construction and centerline flags,
saved `line-roles.obk`, and reopened it in Go. `observations.json` records the
reopened IDs, effective construction role, centerline role, and construction
role after clearing centerline. The Rust persistence test compares its imported
candidate against these observations and checks native recipe round trips.

This covers candidate conversion only. Normal legacy document open and Go-format
save remain unavailable. The profile-order witness and solve precision needed
for legacy extrusion remain open (#3882, #3928). The full gate was not run.
