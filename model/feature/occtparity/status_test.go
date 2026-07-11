// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

func TestClassifyMirrorsOCCT(t *testing.T) {
	todo := Record{TODO: "TODO OCC22817 All:TEST INCOMPLETE"}
	if classify(todo, true, false, false) != SkipTODO {
		t.Fatal("TODO case must skip")
	}
	ok := Record{}
	if classify(ok, true, true, true) != Pass {
		t.Fatal("clean run must pass")
	}
	if classify(ok, true, true, false) != FailFaulty {
		t.Fatal("invalid solid must fail Faulty")
	}
	if classify(ok, false, false, false) != SkipImportDivergence {
		t.Fatal("import failure separates from fillet")
	}
}
