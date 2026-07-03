// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/doc"
	"oblikovati.org/model/feature"
)

// The assembly derive/shrinkwrap surface (M11-F06, #631/#716): derive a source
// assembly into the active part as a base body — optionally simplified into a
// lightweight shrinkwrap — and break the link to freeze the result. Each create adds a
// feature to the active part and returns its refreshed detail; break-link addresses
// the feature by its stable id.

// registerAssemblyDeriveHandlers wires the assembly.* derive/shrinkwrap methods.
func (r *Router) registerAssemblyDeriveHandlers() {
	r.mutating(wire.MethodAssemblyDeriveCreate, "Derive Component", assemblyDeriveCreate)
	r.mutating(wire.MethodAssemblyShrinkwrapCreate, "Shrinkwrap", assemblyShrinkwrapCreate)
	r.mutating(wire.MethodAssemblyDeriveBreakLink, "Break Link", assemblyDeriveBreakLink)
	r.readOnly(wire.MethodAssemblyDeriveStatus, typedPart(assemblyDeriveStatus))
	r.readOnly(wire.MethodAssemblyDeriveUpdate, typedPart(assemblyDeriveUpdate))
}

// assemblyDeriveStatus reports the drive state of a derive-family feature: whether it is
// out of date relative to its source, and the source's saved vs current revision (#751).
func assemblyDeriveStatus(s *app.Session, part *compdef.PartComponentDefinition, in wire.DeriveStatusArgs) (wire.DeriveStatusResult, error) {
	derive, err := deriveStatusFeatureByID(part, in.ID, wire.MethodAssemblyDeriveStatus)
	if err != nil {
		return wire.DeriveStatusResult{}, err
	}
	return deriveStatusResult(s, derive), nil
}

// assemblyDeriveUpdate re-syncs a derive to its source's current revision, clearing its
// out-of-date state, and recomputes the part (#751).
func assemblyDeriveUpdate(s *app.Session, part *compdef.PartComponentDefinition, in wire.DeriveStatusArgs) (wire.DeriveStatusResult, error) {
	derive, err := deriveStatusFeatureByID(part, in.ID, wire.MethodAssemblyDeriveUpdate)
	if err != nil {
		return wire.DeriveStatusResult{}, err
	}
	derive.AcknowledgeSource(currentSourceRevision(s, derive.SourceLink().Document))
	part.Recompute()
	return deriveStatusResult(s, derive), nil
}

// deriveStatusFeatureByID resolves the derive-family feature addressed by id in the part's
// program, erroring when the feature is not a derive.
func deriveStatusFeatureByID(part *compdef.PartComponentDefinition, id uint64, method string) (feature.DeriveStatus, error) {
	pf, _, err := partFeatureByID(part, id, method)
	if err != nil {
		return nil, err
	}
	derive, ok := pf.Definition().(feature.DeriveStatus)
	if !ok {
		return nil, fmt.Errorf("%s: feature %d is a %s, not a derived/shrinkwrap component", method, id, pf.Kind())
	}
	return derive, nil
}

// deriveStatusResult renders a derive's drive state, resolving the source's current
// revision through the workspace (empty when the source is not open/resolvable).
func deriveStatusResult(s *app.Session, derive feature.DeriveStatus) wire.DeriveStatusResult {
	link := derive.SourceLink()
	return wire.DeriveStatusResult{
		OutOfDate:       derive.OutOfDate(),
		Linked:          derive.Linked(),
		SourceDocument:  link.Document,
		SavedRevision:   link.DatabaseRevisionID,
		CurrentRevision: currentSourceRevision(s, link.Document),
	}
}

// currentSourceRevision returns the source document's current recipe revision, or "" when
// the document is not open in the workspace (so the drive state cannot be compared).
func currentSourceRevision(s *app.Session, sourceName string) string {
	if sourceName == "" {
		return ""
	}
	d, ok := s.Workspace().ByName(sourceName)
	if !ok {
		return ""
	}
	return d.FileIdentity().DatabaseRevisionID
}

// assemblyDeriveCreate derives the source assembly document into the active part as a
// base body (include-all) and returns the new feature's detail. The source's identity is
// captured on the derive and recorded as a reference of the active part, so the link
// survives a save and a stale source is detected on reopen (#715).
func assemblyDeriveCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, source, sourceDoc, err := resolveDeriveSource(s, raw, wire.MethodAssemblyDeriveCreate)
	if err != nil {
		return nil, err
	}
	pf := feature.NewDerivedAssemblyComponents(part.Features()).AddDerived(source, deriveSourceLink(sourceDoc))
	// Record the part→source edge now so the first save snapshots it (mirrors
	// PlaceComponentFromFile); the source is already open, so this just resolves it.
	s.ActiveDocument().OpenReference(sourceDoc.FullDocumentName())
	return commitNewDerive(part, pf)
}

// deriveSourceLink captures the source assembly document's identity for a derive link.
func deriveSourceLink(sourceDoc *doc.Document) feature.DeriveSourceLink {
	id := sourceDoc.FileIdentity()
	return feature.DeriveSourceLink{
		Document:           sourceDoc.FullDocumentName(),
		InternalName:       id.InternalName,
		DatabaseRevisionID: id.DatabaseRevisionID,
	}
}

