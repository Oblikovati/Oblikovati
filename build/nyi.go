// SPDX-License-Identifier: GPL-2.0-only

package build

import "fmt"

// NotYetImplemented reports an intentionally unfinished code path, keyed by a
// PBI/issue id so the gap is greppable and visible at runtime. Prefer it over
// TODO/FIXME comments for unimplemented branches (architecture/core/01).
//
//	func (e *Extrude) Recompute() error { return build.NotYetImplemented("PBI-082") }
func NotYetImplemented(issueID string) error {
	return fmt.Errorf("not yet implemented: %s", issueID)
}
