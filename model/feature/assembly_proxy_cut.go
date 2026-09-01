// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/transform"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/model/occurrence"
)

// occurrenceBodies is a component definition that owns evaluated bodies — a part
// definition. An [AssemblyProxyCutFeature] reads its source occurrence's bodies through
// this without importing model/compdef.
type occurrenceBodies interface {
	SurfaceBodies() *topo.SurfaceBodies
}

// AssemblyProxyCutFeature is an assembly-machining feature whose tool is supplied as an
// occurrence-context proxy of placed geometry, not as a definition-space native fixed
// at construction (M11-F08 proxy inputs, #734). Its source is another occurrence in the
// assembly; at every recompute it resolves that occurrence's current bodies into
// assembly space through the occurrence context — its placement transform, the same
// definition→assembly view [proxy.NewContext] exposes — and booleans them against each
// participant body, so the cut is associative: move or edit the source component and the
// machining follows.
//
// This realizes the issue's "proxy-to-profile resolution path": the input is the source
// occurrence's geometry viewed through its assembly context, re-resolved each rebuild,
// versus [AssemblyCutFeature]'s tool box frozen in assembly space.
//
// Example: subtract the bracket placed at occurrence src from every participant —
//
//	f := feature.NewAssemblyProxyCutFeature(src, ops.Cut)
type AssemblyProxyCutFeature struct {
	source *occurrence.Occurrence
	op     ops.PartFeatureOperation
}

// NewAssemblyProxyCutFeature returns a proxy-input feature that applies op with the
// geometry of the source occurrence (resolved into assembly space each recompute).
func NewAssemblyProxyCutFeature(source *occurrence.Occurrence, op ops.PartFeatureOperation) *AssemblyProxyCutFeature {
	return &AssemblyProxyCutFeature{source: source, op: op}
}

// Kind implements [Feature].
func (f *AssemblyProxyCutFeature) Kind() string { return kindAssemblyProxyCut }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblyProxyCutFeature) Operation() ops.PartFeatureOperation { return f.op }

// Source returns the occurrence whose proxied geometry is the tool.
func (f *AssemblyProxyCutFeature) Source() *occurrence.Occurrence { return f.source }

// Recompute resolves the source occurrence's bodies into assembly space through its
// proxy context and booleans them against every running body, replacing each with the
// result (an emptied body is dropped). A source with no bodies is a lost input the
// engine turns into feature health, not a panic.
func (f *AssemblyProxyCutFeature) Recompute(in Input) (Output, error) {
	tools, err := f.resolveTools()
	if err != nil {
		return Output{}, err
	}
	out := append([]*topo.Body(nil), in.Bodies...)
	for _, tool := range tools {
		out, err = applyToolToAll(f.op, out, tool, in.Diag)
		if err != nil {
			return Output{}, err
		}
	}
	return Output{Bodies: out}, nil
}

// resolveTools views the source occurrence's bodies in assembly space through its
// occurrence context (its placement transform) — the proxy-input resolution this
// feature exists to demonstrate, re-read each call so the tool tracks the source.
func (f *AssemblyProxyCutFeature) resolveTools() ([]*topo.Body, error) {
	def, ok := f.source.Definition().(occurrenceBodies)
	if !ok {
		return nil, fmt.Errorf("assemblyProxyCut: source occurrence %q has no bodies to proxy", f.source.Name())
	}
	world := f.source.Transform()
	var tools []*topo.Body
	for i, b := range def.SurfaceBodies().All() {
		t, err := transform.TransformBody(b, world, proxyToolLineage(i))
		if err != nil {
			return nil, fmt.Errorf("assemblyProxyCut: proxy source body %d into context: %w", i, err)
		}
		tools = append(tools, t)
	}
	return tools, nil
}

// applyToolToAll booleans tool against each body, dropping any emptied result. rec collects the
// kernel's boolean-fallback diagnostics (#1601; nil discards).
func applyToolToAll(op ops.PartFeatureOperation, bodies []*topo.Body, tool *topo.Body, rec *diag.Recorder) ([]*topo.Body, error) {
	out := make([]*topo.Body, 0, len(bodies))
	for i, target := range bodies {
		res, err := ops.BooleanWithDiagnostics(op, target, tool, rec)
		if err != nil {
			return nil, fmt.Errorf("assemblyProxyCut: boolean on body %d: %w", i, err)
		}
		if res != nil && len(res.Faces()) > 0 {
			out = append(out, res)
		}
	}
	return out, nil
}

// proxyToolLineage gives each proxied tool body a distinct lineage prefix so repeated
// resolutions keep independent reference keys.
func proxyToolLineage(index int) func(topo.Lineage) topo.Lineage {
	prefix := topo.Tok(kindAssemblyProxyCut, "tool", index)
	return func(l topo.Lineage) topo.Lineage {
		return topo.NewLineage(append([]topo.LineageToken{prefix}, l.Tokens()...)...)
	}
}
