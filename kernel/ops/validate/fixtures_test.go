// SPDX-License-Identifier: GPL-2.0-only

package validate

// Fixture builders restated from kernel/ops' test package. Go cannot share a _test.go
// helper across packages, and a shared fixture package would import kernel/ops, which
// kernel/ops' own tests could then not use (import cycle). This is the test scaffolding
// sonar.cpd.exclusions already accounts for. They build bodies through topo alone, so
// they carry no dependency on the operation layer.

import (
	stdmath "math"
)

// near reports whether two parameter samples coincide within tol (inclusive, so exact
// duplicates collapse even on a degenerate zero-span axis).
func near(a, b, tol float64) bool { return stdmath.Abs(a-b) <= tol }
