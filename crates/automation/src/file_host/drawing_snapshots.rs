// SPDX-License-Identifier: GPL-2.0-only
//! Capture saved drawing presentation from the same projection service as the live view.
use super::*;
use oblikovati_api::types::{DOCUMENT_DRAWING, DOCUMENT_PART};
use oblikovati_persistence::package::{SavedDrawingView, SavedDrawingViews};

pub(super) fn stage<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn crate::DrawingViewProjectionAccess>,
    id: u64,
) -> Result<(), String> {
    if files.document_type(id).map_err(|cause| cause.to_string())? != DOCUMENT_DRAWING {
        return Ok(());
    }
    let saved = capture(files, backend, id)?;
    files
        .keep_drawing_views(id, saved)
        .map_err(|cause| cause.to_string())
}
fn capture<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    backend: Option<&dyn crate::DrawingViewProjectionAccess>,
    id: u64,
) -> Result<SavedDrawingViews, String> {
    let drawing = files
        .workspace()
        .session(id)
        .map_err(|cause| cause.to_string())?
        .snapshot()
        .drawing
        .as_ref()
        .ok_or_else(|| format!("drawing {id}: expected drawing content"))?;
    if let Some(frozen) = files
        .deferred_drawing_views(id)
        .map_err(|cause| cause.to_string())?
    {
        if drawing.recipe() != &frozen.drawing_recipe {
            return Err(format!(
                "deferred drawing {id}: expected the saved drawing recipe before saving frozen views"
            ));
        }
        return Ok(frozen.clone());
    }
    let mut saved = SavedDrawingViews {
        version: SavedDrawingViews::VERSION,
        drawing_recipe: drawing.recipe().clone(),
        views: Vec::new(),
    };
    if drawing.sheets().iter().all(|sheet| sheet.views.is_empty()) {
        return Ok(saved);
    }
    saved.views = capture_from_source(files, drawing, backend, id)?;
    Ok(saved)
}
fn capture_from_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    drawing: &oblikovati_model::drawing::DrawingDocument,
    backend: Option<&dyn crate::DrawingViewProjectionAccess>,
    id: u64,
) -> Result<Vec<SavedDrawingView>, String> {
    let source = drawing_source(files, id, drawing.model_reference())?;
    let backend = backend
        .ok_or_else(|| format!("drawing {id}: expected a projection service for saved views"))?;
    let session = files
        .workspace()
        .session(source)
        .map_err(|cause| cause.to_string())?;
    let guid = files
        .workspace()
        .identity(source)
        .map_err(|cause| cause.to_string())?
        .internal_name;
    sample_views(drawing, session.history(), guid, backend)
}
fn drawing_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    id: u64,
    name: &str,
) -> Result<u64, String> {
    let source = match files.drawing_reference(id, name) {
        Some(source) => Some(source),
        None => files
            .resolve_parameter_source(name)
            .map_err(|cause| cause.to_string())?,
    }
    .ok_or_else(|| {
        format!("drawing {id} model {name:?}: expected a loaded part for saved views")
    })?;
    if files
        .document_type(source)
        .map_err(|cause| cause.to_string())?
        != DOCUMENT_PART
    {
        return Err(format!(
            "drawing {id} model {name:?}: expected a part for saved views"
        ));
    }
    Ok(source)
}
fn sample_views(
    drawing: &oblikovati_model::drawing::DrawingDocument,
    history: &oblikovati_model::history::DocumentEditHistory,
    guid: oblikovati_model::identity::DocumentGuid,
    backend: &dyn crate::DrawingViewProjectionAccess,
) -> Result<Vec<SavedDrawingView>, String> {
    let mut views = Vec::new();
    for sheet in drawing.sheets() {
        for view in &sheet.views {
            views.push(SavedDrawingView {
                sheet: sheet.name.clone(),
                view: view.name.clone(),
                lines: backend.saved_lines(history, guid, view)?,
            });
        }
    }
    Ok(views)
}
