// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendEncodeRegularity is the parity gate for OCCT's encoderegularity grid.
func TestOCCTBlendEncodeRegularity(t *testing.T) { runCorpusGrids(t, "encoderegularity") }
