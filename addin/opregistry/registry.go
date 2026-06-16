// SPDX-License-Identifier: GPL-2.0-only

// Package opregistry is the self-describing registry of model-mutating operations
// the bridge exposes that have no first-party core registry yet (features and,
// later, sketch geometry). Each operation is an OperationDescriptor: a name, a
// one-line summary, a JSON Schema for its arguments (so an LLM can fill them), and
// an Apply func over the live session. Adding a feature kind to the MCP surface is
// "register one descriptor" — features.list and the schema resources read this
// registry, mirroring the future core registry (ADR-0003).
//
// # Parameter-aware arguments (MANDATORY for every new operation)
//
// Every numeric argument of a feature or tool MUST be parameter-aware and survive
// parameter edits. Do not parse an argument to a frozen float64; resolve it through
// one of the *Closure helpers (lengthClosure, angleClosure, numberClosure, and
// optionalAngleClosure in feature_refs.go) and pass the returned func() value to the
// model builder. A plain literal ("10 mm") becomes a constant;
// any expression ("h", "od/2") is backed by an auto model parameter so it joins the
// dependency graph. set_parameter then marks features dirty and the next recompute
// re-reads the closure at the new value (see addin/router/parameters.go and
// compdef.PartComponentDefinition.Recompute). The model feature/tool field MUST be a
// func() float64 (not a float64) so the value is read live on every recompute.
//
// The only permitted exceptions, which take a literal value, are non-dimensional
// arguments: solver tolerances (stitch tolerance, mid-surface thin-wall threshold)
// and free-form primitive sizes (the subdivision cage is generated once at creation
// and then sculpted, so it has no live size). Integer-count arguments declared as
// JSON ints (pattern occurrences) are a known gap pending a string-expression DTO.
package opregistry

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/app"
)

// ApplyFunc performs an operation against the session, given its JSON arguments,
// and returns a JSON result. It runs on the session goroutine (via the Dispatcher),
// so it may touch the model directly.
type ApplyFunc func(s *app.Session, args json.RawMessage) (json.RawMessage, error)

// OperationDescriptor is one self-describing operation.
type OperationDescriptor struct {
	Name    string
	Summary string
	Schema  json.RawMessage // JSON Schema (draft 2020-12) for the args object
	Apply   ApplyFunc
}

// Registry is an ordered, name-indexed set of operation descriptors.
type Registry struct {
	byName map[string]*OperationDescriptor
	order  []string
}

// New returns an empty registry.
func New() *Registry {
	return &Registry{byName: map[string]*OperationDescriptor{}}
}

// Register adds a descriptor, erroring on a missing name, missing Apply, or duplicate.
func (r *Registry) Register(d *OperationDescriptor) error {
	if d.Name == "" {
		return fmt.Errorf("opregistry: descriptor has no name")
	}
	if d.Apply == nil {
		return fmt.Errorf("opregistry: descriptor %q has no Apply", d.Name)
	}
	if _, dup := r.byName[d.Name]; dup {
		return fmt.Errorf("opregistry: operation %q already registered", d.Name)
	}
	r.byName[d.Name] = d
	r.order = append(r.order, d.Name)
	return nil
}

// ByName looks up a descriptor.
func (r *Registry) ByName(name string) (*OperationDescriptor, bool) {
	d, ok := r.byName[name]
	return d, ok
}

// All returns the descriptors in registration order.
func (r *Registry) All() []*OperationDescriptor {
	out := make([]*OperationDescriptor, len(r.order))
	for i, name := range r.order {
		out[i] = r.byName[name]
	}
	return out
}

// Default returns a registry seeded with the built-in feature operations. It panics
// if a built-in descriptor is malformed, since that is a programming error.
func Default() *Registry {
	r := New()
	// The full built-in feature surface, so add_feature (and the MCP bridge) can drive every
	// modeling operation: additive profile features, the subtractive/dress-up family (by edge/
	// face reference key), direct edits, patterns, and freeform primitives.
	for _, d := range []*OperationDescriptor{
		// Additive profile features.
		extrudeDescriptor(),
		revolveDescriptor(),
		ribDescriptor(),
		embossDescriptor(),
		coilDescriptor(),
		loftDescriptor(),
		// Subtractive / dress-up (edge & face reference keys).
		filletDescriptor(),
		chamferDescriptor(),
		shellDescriptor(),
		draftDescriptor(),
		lipDescriptor(),
		holeDescriptor(),
		bossDescriptor(),
		grillDescriptor(),
		threadDescriptor(),
		// Direct edits and booleans.
		combineDescriptor(),
		thickenDescriptor(),
		trimDescriptor(),
		directEditDescriptor(),
		moveFaceDescriptor(),
		faceOffsetDescriptor(),
		deleteFaceDescriptor(),
		simplifyDescriptor(),
		unwrapDescriptor(),
		modelToleranceDescriptor(),
		splitFaceDescriptor(),
		replaceFaceDescriptor(),
		moveBodyDescriptor(),
		bendPartDescriptor(),
		sheetMetalFaceDescriptor(),
		sheetMetalFlangeDescriptor(),
		sheetMetalHemDescriptor(),
		sheetMetalBendDescriptor(),
		sheetMetalFoldDescriptor(),
		sheetMetalCornerDescriptor(),
		sheetMetalContourFlangeDescriptor(),
		sheetMetalLoftedFlangeDescriptor(),
		sheetMetalContourRollDescriptor(),
		sheetMetalCornerSeamDescriptor(),
		splitSolidDescriptor(),
		coreCavityDescriptor(),
		hullDescriptor(),
		// Additive along a path / patterns.
		sweepDescriptor(),
		rectPatternDescriptor(),
		circPatternDescriptor(),
		mirrorDescriptor(),
		sketchDrivenDescriptor(),
		// Surfacing.
		boundaryPatchDescriptor(),
		ruledSurfaceDescriptor(),
		surfaceOffsetDescriptor(),
		extendDescriptor(),
		midSurfaceDescriptor(),
		stitchDescriptor(),
		sculptDescriptor(),
		// Freeform primitives.
		freeformBoxDescriptor(),
		freeformPlaneDescriptor(),
		freeformQuadBallDescriptor(),
		// Mesh reference geometry.
		meshDescriptor(),
	} {
		if err := r.Register(d); err != nil {
			panic(fmt.Sprintf("opregistry: seeding default registry: %v", err))
		}
	}
	return r
}
