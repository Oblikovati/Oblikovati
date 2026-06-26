// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
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
	r.mutating(wire.MethodAssemblyPatternCreate, "Pattern Components", assemblyPatternCreate)
	r.mutating(wire.MethodAssemblyMirror, "Mirror Components", assemblyMirror)
	r.mutating(wire.MethodAssemblyMirrorIntoPart, "Mirror", assemblyMirrorIntoPart)
	r.mutating(wire.MethodAssemblyCopy, "Copy Component", assemblyCopy)
	r.mutating(wire.MethodAssemblySubstitute, "Substitute Component", assemblySubstitute)
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
	return newOccurrencesReply(occurrence.PatternComponents(asm.Occurrences(), seed, pat))
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

// assemblyMirrorIntoPart mirrors each source occurrence into a NEW opposite-hand part
// document and places that as the mirrored occurrence (#717) — the document-producing
// counterpart of assemblyMirror.
func assemblyMirrorIntoPart(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	asm, err := modelaccess.ActiveAssembly(s)
	if err != nil {
		return nil, err
	}
	sources, reflect, err := parseMirrorIntoPart(asm, raw)
	if err != nil {
		return nil, err
	}
	asmDoc := s.ActiveDocument()
	created := make([]*occurrence.Occurrence, 0, len(sources))
	for _, src := range sources {
		occ, err := mirrorOccurrenceIntoPart(s, asm, asmDoc, src, reflect)
		if err != nil {
			return nil, err
		}
		created = append(created, occ)
	}
	// AddPart activated each new mirror part; leave the assembly active, as assemblyMirror does.
	if err := s.Workspace().SetActiveDocument(asmDoc); err != nil {
		return nil, err
	}
	return newOccurrencesReply(created)
}

// parseMirrorIntoPart resolves the source occurrences and the mirror-plane reflection from
// the request.
func parseMirrorIntoPart(asm *compdef.AssemblyComponentDefinition, raw json.RawMessage) ([]*occurrence.Occurrence, math.Matrix4, error) {
	var in wire.MirrorIntoPartArgs
	if err := decode(raw, &in); err != nil {
		return nil, math.Matrix4{}, err
	}
	sources, err := occurrencesByID(asm, in.Sources, wire.MethodAssemblyMirrorIntoPart)
	if err != nil {
		return nil, math.Matrix4{}, err
	}
	normal, err := math.NewUnitVector3(in.Normal[0], in.Normal[1], in.Normal[2])
	if err != nil {
		return nil, math.Matrix4{}, fmt.Errorf("%s: normal %v is not a direction: %w", wire.MethodAssemblyMirrorIntoPart, in.Normal, err)
	}
	return sources, math.Reflection4(math.P3(in.Origin[0], in.Origin[1], in.Origin[2]), normal), nil
}

// mirrorOccurrenceIntoPart derives src's component into a new opposite-hand part and places
// it so the occurrence's WORLD geometry equals the plane-mirror of the source. The new
// part's local geometry is the source reflected about its own YZ plane (M), and the
// occurrence sits at T = reflect·S·M — which has positive determinant, so the handedness
// lives in the part, not the placement (a real opposite-hand part, not a reflected instance).
func mirrorOccurrenceIntoPart(s *app.Session, asm *compdef.AssemblyComponentDefinition, asmDoc *doc.Document, src *occurrence.Occurrence, reflect math.Matrix4) (*occurrence.Occurrence, error) {
	sourceName := src.ComponentName()
	if sourceName == "" {
		return nil, fmt.Errorf("%s: occurrence %q is not file-backed; mirror-into-part needs a source document", wire.MethodAssemblyMirrorIntoPart, src.Name())
	}
	sourceDoc, ok := s.Workspace().ByName(sourceName)
	if !ok {
		return nil, fmt.Errorf("%s: source document %q is not open", wire.MethodAssemblyMirrorIntoPart, sourceName)
	}
	local := localMirror()
	mirroredDoc, err := deriveMirrorPart(s, sourceDoc, sourceName, local)
	if err != nil {
		return nil, err
	}
	placement := reflect.Mul(src.Transform()).Mul(local) // T = reflect·S·M
	return asm.PlaceComponentFromFile(asmDoc, mirroredDoc, src.Name()+"-mirror", placement)
}

// deriveMirrorPart creates a new part document whose geometry is sourceDoc's body reflected
// by local (the opposite-hand part), linked to the source for re-derive on open, and
// returns it. The part→source reference is recorded so a save snapshots the link.
func deriveMirrorPart(s *app.Session, sourceDoc *doc.Document, sourceName string, local math.Matrix4) (*doc.Document, error) {
	source, ok := sourceDoc.Content().(feature.BodySource)
	if !ok {
		return nil, fmt.Errorf("%s: source %q is not a part", wire.MethodAssemblyMirrorIntoPart, sourceName)
	}
	mirroredDoc, err := compdef.AddPart(s.Workspace(), mirrorPartName(sourceName), false)
	if err != nil {
		return nil, fmt.Errorf("%s: create mirror part: %w", wire.MethodAssemblyMirrorIntoPart, err)
	}
	mirroredPart := mirroredDoc.Content().(*compdef.PartComponentDefinition)
	id := sourceDoc.FileIdentity()
	link := feature.DeriveSourceLink{Document: sourceName, InternalName: id.InternalName, DatabaseRevisionID: id.DatabaseRevisionID}
	feature.NewDerivedComponents(mirroredPart.Features()).AddDerived(source, local, link)
	mirroredDoc.OpenReference(sourceName) // mirror-part → source edge, for save + re-derive on open
	mirroredPart.Recompute()
	return mirroredDoc, nil
}

// localMirror is the reflection about a part's local YZ plane — the opposite-hand baking
// transform. The unit X is always valid, so the construction cannot fail.
func localMirror() math.Matrix4 {
	normal, _ := math.NewUnitVector3(1, 0, 0)
	return math.Reflection4(math.P3(0, 0, 0), normal)
}

// mirrorPartName derives the mirror part's document name from the source's:
// "<base>-mirror<ext>" (e.g. widget.obk → widget-mirror.obk).
func mirrorPartName(sourceName string) string {
	ext := filepath.Ext(sourceName)
	return strings.TrimSuffix(sourceName, ext) + "-mirror" + ext
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
