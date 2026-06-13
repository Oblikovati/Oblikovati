// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/occurrence"
)

// The assembly replication surface (M11-F04, #729): replicate placed components by
// session id — pattern (circular/rectangular), mirror across a plane, independent copy,
// and substitute a set with one simplified representation. Each op adds occurrences to
// the active assembly; the additive ops reply with what they created, substitute with
// the substitute occurrence. Occurrence resolution, info rendering, and the simplified-
// definition lookup are shared with the occurrence surface (assembly_occurrences.go).

// registerAssemblyReplicationHandlers wires the assembly.* replication methods.
func (r *Router) registerAssemblyReplicationHandlers() {
	r.handlers[wire.MethodAssemblyPatternCreate] = assemblyPatternCreate
	r.handlers[wire.MethodAssemblyMirror] = assemblyMirror
	r.handlers[wire.MethodAssemblyCopy] = assemblyCopy
	r.handlers[wire.MethodAssemblySubstitute] = assemblySubstitute
}

// assemblyPatternCreate replicates the seed occurrence across an arrangement, adding one
// occurrence per generated element beyond the seed.
func assemblyPatternCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.CreatePatternArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	seed, err := occurrenceByID(asm, in.Seed, wire.MethodAssemblyPatternCreate)
	if err != nil {
		return nil, err
	}
	arr, err := arrangementFromArgs(in)
	if err != nil {
		return nil, err
	}
	pat := occurrence.NewOccurrencePattern(seed.Definition(), seed.Transform(), arr)
	return newOccurrencesReply(patternOccurrences(asm, seed, pat))
}

// assemblyMirror adds a mirror of each source occurrence across the plane (origin, normal).
func assemblyMirror(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.MirrorComponentsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sources, err := occurrencesByID(asm, in.Sources, wire.MethodAssemblyMirror)
	if err != nil {
		return nil, err
	}
	normal, err := math.NewUnitVector3(in.Normal[0], in.Normal[1], in.Normal[2])
	if err != nil {
		return nil, fmt.Errorf("%s: normal %v is not a direction: %w", wire.MethodAssemblyMirror, in.Normal, err)
	}
	origin := math.P3(in.Origin[0], in.Origin[1], in.Origin[2])
	return newOccurrencesReply(occurrence.MirrorComponents(asm.Occurrences(), sources, origin, normal))
}

// assemblyCopy adds an independent copy of each source occurrence.
func assemblyCopy(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.CopyComponentsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sources, err := occurrencesByID(asm, in.Sources, wire.MethodAssemblyCopy)
	if err != nil {
		return nil, err
	}
	return newOccurrencesReply(occurrence.CopyComponents(asm.Occurrences(), sources))
}

// assemblySubstitute suppresses the source occurrences and adds one occurrence instancing
// the simplified component held by an open document.
func assemblySubstitute(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	var in wire.SubstituteComponentsArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	sources, err := occurrencesByID(asm, in.Sources, wire.MethodAssemblySubstitute)
	if err != nil {
		return nil, err
	}
	def, err := placeableDefinition(s, in.Document, wire.MethodAssemblySubstitute)
	if err != nil {
		return nil, err
	}
	sub := occurrence.Substitute(asm.Occurrences(), sources, in.Name, def, matrixFromWire(in.Transform))
	return occurrenceReply(sub)
}

// patternOccurrences places a real occurrence for each pattern element beyond the seed
// (element 0 is the seed, already placed), reusing the pattern's arrangement placements.
func patternOccurrences(asm *compdef.AssemblyComponentDefinition, seed *occurrence.Occurrence, pat *occurrence.OccurrencePattern) []*occurrence.Occurrence {
	created := make([]*occurrence.Occurrence, 0, pat.Count())
	for i := 1; i < pat.Count(); i++ {
		name := fmt.Sprintf("%s-p%d", seed.Name(), i+1)
		created = append(created, asm.Place(name, seed.Definition(), pat.Element(i).Transform()))
	}
	return created
}

// arrangementFromArgs builds the pattern arrangement from the request's kind + parameters.
func arrangementFromArgs(in wire.CreatePatternArgs) (occurrence.Arrangement, error) {
	switch in.Kind {
	case "circular":
		return circularArrangement(in)
	case "rectangular":
		return rectangularArrangement(in)
	}
	return nil, fmt.Errorf("%s: unknown pattern kind %q (want circular/rectangular)", wire.MethodAssemblyPatternCreate, in.Kind)
}

// circularArrangement builds a circular arrangement (Angle radians between Count elements
// about the axis through Origin).
func circularArrangement(in wire.CreatePatternArgs) (occurrence.Arrangement, error) {
	axis, err := math.NewUnitVector3(in.Axis[0], in.Axis[1], in.Axis[2])
	if err != nil {
		return nil, fmt.Errorf("%s: axis %v is not a direction: %w", wire.MethodAssemblyPatternCreate, in.Axis, err)
	}
	if in.Count < 1 {
		return nil, fmt.Errorf("%s: count %d must be >= 1", wire.MethodAssemblyPatternCreate, in.Count)
	}
	return occurrence.CircularArrangement{
		Origin: math.P3(in.Origin[0], in.Origin[1], in.Origin[2]), Axis: axis, Step: in.Angle, Count: in.Count,
	}, nil
}

// rectangularArrangement builds a Count1×Count2 grid arrangement along two directions.
func rectangularArrangement(in wire.CreatePatternArgs) (occurrence.Arrangement, error) {
	d1, err := math.NewUnitVector3(in.Direction1[0], in.Direction1[1], in.Direction1[2])
	if err != nil {
		return nil, fmt.Errorf("%s: direction1 %v is not a direction: %w", wire.MethodAssemblyPatternCreate, in.Direction1, err)
	}
	d2, err := math.NewUnitVector3(in.Direction2[0], in.Direction2[1], in.Direction2[2])
	if err != nil {
		return nil, fmt.Errorf("%s: direction2 %v is not a direction: %w", wire.MethodAssemblyPatternCreate, in.Direction2, err)
	}
	if in.Count1 < 1 || in.Count2 < 1 {
		return nil, fmt.Errorf("%s: counts %d×%d must be >= 1", wire.MethodAssemblyPatternCreate, in.Count1, in.Count2)
	}
	return occurrence.RectangularArrangement{
		Dir1: d1, Spacing1: in.Spacing1, Count1: in.Count1, Dir2: d2, Spacing2: in.Spacing2, Count2: in.Count2,
	}, nil
}

// newOccurrencesReply marshals the occurrences an additive replication op created.
func newOccurrencesReply(created []*occurrence.Occurrence) (json.RawMessage, error) {
	out := make([]wire.OccurrenceInfo, len(created))
	for i, o := range created {
		out[i] = occurrenceInfo(o)
	}
	return json.Marshal(wire.NewOccurrencesResult{Created: out})
}
