// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"fmt"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// AssemblySweepFeature is the assembly-machining sweep (M11-F08 kind set, #735): a sketch
// profile authored on an assembly work plane, swept along an explicit polyline path in the
// assembly's space, and booleaned against each participant body. It reuses the part sweep
// geometry (sweepSectionsCfg/sweptSolid) so a profiled channel (Cut) or rib (Join) machines
// every participant in place. The path is supplied directly — the assembly analogue of the
// revolve's explicit axis — since assembly-context 3D sketch paths are a later subsystem.
//
// Example: cut a channel of profile sk along a vertical path through every participant —
//
//	f := feature.NewAssemblySweepFeature(sk, 0, ops.Cut, []math.Point3{p0, p1})
type AssemblySweepFeature struct {
	sketch       *sketch.Sketch
	profileIndex int
	op           ops.PartFeatureOperation
	path         []math.Point3
}

// NewAssemblySweepFeature returns a sweep of the sketch's profileIndex-th closed region
// along path, applying op. op is typically [ops.Cut] (a swept channel) or [ops.Join] (a
// swept rib). path must have at least two points.
func NewAssemblySweepFeature(skt *sketch.Sketch, profileIndex int, op ops.PartFeatureOperation, path []math.Point3) *AssemblySweepFeature {
	return &AssemblySweepFeature{sketch: skt, profileIndex: profileIndex, op: op, path: path}
}

// Kind implements [Feature].
func (f *AssemblySweepFeature) Kind() string { return "assemblySweep" }

// Operation reports the boolean the feature applies, satisfying [OperationalFeature].
func (f *AssemblySweepFeature) Operation() ops.PartFeatureOperation { return f.op }

// Recompute sweeps the profile along the path into an assembly-space tool and booleans it
// against every running body. A missing/open profile or a degenerate path is a lost input
// the engine turns into feature health, not a panic.
func (f *AssemblySweepFeature) Recompute(in Input) (Output, error) {
	tool, err := f.buildTool()
	if err != nil {
		return Output{}, err
	}
	out, err := applyToolToAll(f.op, in.Bodies, tool)
	if err != nil {
		return Output{}, err
	}
	return Output{Bodies: out}, nil
}

// buildTool sweeps the resolved profile along the path into the assembly-space tool, using
// the default normal-to-path orientation with no twist.
func (f *AssemblySweepFeature) buildTool() (*topo.Body, error) {
	if len(f.path) < 2 {
		return nil, fmt.Errorf("assemblySweep: path needs at least two points, got %d", len(f.path))
	}
	prof, err := resolveSingleProfile(f.sketch, f.profileIndex, "assemblySweep")
	if err != nil {
		return nil, err
	}
	cfg := sweepConfig{orientation: types.NormalToPath, twistAt: func(float64) float64 { return 0 }}
	sections, err := sweepSectionsCfg(prof, f.sketch.Plane(), f.path, cfg)
	if err != nil {
		return nil, fmt.Errorf("assemblySweep: %w", err)
	}
	return sweptSolid(sections, false, "asmSweep")
}
