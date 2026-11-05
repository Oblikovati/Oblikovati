// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    types::{DOCUMENT_PART, DocumentType},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(crate) fn active_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    require_active_kind(files, &[DOCUMENT_PART], "part document")
}
pub(crate) fn require_active_kind<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    accepted: &[DocumentType],
    purpose: &str,
) -> Result<u64, Error> {
    let document = files.workspace().active_document().ok_or_else(|| {
        Error(format!(
            "active document is absent: expected an active {purpose}"
        ))
    })?;
    let kind = files
        .document_type(document)
        .map_err(|cause| Error(cause.to_string()))?;
    if !accepted.contains(&kind) {
        return Err(Error(format!(
            "document {document} kind {kind:?}: expected {purpose} with kind in {accepted:?}"
        )));
    }
    Ok(document)
}
