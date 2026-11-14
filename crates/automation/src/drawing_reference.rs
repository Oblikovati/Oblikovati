// SPDX-License-Identifier: GPL-2.0-only
//! The existing drawing contract edits the active drawing through its history owner.
use oblikovati_api::{
    Error,
    types::DOCUMENT_DRAWING,
    wire::{SetModelReferenceArgs, SetModelReferenceResult},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetModelReferenceArgs,
) -> Result<Vec<u8>, Error> {
    let document = files.workspace().active_document().ok_or_else(|| {
        Error("drawing model reference: expected an active drawing document".into())
    })?;
    let kind = files.document_type(document).map_err(api_error)?;
    if kind != DOCUMENT_DRAWING {
        return Err(Error(format!(
            "drawing model reference document {document}, type {kind:?}: expected a drawing"
        )));
    }
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set Model Reference",
            |snapshot, _| {
                let drawing = snapshot.drawing.as_mut().ok_or_else(|| {
                    Error(format!(
                        "drawing document {document}: expected drawing content"
                    ))
                })?;
                drawing
                    .set_model_reference(&request.full_document_name)
                    .map_err(api_error)?;
                Ok::<_, Error>(drawing.model_reference().to_string())
            },
            |_, reference| {
                crate::encode(&SetModelReferenceResult {
                    model_reference: reference,
                })
            },
        )
        .map_err(api_error)
}
fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
