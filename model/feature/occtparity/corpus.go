// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
)

// corpusJSON is the generated OCCT tests/blend corpus (all grids), produced by
// test-utilities/occt-blend/gen from the OCCT oracle. Regenerate via:
//
//	go run ./test-utilities/occt-blend/gen -occt ../OCCT/tests/blend \
//	  -oracle test-utilities/occt-blend/oracle/occt_blend_oracle -out model/feature/occtparity
//
//go:embed corpus.json
var corpusJSON []byte

// Corpus returns every OCCT tests/blend case as a Record. It is committed generated data
// (corpus.json, embedded) rather than a Go literal: the full corpus (buildevol variable-
// radius laws included) runs to several thousand lines as a literal, which corpus.json plus
// this accessor avoids while keeping the same []Record surface for the runner.
func Corpus() []Record {
	var records []Record
	if err := json.Unmarshal(corpusJSON, &records); err != nil {
		panic(fmt.Sprintf("occtparity: embedded corpus.json does not parse as []Record: %v", err))
	}
	return records
}

// CorpusFixtureDir is the absolute directory holding the committed STEP fixtures that Corpus()
// records reference via InputStep. It is resolved from this source file's location (not the
// test's working directory) so callers in any package — occtparity's own tests and the feature
// grid tests, which run with different CWDs — find the fixtures identically.
func CorpusFixtureDir() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "fixtures"
	}
	return filepath.Join(filepath.Dir(thisFile), "fixtures")
}
