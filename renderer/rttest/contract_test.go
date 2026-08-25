// SPDX-License-Identifier: GPL-2.0-only

package rttest

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestFakeIntersectorSatisfiesContract proves the shared suite itself works, using
// renderer.FakeIntersector as the reference implementation every real backend
// (PBI-333/334) is checked against.
func TestFakeIntersectorSatisfiesContract(t *testing.T) {
	RunIntersectorContractTests(t, func(tris []renderer.Triangle) renderer.Intersector {
		return renderer.NewFakeIntersector(tris)
	})
}
