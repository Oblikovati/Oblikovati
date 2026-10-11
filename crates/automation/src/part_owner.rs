// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, types::DOCUMENT_PART};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn active_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    let document = files.workspace().active_document().ok_or_else(|| {
        Error("active document is absent: expected an active part document".into())
    })?;
    let kind = files
        .document_type(document)
        .map_err(|cause| Error(cause.to_string()))?;
    if kind != DOCUMENT_PART {
        return Err(Error(format!(
            "document {document} kind {kind:?}: expected {DOCUMENT_PART:?} for a model query or command"
        )));
    }
    Ok(document)
}
