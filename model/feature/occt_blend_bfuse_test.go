// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendBfuse is the parity gate for OCCT's bfuseblend grid (fuse two solids, then
// blend every edge of their boolean section).
func TestOCCTBlendBfuse(t *testing.T) { runCorpusGrids(t, "bfuseblend") }
