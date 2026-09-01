// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"slices"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

// The kernel reports a degradation two ways. An OPERATION-scoped report goes straight into the
// [diag.Recorder] the caller threaded in — that is how ops.BooleanWithDiagnostics reaches a feature
// reply. A RESULT-carried report instead rides on the value produced: [topo.Body.BuildDiagnostics]
// for what the assembler could not do ideally, ops.Mesh.Diagnostics for what the tessellator could
// not. Nothing outside kernel/ops ever read the second kind, so an add-in saw an empty diagnostics
// array while the kernel knew the mesh did not cover its own face — which is how Oblikovati#2038
// shipped a body reading 77% low on volume without a word. This file closes that gap (#2058).

// fileResultBodies records which of the bodies the recompute left standing each feature PRODUCED.
// It is pointer bookkeeping only — no geometry is read — because it runs on every recompute and a
// recompute must not pay for a report nobody asked for. The reading happens in
// [PartFeature.Diagnostics].
//
// A feature whose set of surviving bodies is unchanged keeps its memoized report: a built body's
// geometry is immutable, so its verdict cannot have changed.
func (fs *PartFeatures) fileResultBodies(end int) {
	produced := make(map[*PartFeature][]*topo.Body, end)
	for _, b := range fs.result {
		if owner := fs.producerOf(b, end); owner != nil {
			produced[owner] = append(produced[owner], b)
		}
	}
	for _, pf := range fs.items {
		pf.setResultBodies(produced[pf])
	}
}

// producerOf returns the feature that made b: the earliest one whose output holds this exact body,
// since a feature that leaves a body alone hands back the same pointer. A body no feature claims
// cannot arise from a recomputed result, but if one ever did the last evaluated feature owns it — a
// report with nowhere to go would be exactly the silence this fixes.
func (fs *PartFeatures) producerOf(b *topo.Body, end int) *PartFeature {
	if b == nil || end == 0 {
		return nil
	}
	for _, pf := range fs.items[:end] {
		if holdsBody(pf.cached, b) {
			return pf
		}
	}
	return fs.items[end-1]
}

// holdsBody reports whether bodies contains this exact body (identity, not equality).
func holdsBody(bodies []*topo.Body, b *topo.Body) bool {
	return slices.Contains(bodies, b)
}

// sameBodies reports whether two body lists are the same bodies in the same order.
func sameBodies(a, b []*topo.Body) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// bodyDegradations reads what the bodies carry: the assembler's build report, and what the
// tessellator records while meshing them.
//
// This is the expensive half of #2058 and the reason it is not done during recompute. Meshing can
// cost far more than building — a modeled thread retypes one face in ~400 µs and meshes as a
// height-field grid in ~11 ms (thread_test.go's no-boolean budget) — so a recompute that paid for
// this would make a fast path 27x slower for a report nobody had asked for yet.
func bodyDegradations(bodies []*topo.Body) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, b := range bodies {
		if b == nil {
			continue
		}
		out = append(out, degradationsIn(b.BuildDiagnostics())...)
		out = append(out, degradationsIn(query.BodyMeshDiagnostics(b, displayQuality()))...)
	}
	return out
}

// degradationsIn keeps everything that DEGRADED, dropping [diag.Info].
//
// Info is defined as "a path was taken, nothing degraded" — e.g. the fillet edge catalog stamps every
// body it assembles so a kernel corpus gate can tell "reported nothing" from "never ran". That marker
// is meaningful to the corpus gate and pure noise on a feature reply, which exists to answer "did
// anything about my feature come out worse than I asked for".
func degradationsIn(ds []diag.Diagnostic) []diag.Diagnostic {
	var out []diag.Diagnostic
	for _, d := range ds {
		if d.Severity != diag.Info {
			out = append(out, d)
		}
	}
	return out
}

// displayQuality is the tolerance the result report meshes at: the DISPLAY tolerance, because the
// mesh whose defects the user is about to meet is the one the viewport draws. Reporting against
// ops.PropertyQuality() instead would answer a question nobody asked and cost ~10x the facets.
func displayQuality() ops.Quality { return ops.DefaultQuality() }
