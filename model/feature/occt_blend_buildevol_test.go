// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendBuildevol is the parity gate for OCCT's variable-radius (buildevol) grid.
// Cases skip until the runner drives variable radius (Task 12) and the engine supports it.
func TestOCCTBlendBuildevol(t *testing.T) { runCorpusGrids(t, "buildevol", "tolblend_buildvol") }
