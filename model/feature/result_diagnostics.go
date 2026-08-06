// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// The kernel reports a degradation two ways. An OPERATION-scoped report goes straight into the
// [diag.Recorder] the caller threaded in — that is how ops.BooleanWithDiagnostics reaches a feature
// reply. A RESULT-carried report instead rides on the value produced: [topo.Body.BuildDiagnostics]
// for what the assembler could not do ideally, ops.Mesh.Diagnostics for what the tessellator could
// not. Nothing outside kernel/ops ever read the second kind, so an add-in saw an empty diagnostics
// array while the kernel knew the mesh did not cover its own face — which is how Oblikovati#2038
// shipped a body reading 77% low on volume without a word. This file closes that gap (#2058).

// diagnoseResultBodies drains the result-carried channels of the bodies the recompute left standing
// and files each body's degradations on the feature that produced it. end is the evaluation cutoff.
//
// It reads the RESULT rather than every feature's output because a defect in an intermediate body
// that a later feature cut away is not something the user has, and reporting it would be a false
// alarm. A body still standing from the previous pass keeps its verdict — a built body's geometry is
// immutable — so each body is meshed exactly once, and a recompute that changes nothing is free.
//
// It is not free in general: reporting at all costs one display-tolerance mesh of every body a
// recompute produces (measured +7% on model/feature's suite, +25% on the fillet parity corpus, whose
// bands are expensive to mesh relative to building them). There is no cheaper honest answer — the
// reply is written before any consumer has meshed the body, so somebody has to. CLAUDE.md ranks
// tessellation correctness above features, and a silent 77%-low body (#2038) is what the alternative
// buys.
func (fs *PartFeatures) diagnoseResultBodies(end int) {
	settled := fs.resultDiags
	fs.resultDiags = make(map[*topo.Body][]diag.Diagnostic, len(fs.result))
	for _, b := range fs.result {
		if b != nil {
			fs.resultDiags[b] = bodyDegradations(settled, b)
		}
	}
	fs.fileResultDiagnostics(end)
}

// bodyDegradations returns what b carries, reusing the previous pass's verdict when this exact body
// was already judged (a two-value lookup, so a body known to be CLEAN is remembered as such and not
// re-meshed).
func bodyDegradations(settled map[*topo.Body][]diag.Diagnostic, b *topo.Body) []diag.Diagnostic {
	if known, judged := settled[b]; judged {
		return known
	}
	out := degradationsIn(b.BuildDiagnostics())
	return append(out, degradationsIn(ops.BodyMeshDiagnostics(b, displayQuality()))...)
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

// fileResultDiagnostics attributes each result body's degradations to the feature that produced it,
// REPLACING the previous pass's attribution so repeated recomputes never accumulate duplicates.
func (fs *PartFeatures) fileResultDiagnostics(end int) {
	for _, pf := range fs.items {
		pf.resultDiags = nil
	}
	for _, b := range fs.result {
		producer := fs.producerOf(b, end)
		if producer == nil || len(fs.resultDiags[b]) == 0 {
			continue
		}
		producer.resultDiags = append(producer.resultDiags, fs.resultDiags[b]...)
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
	for _, held := range bodies {
		if held == b {
			return true
		}
	}
	return false
}

// displayQuality is the tolerance the result drain meshes at: the DISPLAY tolerance, because the mesh
// whose defects the user is about to meet is the one the viewport draws. Reporting against
// ops.PropertyQuality() instead would answer a question nobody asked and cost ~10x the facets.
func displayQuality() ops.Quality { return ops.DefaultQuality() }
