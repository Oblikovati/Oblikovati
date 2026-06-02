// SPDX-License-Identifier: GPL-2.0-only

// Package doc is the document layer: the file/identity/lifecycle/reference unit,
// cleanly split from the modeling content it holds (the document/content split,
// parametric-cad §1b). A [Document] owns its display name, full document name,
// dirty flag and open/initialized state; the modeling payload lives behind the
// [Content] interface, whose concrete types arrive in later milestones (M07 part,
// M11 assembly, M14 drawing, M16 presentation).
//
// This separation is what lets one part be referenced by many assemblies: the
// document is the shared file identity, the content is the data (architecture
// core/05-documents-persistence-identity.md).
//
// The four document kinds are concrete specializations — [PartDocument],
// [AssemblyDocument], [DrawingDocument], [PresentationDocument] — each embedding
// the common [Document] base and exposing a typed content accessor. The base
// carries a [DocumentType] discriminator so the workspace (M03-F02) can handle
// open documents generically without RTTI (architecture core/02).
package doc
