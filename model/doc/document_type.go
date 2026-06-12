// SPDX-License-Identifier: GPL-2.0-only

package doc

import "oblikovati.org/api/types"

// DocumentType discriminates the four document kinds plus an unknown sentinel. The
// type and its stable values are defined once in the Apache-2.0 contract
// ([types.DocumentType]); this alias keeps the historical doc.DocumentType /
// doc.Part spelling working unchanged across the implementation (architecture
// core/05, ADR-0018). The values are persisted in package manifests and the
// reference graph and must never be renumbered — see [types.DocumentType].
type DocumentType = types.DocumentType

// SubTypeID refines the base DocumentType with a flavored sub-type — the
// persisted discriminator behind sheet-metal parts and add-in flavors
// (M03-F11, #612). Defined once in the contract ([types.DocumentSubTypeID]);
// this alias keeps doc-layer spellings short.
type SubTypeID = types.DocumentSubTypeID

const (
	// Unknown is the zero value: a document whose kind has not been resolved
	// (e.g. an unresolved reference stub before its manifest is read).
	Unknown = types.DocumentUnknown
	// Part holds a single modeled part (PartComponentDefinition, M07).
	Part = types.DocumentPart
	// Assembly holds component occurrences and constraints (AssemblyComponentDefinition, M11).
	Assembly = types.DocumentAssembly
	// Drawing holds annotated sheets/views of other documents (M14).
	Drawing = types.DocumentDrawing
	// Presentation holds exploded/animated views of an assembly (M16).
	Presentation = types.DocumentPresentation
)

// PackageExtension is the on-disk file extension for every document kind. Unlike
// COM's per-type extensions (.ipt/.iam/.idw/.ipn), the modern format is one
// portable zip package whose manifest carries the root [DocumentType]
// (architecture core/05).
const PackageExtension = ".obk"
