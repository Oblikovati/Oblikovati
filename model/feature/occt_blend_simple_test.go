// SPDX-License-Identifier: GPL-2.0-only

package feature_test

import "testing"

// TestOCCTBlendSimple is the parity gate for OCCT's tests/blend/simple grid (inline-primitive
// constant-radius fillets). Reds are the ADR-0050 greening backlog, never loosened.
func TestOCCTBlendSimple(t *testing.T) { runCorpusGrids(t, "simple") }
