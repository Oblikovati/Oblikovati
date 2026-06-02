// SPDX-License-Identifier: GPL-2.0-only

// Package compdef holds component definitions — the modeling content a document
// owns, cleanly split from the document's file/identity/lifecycle (parametric-cad
// §1b, architecture core/05). A [PartComponentDefinition] is the root the feature
// engine (M08) operates within: it owns the part's surface bodies (kernel/topo),
// parameters (model/param), sketches (model/sketch), bounding boxes, a model-
// geometry version that changes on every edit, and the end-of-part rollback marker.
//
// It implements doc.Content (compdef → doc, one-way: doc never imports compdef), so
// a part document carries a *PartComponentDefinition via [doc.Document.SetContent]
// and callers retrieve it by type-asserting [doc.Document.Content].
package compdef
