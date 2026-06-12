// SPDX-License-Identifier: GPL-2.0-only

package doc

import "oblikovati.org/api/contract"

// Document implements the Apache-2.0 in-process contract. This assertion keeps the
// implementation and the published interface honest at compile time (ADR-0018): if
// a method's signature drifts from [contract.Document], the build breaks here rather
// than silently diverging the public surface from what /source actually does.
var _ contract.Document = (*Document)(nil)

// The file surface (M03-F07, #608): the file object and the two descriptor
// views — one FileReference satisfies both the file-side flags contract and
// the document-side status/bridge contract.
var (
	_ contract.File                     = (*File)(nil)
	_ contract.FileDescriptor           = (*FileReference)(nil)
	_ contract.ReferencedFileDescriptor = (*FileReference)(nil)
)

// External-file attachments (M03-F08, #609).
var (
	_ contract.FileAttachment  = (*FileAttachment)(nil)
	_ contract.FileAttachments = (*FileAttachments)(nil)
)

// The add-in data registry (M03-F10, #611).
var (
	_ contract.DocumentInterest  = InterestRecordView{}
	_ contract.DocumentInterests = (*DocumentInterests)(nil)
)
