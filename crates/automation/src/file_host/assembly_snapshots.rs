// SPDX-License-Identifier: GPL-2.0-only
//! Stage a qualified presentation without changing the outcome of the model save.
use super::*;
use oblikovati_api::{
    types::DOCUMENT_ASSEMBLY,
    wire::{SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE, SaveWarning},
};
use oblikovati_persistence::package::SavedAssemblyView;
use std::path::Path;

pub(super) fn stage<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&mut (dyn crate::PartTopologyAccess<O, W, F> + '_)>,
    document: u64,
    destination: &Path,
) -> Result<Option<SaveWarning>, String> {
    files.clear_assembly_view(document);
    if files
        .document_type(document)
        .map_err(|error| error.to_string())?
        != DOCUMENT_ASSEMBLY
    {
        return Ok(None);
    }
    let destination = match files.assembly_save_destination(destination) {
        Ok(path) => path,
        Err(reason) => return Ok(Some(warning(document, reason))),
    };
    match capture(files, backend, document, &destination) {
        Ok(view) => match files.stage_assembly_view(document, &destination, view) {
            Ok(()) => Ok(None),
            Err(reason) => Ok(Some(warning(document, reason))),
        },
        Err(reason) => Ok(Some(warning(document, reason))),
    }
}
fn capture<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&mut (dyn crate::PartTopologyAccess<O, W, F> + '_)>,
    document: u64,
    destination: &Path,
) -> Result<SavedAssemblyView, String> {
    let sources = files.assembly_saved_sources(document, destination)?;
    let catalog_digest = files.assembly_view_catalog_digest()?;
    let occurrences = files
        .workspace()
        .session(document)
        .map_err(|error| error.to_string())?
        .snapshot()
        .assembly
        .clone()
        .ok_or_else(|| format!("document {document}: expected an assembly"))?;
    let backend =
        backend.ok_or_else(|| format!("assembly {document}: expected a presentation service"))?;
    let (triangles, chord_error, angular_error) = backend.assembly_view(files, document)?;
    let view = SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences,
        sources,
        catalog_digest,
        chord_error,
        angular_error,
        triangles,
    };
    view.validate_content().map_err(|error| error.to_string())?;
    Ok(view)
}
fn warning(document: u64, detail: String) -> SaveWarning {
    SaveWarning {
        document,
        code: SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE.into(),
        detail: format!("Express assembly cache unavailable: {detail}"),
    }
}
