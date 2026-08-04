// SPDX-License-Identifier: GPL-2.0-only

// Command gen walks every OCCT tests/blend/<grid>/<case> script, runs the OCCT oracle
// (test-utilities/occt-blend/oracle/occt_blend_oracle) over each one, and emits a committed
// Go corpus (model/feature/occtparity/corpus.json + corpus.go) plus the STEP fixtures the
// runner imports.
//
// Usage:
//
//	go run ./test-utilities/occt-blend/gen \
//	  -occt ../OCCT/tests/blend \
//	  -oracle test-utilities/occt-blend/oracle/occt_blend_oracle \
//	  -out model/feature/occtparity
//
// A case is never dropped: if the oracle fails to run, writes no JSON, or writes JSON that
// does not parse, the case still gets a Record with TODO set to "unparsed: <reason>" so the
// corpus size invariant (model/feature/occtparity/corpus_test.go) always matches the true
// OCCT case count.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// caseNamePattern matches an OCCT tests/blend case file: one capital letter + a digit run
// (e.g. "A1", "K12"). It excludes the grid's "begin"/"end" driver scripts and any listing
// files, which never match this shape.
var caseNamePattern = regexp.MustCompile(`^[A-Z][0-9]+$`)

// oracleTimeout bounds a single DRAWEXE invocation; real cases take ~0.5-1s, but a
// pathological multi-blend/loop case must not hang the whole 475-case run.
const oracleTimeout = 120 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}