// assemblyShrinkwrapCreate derives the source assembly into the active part as a
// simplified, lightweight base body per the removal/envelope options.
func assemblyShrinkwrapCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.ShrinkwrapCreateArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	source, sourceDoc, err := assemblySource(s, in.Source, wire.MethodAssemblyShrinkwrapCreate)
	if err != nil {
		return nil, err
	}
	def, err := shrinkwrapDefinition(in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewShrinkwrapComponents(part.Features()).AddShrinkwrap(source, def, deriveSourceLink(sourceDoc))
	// Record the part→source edge so the first save snapshots the link (the source is open).
	s.ActiveDocument().OpenReference(sourceDoc.FullDocumentName())
	return commitNewDerive(part, pf)
}

// assemblyDeriveBreakLink freezes and severs a derived-assembly or shrinkwrap feature's
// source link, addressed by its stable id.
func assemblyDeriveBreakLink(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, err
	}
	var in wire.DeriveBreakLinkArgs
	if err := decode(raw, &in); err != nil {
		return nil, err
	}
	pf, idx, err := partFeatureByID(part, in.ID, wire.MethodAssemblyDeriveBreakLink)
	if err != nil {
		return nil, err
	}
	linked, ok := pf.Definition().(interface{ BreakLink() error })
	if !ok {
		return nil, fmt.Errorf("%s: feature %d is a %s, not a derived/shrinkwrap component",
			wire.MethodAssemblyDeriveBreakLink, in.ID, pf.Kind())
	}
	if err := linked.BreakLink(); err != nil {
		return nil, fmt.Errorf("%s: %w", wire.MethodAssemblyDeriveBreakLink, err)
	}
	part.Recompute()
	return featureDetailReply(part, pf, idx)
}

// resolveDeriveSource resolves the active part and the DeriveCreateArgs source assembly
// document, shared by the derive create handler. The source document is returned so the
// caller can capture its identity for the derive link (#715).
func resolveDeriveSource(s *app.Session, raw json.RawMessage, method string) (*compdef.PartComponentDefinition, feature.AssemblyBodySource, *doc.Document, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, nil, err
	}
	var in wire.DeriveCreateArgs
	if err := decode(raw, &in); err != nil {
		return nil, nil, nil, err
	}
	source, sourceDoc, err := assemblySource(s, in.Source, method)
	if err != nil {
		return nil, nil, nil, err
	}
	return part, source, sourceDoc, nil
}

// assemblySource resolves an open document id to its assembly body source and document,
// rejecting a non-assembly document (a part has no occurrence tree to derive).
func assemblySource(s *app.Session, id uint64, method string) (feature.AssemblyBodySource, *doc.Document, error) {
	d, err := documentByID(s, id)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", method, err)
	}
	source, ok := d.Content().(feature.AssemblyBodySource)
	if !ok {
		return nil, nil, fmt.Errorf("%s: document %d (%s) is not an assembly", method, id, d.DisplayName())
	}
	return source, d, nil
}

// commitNewDerive gives the new derive/shrinkwrap feature a unique name, recomputes the
// part, and replies with its detail.
func commitNewDerive(part *compdef.PartComponentDefinition, pf *feature.PartFeature) (json.RawMessage, error) {
	pf.SetName(part.Features().UniqueName(pf.Kind()))
	part.Recompute()
	_, idx, err := partFeatureByID(part, uint64(pf.ID()), "assembly.derive")
	if err != nil {
		return nil, err
	}
	return featureDetailReply(part, pf, idx)
}

// shrinkwrapDefinition maps the wire options to the model shrinkwrap recipe.
func shrinkwrapDefinition(in wire.ShrinkwrapCreateArgs) (feature.ShrinkwrapDefinition, error) {
	remove, err := removeStyle(in.RemoveStyle)
	if err != nil {
		return feature.ShrinkwrapDefinition{}, err
	}
	envelope, err := envelopeStyle(in.EnvelopeStyle)
	if err != nil {
		return feature.ShrinkwrapDefinition{}, err
	}
	return feature.ShrinkwrapDefinition{
		RemoveStyle:     remove,
		MinPartVolume:   in.MinPartVolume,
		EnvelopeStyle:   envelope,
		PatchHoles:      in.PatchHoles,
		MaxHoleDiameter: in.MaxHoleDiameter,
	}, nil
}

// removeStyle maps the public remove style to the model one.
func removeStyle(s types.ShrinkwrapRemoveStyle) (feature.ShrinkwrapRemoveStyle, error) {
	switch s {
	case types.RemoveNone:
		return feature.RemoveNone, nil
	case types.RemoveSmallParts:
		return feature.RemoveSmallParts, nil
	case types.RemoveInternalParts:
		return feature.RemoveInternalParts, nil
	}
	return 0, fmt.Errorf("%s: unknown shrinkwrap remove style %d", wire.MethodAssemblyShrinkwrapCreate, int32(s))
}

// envelopeStyle maps the public envelope style to the model one.
func envelopeStyle(s types.ShrinkwrapEnvelopeStyle) (feature.ShrinkwrapEnvelopeStyle, error) {
	switch s {
	case types.EnvelopeNone:
		return feature.EnvelopeNone, nil
	case types.EnvelopePerPart:
		return feature.EnvelopePerPart, nil
	case types.EnvelopeWhole:
		return feature.EnvelopeWhole, nil
	}
	return 0, fmt.Errorf("%s: unknown shrinkwrap envelope style %d", wire.MethodAssemblyShrinkwrapCreate, int32(s))
}
