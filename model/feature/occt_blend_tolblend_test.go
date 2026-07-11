// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendTolblend is the parity gate for OCCT's tolerance-blend (tolblend_simple) grid.
func TestOCCTBlendTolblend(t *testing.T) { runCorpusGrids(t, "tolblend_simple") }