// run generates the corpus, returning an error rather than exiting so the deferred oracle temp-dir
// cleanup actually runs — os.Exit skips defers, which leaked a temp dir on every error path.
func run() error {
	occtDir := flag.String("occt", "../OCCT/tests/blend", "OCCT tests/blend root")
	oraclePath := flag.String("oracle", "test-utilities/occt-blend/oracle/occt_blend_oracle", "oracle wrapper path")
	outDir := flag.String("out", "model/feature/occtparity", "output package dir")
	flag.Parse()

	cases, err := collectCases(*occtDir)
	if err != nil {
		return fmt.Errorf("collect cases: %w", err)
	}
	tmpDir, err := os.MkdirTemp("", "occt-blend-oracle-")
	if err != nil {
		return fmt.Errorf("mkdir temp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	records := make([]record, 0, len(cases))
	for _, c := range cases {
		records = append(records, runOneCase(*oraclePath, *outDir, tmpDir, c))
	}
	if err := writeCorpus(*outDir, records); err != nil {
		return fmt.Errorf("write corpus: %w", err)
	}
	logSummary(records)
	return nil
}

// caseRef identifies one OCCT tests/blend case by its grid and case name plus the path to
// its .tcl-less script file (OCCT case files have no extension).
type caseRef struct {
	grid, name, path string
}

// collectCases walks occtDir's grid subdirectories and returns every case file, sorted grid
// ascending then case ascending — the plan's required, deterministic corpus order.
func collectCases(occtDir string) ([]caseRef, error) {
	grids, err := os.ReadDir(occtDir)
	if err != nil {
		return nil, fmt.Errorf("collectCases: read %s: %w", occtDir, err)
	}
	var gridNames []string
	for _, g := range grids {
		if g.IsDir() {
			gridNames = append(gridNames, g.Name())
		}
	}
	sort.Strings(gridNames)

	var cases []caseRef
	for _, grid := range gridNames {
		names, err := casesInGrid(filepath.Join(occtDir, grid))
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			cases = append(cases, caseRef{grid: grid, name: name, path: filepath.Join(occtDir, grid, name)})
		}
	}
	return cases, nil
}

// casesInGrid lists one grid directory's case files, sorted ascending.
func casesInGrid(gridDir string) ([]string, error) {
	entries, err := os.ReadDir(gridDir)
	if err != nil {
		return nil, fmt.Errorf("casesInGrid: read %s: %w", gridDir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && caseNamePattern.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// runOneCase runs the oracle over one case and returns its Record, never dropping the case:
// an oracle/parse failure is captured as an "unparsed: <reason>" TODO record instead.
func runOneCase(oraclePath, outDir, tmpDir string, c caseRef) record {
	out, runErr := invokeOracle(oraclePath, c.path, tmpDir)
	jsonPath := filepath.Join(tmpDir, c.name+".json")
	stepPath := filepath.Join(tmpDir, c.name+".step")
	defer os.Remove(jsonPath)
	defer os.Remove(stepPath)

	raw, readErr := os.ReadFile(jsonPath)
	if readErr != nil {
		return unparsedRecord(c, fmt.Sprintf("oracle produced no JSON (run err=%v): %s", runErr, tail(out, 400)))
	}
	var r record
	if err := json.Unmarshal(raw, &r); err != nil {
		return unparsedRecord(c, fmt.Sprintf("JSON did not parse: %v", err))
	}
	r.Grid, r.Case = c.grid, c.name // authoritative: the directory walk, not the oracle's own echo
	r.InputStep = resolveFixture(stepPath, outDir, c, r.InputStep)
	return withMissingFixtureTODO(r)
}

// withMissingFixtureTODO flags a case whose `blend` (or mkevol/bfuseblend) call never ran at
// all — no TODO marker, yet zero picks — with a synthetic TODO. This is the ~27-fixture gap
// documented in test-utilities/occt-blend/SOURCES.md: the case's `restore [locate_data_file
// …]` target isn't vendored, so `catch {source $casePath}` (oracle.tcl) aborts the script
// before it ever reaches `blend`, leaving ORV(picks) empty while scanCheckprops still read a
// real reference area straight from the case's static text. Without this, such a case would
// violate corpus_test.go's non-TODO invariant (ExpectedArea>0 but no picks to drive); with it,
// RunCase skips it with a diagnosable reason instead of a generic import-divergence.
func withMissingFixtureTODO(r record) record {
	if r.TODO == "" && len(r.Picks) == 0 {
		r.TODO = "occtparity: no picks resolved — restore likely failed on an unvendored " +
			"test-data fixture (see test-utilities/occt-blend/SOURCES.md)"
	}
	return r
}

// resolveFixture commits the oracle's temp STEP export to the committed fixtures tree and
// returns the record's fixture-relative InputStep, or "" when there is nothing to commit.
//
// oracle.tcl's own stepwrite call is wrapped in a Tcl `catch` and silently no-ops whenever the
// blend's input shape never resolved to a real shape — every TODO case (OCCT's own marker,
// occtparity's multi-blend, or an unresolved-edge pick) and every case whose `restore` target
// is one of the ~27 unvendored test-data fixtures (test-utilities/occt-blend/SOURCES.md) hits
// this. The JSON's inputStep field still optimistically names "<case>.step" in that situation
// (it's a template default, not a promise). So a missing physical file here is expected, not a
// generator defect: we quietly fall back to "" rather than downgrading an otherwise-complete
// record (verb/expectedArea/todo/picks all still valid) to an opaque "unparsed" placeholder.
func resolveFixture(stepPath, outDir string, c caseRef, claimedInputStep string) string {
	if claimedInputStep == "" {
		return ""
	}
	if err := commitFixture(stepPath, outDir, c); err != nil {
		return ""
	}
	return c.grid + "/" + c.name + ".step"
}

// invokeOracle runs the bash oracle wrapper with a hang-guard timeout and returns its
// combined stdout+stderr (for diagnostics) and any run error.
func invokeOracle(oraclePath, casePath, tmpDir string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), oracleTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, oraclePath, casePath, tmpDir)
	out, err := cmd.CombinedOutput()
	return out, err
}

// commitFixture copies the oracle's temp STEP export to the committed fixtures tree.
func commitFixture(stepPath, outDir string, c caseRef) error {
	dstDir := filepath.Join(outDir, "fixtures", c.grid)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return fmt.Errorf("commitFixture: mkdir %s: %w", dstDir, err)
	}
	dst := filepath.Join(dstDir, c.name+".step")
	if err := copyFile(stepPath, dst); err != nil {
		return fmt.Errorf("commitFixture: copy %s -> %s: %w", stepPath, dst, err)
	}
	return nil
}

// copyFile copies src to dst, both regular files.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// unparsedRecord builds the loud, never-dropped placeholder for a case the oracle could not
// produce a usable record for. It carries no picks/area so corpus_test.go's non-TODO
// invariant does not apply to it, and the TODO string documents why.
func unparsedRecord(c caseRef, reason string) record {
	return record{Grid: c.grid, Case: c.name, TODO: "unparsed: " + reason}
}

// tail bounds diagnostic output to its last n bytes so a runaway oracle log does not blow up
// the unparsed record's TODO string.
func tail(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// logSummary reports the generated corpus's shape — the counts a regeneration is checked against
// (a jump in unparsed or empty-inputStep means the OCCT script parser has drifted).
func logSummary(records []record) {
	var unparsed, multiBlend, todoOther, emptyInputStep int
	for _, r := range records {
		switch {
		case strings.HasPrefix(r.TODO, "unparsed:"):
			unparsed++
		case strings.Contains(r.TODO, "multi-blend"):
			multiBlend++
		case r.TODO != "":
			todoOther++
		}
		if r.InputStep == "" {
			emptyInputStep++
		}
	}
	slog.Info("corpus generated", "total", len(records), "todoOCCT", todoOther,
		"multiBlend", multiBlend, "unparsed", unparsed, "emptyInputStep", emptyInputStep)
}

// locator/pick/record mirror model/feature/occtparity/record.go's JSON shape exactly (same
// field names and json tags) without importing that package into this generator — Task 9's
// generator emits that package's source, so it must not depend on it.
type locator struct {
	Midpoint  [3]float64 `json:"midpoint"`
	Direction [3]float64 `json:"direction"`
	Centroid  [3]float64 `json:"centroid"`
	Length    float64    `json:"length"`
}
type pick struct {
	Radius  float64      `json:"radius"`
	Locator locator      `json:"locator"`
	Law     [][2]float64 `json:"law"`
}
type record struct {
	Grid         string  `json:"grid"`
	Case         string  `json:"case"`
	Verb         string  `json:"verb"`
	ExpectedArea float64 `json:"expectedArea"`
	Deps         float64 `json:"deps"`
	TODO         string  `json:"todo"`
	InputStep    string  `json:"inputStep"`
	Picks        []pick  `json:"picks"`
}

// writeCorpus emits corpus.json (the 475 records) and a small static corpus.go that embeds
// and unmarshals it. The embed form is used instead of a literal []Record{...} because the
// full corpus (buildevol laws included) would produce several thousand lines of Go literal —
// well past this project's 500-line-file convention and unwieldy to review as source.
func writeCorpus(outDir string, records []record) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("writeCorpus: mkdir %s: %w", outDir, err)
	}
	jsonBytes, err := json.MarshalIndent(records, "", "\t")
	if err != nil {
		return fmt.Errorf("writeCorpus: marshal: %w", err)
	}
	jsonPath := filepath.Join(outDir, "corpus.json")
	if err := os.WriteFile(jsonPath, append(jsonBytes, '\n'), 0o644); err != nil {
		return fmt.Errorf("writeCorpus: write %s: %w", jsonPath, err)
	}
	return writeCorpusGo(outDir)
}

// corpusGoTemplate is the generated corpus.go accessor source, hoisted out of writeCorpusGo so
// that function stays readable (the template is data, not logic).
const corpusGoTemplate = `// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	_ "embed"
	"encoding/json"
	"fmt"
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

// CorpusFixtureDir is the directory (relative to this package, i.e. resolved from a test's
// working directory) holding the committed STEP fixtures Corpus() records reference via
// InputStep.
func CorpusFixtureDir() string { return "fixtures" }
`

// writeCorpusGo emits the static, gofmt'ed corpus.go accessor for the embedded corpus.json.
// writeCorpusGo emits the package's corpus.go accessor (gofmt-formatted) beside corpus.json.
func writeCorpusGo(outDir string) error {
	src := corpusGoTemplate
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("writeCorpusGo: gofmt generated source: %w", err)
	}
	path := filepath.Join(outDir, "corpus.go")
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return fmt.Errorf("writeCorpusGo: write %s: %w", path, err)
	}
	return nil
}
