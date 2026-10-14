// SPDX-License-Identifier: GPL-2.0-only
use crate::encode;
use oblikovati_api::{Error, wire::*};
use oblikovati_model::sketch_settings::SketchSettings;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Read persisted settings without making the document's default record explicit.
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: GetSketchSettingsArgs,
) -> Result<Vec<u8>, Error> {
    let document = selected_document(files, request.document)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    encode(&SketchSettingsResult {
        settings: session.snapshot().effective_sketch_settings(),
    })
}

/// Replace one document's settings through its history owner.
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetSketchSettingsArgs,
) -> Result<Vec<u8>, Error> {
    let document = selected_document(files, request.document)?;
    let result = store(files, document, request.settings)?;
    encode(&result)
}

fn store<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    settings: SketchSettings,
) -> Result<SketchSettingsResult, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run("Set sketch settings", |snapshot, _| {
            snapshot.sketch_settings = Some(settings.clone());
            Ok::<_, String>(SketchSettingsResult { settings })
        })
        .map_err(api_error)
}

fn selected_document<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    requested: u64,
) -> Result<u64, Error> {
    let document = active_or_requested(files, requested)?;
    files.workspace().session(document).map_err(api_error)?;
    Ok(document)
}

fn active_or_requested<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    requested: u64,
) -> Result<u64, Error> {
    if requested != 0 {
        return Ok(requested);
    }
    files.workspace().active_document().ok_or_else(|| {
        Error("active document is absent: expected an active document for sketch settings".into())
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
