// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendComplex is the parity gate for OCCT's complex grid (restored real-world solids).
func TestOCCTBlendComplex(t *testing.T) { runCorpusGrids(t, "complex") }
