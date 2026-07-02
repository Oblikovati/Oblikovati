// SPDX-License-Identifier: GPL-2.0-only

// Package drawing is the modeling content of a drawing document (M14-F01, #384): an
// ordered set of sheets (sizes/orientation, borders, title blocks) plus the primary
// referenced model the drawing documents. Title-block fields resolve against that
// model's iProperties, so a title block shows live part/assembly metadata.
//
// The entry point is [Content] (built by [NewContent]), the doc.Content
// implementation holding the sheets. The package knows nothing of the workspace:
// the host injects resolver seams — [ModelProperties] for iProperty values,
// [BodyResolver] for the referenced B-rep a view projects, [BOMResolver] for
// parts-list rows — and a nil resolver degrades gracefully (blank fields,
// unprojected views, empty lists) rather than failing.
//
// This is the GPL implementation of the api/contract drawing surface (ADR-0018); it
// registers itself as the real content for doc.Drawing so opening a .odd reconstructs
// live sheets rather than the identity-only stub.
package drawing
