// SPDX-License-Identifier: GPL-2.0-only

// Command batch-translate translates every .ipt under an input directory (recursively) into
// .opd packages mirrored under an output directory, then classifies each by the PARAMETRIC
// state it reached — solid / surface / partial (features but no body) / sketch-only / empty —
// and writes a TSV report. It is a developer aid for auditing real-part coverage; it needs no
// running Inventor. With the mesh fallback OFF (the default), a part that can't fully rebuild is
// saved in its partial parametric state, so the report shows how far the feature decode got.
//
// Usage:
//
//	batch-translate -in <dir> -out <dir> [-report <file.tsv>]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/analysis"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/exchange/translators/inventor/ipt"
	"oblikovati.org/model/exchange/translators/inventor/translate"
	"oblikovati.org/persistence"
)

func main() {
	in := flag.String("in", "", "input directory to scan recursively for .ipt files")
	out := flag.String("out", "", "output directory for the mirrored .opd packages")
	report := flag.String("report", "", "report TSV path (default: <out>/_report.tsv)")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "batch-translate: -in and -out are required")
		os.Exit(2)
	}
	rep := *report
	if rep == "" {
		rep = filepath.Join(*out, "_report.tsv")
	}
	rows, tally := run(*in, *out)
	if err := writeReport(rep, rows); err != nil {
		fmt.Fprintln(os.Stderr, "batch-translate:", err)
		os.Exit(1)
	}
	printTally(rep, len(rows), tally)
	printConstraintStats(rows)
}

// row is one part's audit line.
type row struct {
	outcome     string
	volumeMm3   float64
	sketches    int
	features    int
	dof         int // remaining degrees of freedom summed over the part's sketches
	eqs         int // constraint+dimension equations summed over the part's sketches
	constrained int // sketches carrying at least one constraint equation
	featTags    string
	rel         string
}

// run walks inDir for .ipt files, translates each into outDir (mirrored), and classifies it.
func run(inDir, outDir string) ([]row, map[string]int) {
	var rows []row
	tally := map[string]int{}
	_ = filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.EqualFold(filepath.Ext(path), ".ipt") {
			return nil
		}
		r := translateOne(inDir, outDir, path)
		rows = append(rows, r)
		tally[r.outcome]++
		return nil
	})
	sort.Slice(rows, func(i, j int) bool { return rows[i].rel < rows[j].rel })
	return rows, tally
}

// translateOne translates a single .ipt and classifies the resulting .opd.
func translateOne(inDir, outDir, path string) row {
	rel, _ := filepath.Rel(inDir, path)
	outPath := filepath.Join(outDir, strings.TrimSuffix(rel, filepath.Ext(rel))+".opd")
	_ = os.MkdirAll(filepath.Dir(outPath), 0o755)
	r := row{outcome: "ERROR", rel: rel}
	data, err := os.ReadFile(path)
	if err != nil {
		return r
	}
	r.featTags = featureTags(data)
	if _, err := translate.FromInventor(data, outPath); err != nil {
		return r
	}
	classify(&r, outPath)
	return r
}

// classify reopens the translated .opd and records its parametric state.
func classify(r *row, outPath string) {
	ws := doc.NewWorkspace(persistence.NewPackageStore(), contentset.Default())
	reopened, err := ws.Open(outPath, true)
	if err != nil {
		r.outcome = "REOPEN-FAIL"
		return
	}
	def, ok := reopened.Content().(*compdef.PartComponentDefinition)
	if !ok {
		r.outcome = "NOT-PART"
		return
	}
	r.sketches = def.Sketches().Count()
	r.features = def.Features().Count()
	for k := 0; k < r.sketches; k++ {
		a := def.Sketches().Item(k).AnalyzeConstraints()
		r.dof += a.DOF
		r.eqs += a.Equations
		if a.Equations > 0 {
			r.constrained++
		}
	}
	bodies := def.SurfaceBodies().All()
	switch {
	case len(bodies) > 0 && bodies[0].IsSolid():
		r.outcome = "SOLID"
		r.volumeMm3 = analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	case len(bodies) > 0:
		r.outcome = "SURFACE"
		r.volumeMm3 = analysis.MassPropertiesOf(bodies, 1, types.MassPropertiesLow).VolumeMm3
	case r.features > 0:
		r.outcome = "PARTIAL" // features built but computed to no body
	case r.sketches > 0:
		r.outcome = "SKETCH"
	default:
		r.outcome = "EMPTY"
	}
}

// featureTags decodes a short marker of which feature types the .ipt carries — a coverage hint
// independent of whether they built (e.g. "rev" appears even when the revolve declined).
func featureTags(data []byte) string {
	d, err := ipt.Open(data)
	if err != nil || d.IsAssembly() {
		return ""
	}
	seg, ok := d.Segment("PmDCSegment")
	if !ok {
		return ""
	}
	var tags []string
	if len(ipt.DecodeExtrudes(d)) > 0 {
		tags = append(tags, "ext")
	}
	if ipt.HasRevolve(seg) {
		tags = append(tags, "rev")
	}
	if _, ok := ipt.DecodeHole(seg); ok {
		tags = append(tags, "hole")
	}
	if _, ok := ipt.DecodeSweep(seg); ok {
		tags = append(tags, "sweep")
	}
	return strings.Join(tags, ",")
}

// writeReport writes the audit rows as a TSV.
func writeReport(path string, rows []row) error {
	var b strings.Builder
	b.WriteString("outcome\tvolume_mm3\tsketches\tfeatures\tdof\teqs\tfeat_tags\tpart\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\t%.0f\t%d\t%d\t%d\t%d\t%s\t%s\n", r.outcome, r.volumeMm3, r.sketches, r.features, r.dof, r.eqs, r.featTags, r.rel)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// printConstraintStats summarizes the DOF-parity work across the library: how much of the emitted
// sketch geometry now carries constraints/dimensions rather than being a free point-soup.
func printConstraintStats(rows []row) {
	var sketches, constrained, eqs, dof, partsWithSketches, partsWithConstraints int
	for _, r := range rows {
		sketches += r.sketches
		constrained += r.constrained
		eqs += r.eqs
		dof += r.dof
		if r.sketches > 0 {
			partsWithSketches++
		}
		if r.eqs > 0 {
			partsWithConstraints++
		}
	}
	fmt.Println("constraint / DOF stats:")
	fmt.Printf("  %d sketches emitted across %d parts\n", sketches, partsWithSketches)
	fmt.Printf("  %d sketches carry constraints (%.0f%% of emitted)\n", constrained, pct(constrained, sketches))
	fmt.Printf("  %d parts have ≥1 constrained sketch (%.0f%% of parts with sketches)\n", partsWithConstraints, pct(partsWithConstraints, partsWithSketches))
	fmt.Printf("  %d constraint/dimension equations applied; %d DOF remaining\n", eqs, dof)
}

func pct(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(n) / float64(total)
}

// printTally prints a one-line-per-outcome summary to stdout.
func printTally(reportPath string, total int, tally map[string]int) {
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return tally[keys[i]] > tally[keys[j]] })
	fmt.Printf("translated %d parts → %s\n", total, reportPath)
	for _, k := range keys {
		fmt.Printf("  %-12s %3d  (%.0f%%)\n", k, tally[k], 100*float64(tally[k])/float64(total))
	}
}
