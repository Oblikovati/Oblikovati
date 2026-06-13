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
	"oblikovati.org/model/feature"
)

// The assembly derive/shrinkwrap surface (M11-F06, #631/#716): derive a source
// assembly into the active part as a base body — optionally simplified into a
// lightweight shrinkwrap — and break the link to freeze the result. Each create adds a
// feature to the active part and returns its refreshed detail; break-link addresses
// the feature by its stable id.

// registerAssemblyDeriveHandlers wires the assembly.* derive/shrinkwrap methods.
func (r *Router) registerAssemblyDeriveHandlers() {
	r.handlers[wire.MethodAssemblyDeriveCreate] = assemblyDeriveCreate
	r.handlers[wire.MethodAssemblyShrinkwrapCreate] = assemblyShrinkwrapCreate
	r.handlers[wire.MethodAssemblyDeriveBreakLink] = assemblyDeriveBreakLink
}

// assemblyDeriveCreate derives the source assembly document into the active part as a
// base body (include-all) and returns the new feature's detail.
func assemblyDeriveCreate(s *app.Session, raw json.RawMessage) (json.RawMessage, error) {
	part, source, err := resolveDeriveSource(s, raw, wire.MethodAssemblyDeriveCreate)
	if err != nil {
		return nil, err
	}
	pf := feature.NewDerivedAssemblyComponents(part.Features()).AddDerived(source)
	return commitNewDerive(part, pf)
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
	source, err := assemblySource(s, in.Source, wire.MethodAssemblyShrinkwrapCreate)
	if err != nil {
		return nil, err
	}
	def, err := shrinkwrapDefinition(in)
	if err != nil {
		return nil, err
	}
	pf := feature.NewShrinkwrapComponents(part.Features()).AddShrinkwrap(source, def)
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

// resolveDeriveSource resolves the active part and the DeriveCreateArgs source
// assembly, shared by the derive create handler.
func resolveDeriveSource(s *app.Session, raw json.RawMessage, method string) (*compdef.PartComponentDefinition, feature.AssemblyBodySource, error) {
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		return nil, nil, err
	}
	var in wire.DeriveCreateArgs
	if err := decode(raw, &in); err != nil {
		return nil, nil, err
	}
	source, err := assemblySource(s, in.Source, method)
	if err != nil {
		return nil, nil, err
	}
	return part, source, nil
}

// assemblySource resolves an open document id to its assembly body source, rejecting a
// non-assembly document (a part has no occurrence tree to derive).
func assemblySource(s *app.Session, id uint64, method string) (feature.AssemblyBodySource, error) {
	d, err := documentByID(s, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method, err)
	}
	source, ok := d.Content().(feature.AssemblyBodySource)
	if !ok {
		return nil, fmt.Errorf("%s: document %d (%s) is not an assembly", method, id, d.DisplayName())
	}
	return source, nil
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
		RemoveStyle:   remove,
		MinPartVolume: in.MinPartVolume,
		EnvelopeStyle: envelope,
		PatchHoles:    in.PatchHoles,
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
