// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

// This package IS the OCCT blend-parity corpus: 154 tests, ~1390s of the suite's
// ~3060s, the single largest block of test time in the repository. Every test here
// rebuilds shipped corpus records through the real fillet feature, so none of it
// belongs in the inner loop — it is tier 2 in full
// (architecture/testing/03-test-tiers-and-selection.md).
//
// TestMain gates the whole package rather than 154 individual testing.Short() calls:
// one gate cannot drift out of step with the package, and a test added here inherits
// the tier instead of having to remember it.
func TestMain(m *testing.M) {
	flag.Parse() // testing.Short() is only meaningful once the flags are parsed.
	if testing.Short() {
		fmt.Println("occtparity: corpus tier skipped in -short; run `make test-corpus`")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
