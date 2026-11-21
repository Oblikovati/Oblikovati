<!-- SPDX-License-Identifier: GPL-2.0-only -->

# Go sketch center-point fixture

`generate.go` ran from Go application commit
`aa8297d1b0ac33c024f19e54821536d76e8a447a`. It saved and reopened a part
with marked and unmarked standalone points and line-owned defining points.
`observations.json` records the IDs, center-point roles and ownership from the
reopened Go sketch. The Rust candidate test compares these roles and ownership
with a native sketch recipe round trip.

This covers candidate conversion only. Normal legacy document opening and
Go-format saving remain open (#3882). The full gate was not run.
