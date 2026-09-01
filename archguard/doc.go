// SPDX-License-Identifier: GPL-2.0-only

// Package archguard holds repository-level architecture invariants that are enforced
// as ordinary Go tests, so a violation fails CI without shipping any code in the
// binary. It intentionally has no production symbols — the rules are the _test.go files.
//
// # The kernel ground rules (M48/C8, #2184-#2193)
//
// CLAUDE.md states the rules the kernel is built to; until 2026-09-01 none of these ten could
// fail a build, and each had drifted by a measurable amount:
//
//	tolerance_guard_test.go        no bare epsilon literal, over ALL 861 kernel files (#2189)
//	tessellation_downstream_test.go  no deciding package reads the tessellator (#2187)
//	predicate_authority_test.go    one predicate package; no rival, no re-implementation (#2193)
//	dispatch_ladder_test.go        no first-fit dispatch ladder (#2186)
//	map_iteration_order_test.go    no output built by ranging a map (#2192)
//	geom_kind_switch_test.go       geometry-kind switches live only in kernel/geom (#2188)
//	validate_postcondition_test.go exported ops validate the body they return (#2190)
//	key_resolution_test.go         Find*ByKey refuses an ambiguous key (#2191)
//	lineage_namespace_test.go      a lineage tag names the feature INSTANCE (#2185)
//	kernel_net_delta_test.go       the four quantities every kernel PR reports (#2184)
//
// # The ratchet contract
//
// Most of the above cannot pass today: the rule is right and the code has debt. Each therefore
// carries a checked-in baseline — a per-file or per-package count — and fails in BOTH directions:
//
//   - a RISE is a regression, reported with the file and the budget it broke;
//   - a FALL fails too, telling you to lower the entry. A floor nobody lowers stops being a floor,
//     and this is what keeps the baselines from quietly going stale;
//   - a listed entry that is now clean must be DELETED, so the list cannot outlive the debt.
//
// A listed file is capped, never exempt: adding a violation to it fails exactly as adding one to a
// clean file does. Where some sites are genuinely fine, the escape is an annotation on the line —
// `// tol:<kind>` for a dimensionless constant, `// order:<why>` for a map iteration whose order
// cannot reach the output — not a silent allowlist entry.
//
// # Writing a new rule
//
// Two things are not optional. Prove the guard bites: write a probe into the tree, watch the test
// fail, delete it — a guard nobody has seen fail is a guard that does not work. And fail loudly on
// vacuity: if the scan finds nothing because a symbol was renamed or type information is missing,
// t.Fatal rather than pass, because a guard that silently finds nothing is worse than no guard.
//
// Match the SHAPE, not the declaration. `[]func` is not a dispatch ladder; running entries until
// one succeeds is. #2186 named a composition as a third ladder, and a type-keyed guard would have
// pressured correct code into a worse shape.
package archguard
