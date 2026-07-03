// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// Derived parameter tables over the wire (M02-F06, Oblikovati#605):
// parameters.derivedTables.list/add/setLinked/delete. Mutations go through
// the session seams, which resolve the source document, keep the workspace
// reference graph current, and record one undo step each.

// derivedTableInfo marshals one table, resolving the live candidate list from
// the source document (empty when the source is not reachable).
func derivedTableInfo(s *app.Session, t *param.DerivedParameterTable) wire.DerivedParameterTableInfo {
	info := wire.DerivedParameterTableInfo{
		ID: t.ID(), SourceDocument: t.SourceDocument(), Linked: t.Linked(),
		Health:                t.Health().Reason,
		References:            derivedReferences(t),
		HasReferenceComponent: t.OwnedByFeature(),
	}
	if t.OwnedByFeature() {
		info.ReferenceComponent = t.SourceDocument()
	}
	if source, ok := s.LinkableSourceParameters(t.SourceDocument()); ok {
		for _, sv := range source {
			info.Available = append(info.Available, sv.Name)
		}
	} else if info.Health == "" {
		info.Health = "source document " + t.SourceDocument() + " is unavailable"
	}
	return info
}

// derivedReferences projects a table's produced derived parameters to their wire references —
// each derived parameter paired with the source document and parameter it tracks (#1561).
func derivedReferences(t *param.DerivedParameterTable) []wire.DerivedParameterReference {
	refs := t.References()
	if len(refs) == 0 {
		return nil
	}
	out := make([]wire.DerivedParameterReference, len(refs))
	for i, r := range refs {
		out[i] = wire.DerivedParameterReference{
			Parameter: r.DerivedName, SourceDocument: r.SourceDocument, SourceParameter: r.SourceName,
		}
	}
	return out
}

// listDerivedTables returns the active part's or assembly's tables with live candidates.
func listDerivedTables(s *app.Session, holder compdef.ParameterHolder) (wire.ListDerivedParameterTablesResult, error) {
	var out wire.ListDerivedParameterTablesResult
	for _, t := range holder.Parameters().DerivedTables() {
		out.Tables = append(out.Tables, derivedTableInfo(s, t))
	}
	return out, nil
}

// addDerivedTable links parameters from another open document into this one.
func addDerivedTable(s *app.Session, in wire.DerivedParameterTableAddArgs) (wire.DerivedParameterTableInfo, error) {
	t, err := s.AddDerivedParameterTable(in.SourceDocument, in.Linked)
	if err != nil {
		return wire.DerivedParameterTableInfo{}, err
	}
	return derivedTableInfo(s, t), nil
}

// setDerivedTableLinked replaces a table's linked subset.
func setDerivedTableLinked(s *app.Session, args json.RawMessage) (json.RawMessage, error) {
	var in wire.DerivedParameterTableSetLinkedArgs
	if err := decode(args, &in); err != nil {
		return nil, err
	}
	if err := s.SetDerivedTableLinked(in.ID, in.Linked); err != nil {
		return nil, err
	}
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return nil, err
	}
	t, ok := holder.Parameters().DerivedTableByID(in.ID)
	if !ok {
		return json.Marshal(struct{}{})
	}
	return json.Marshal(derivedTableInfo(s, t))
}

// deleteDerivedTable removes a table and its derived parameters.
func deleteDerivedTable(s *app.Session, in wire.DerivedParameterTableDeleteArgs) (struct{}, error) {
	if err := s.DeleteDerivedParameterTable(in.ID); err != nil {
		return struct{}{}, err
	}
	return struct{}{}, nil
}
