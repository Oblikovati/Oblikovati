// SPDX-License-Identifier: GPL-2.0-only
//! Read or edit authored views on the drawing's active sheet.
use super::*;
use oblikovati_api::wire::{
    DeleteViewArgs, ListDrawingViewsResult, RotateViewArgs, SetViewLabelArgs,
};
use oblikovati_model::drawing::{DrawingDocument, DrawingViewLabelEdit};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let owner = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("drawing views: expected an active drawing".into()))?;
    let mut drawing = active_drawing(files, owner)?.clone();
    if method == wire::METHOD_DRAWING_VIEWS_LIST {
        if !request.is_empty() {
            crate::decode::<oblikovati_api::json::EmptyObject>(method, request)?;
        }
        return response(files, backend, replay, owner, &drawing);
    }
    if files
        .deferred_drawing_views(owner)
        .map_err(api_error)?
        .is_some()
    {
        return Err(Error(format!(
            "drawing {owner}: reopen with updates enabled before editing views"
        )));
    }
    let command = edit_view(&mut drawing, method, request)?;
    let reply = response(files, backend, replay, owner, &drawing)?;
    files
        .commands(owner)
        .map_err(api_error)?
        .run(command, |snapshot, _| {
            snapshot.drawing = Some(drawing);
            Ok::<(), Error>(())
        })
        .map_err(api_error)?;
    Ok(reply)
}

fn edit_view(
    drawing: &mut DrawingDocument,
    method: &str,
    request: &[u8],
) -> Result<&'static str, Error> {
    let sheet_index = drawing.active_sheet();
    match method {
        wire::METHOD_DRAWING_VIEWS_DELETE => {
            let args: DeleteViewArgs = crate::decode(method, request)?;
            drawing
                .remove_view(sheet_index, &args.name)
                .map_err(api_error)?;
            Ok("Delete Drawing View")
        }
        wire::METHOD_DRAWING_VIEWS_ROTATE => {
            let args: RotateViewArgs = crate::decode(method, request)?;
            drawing
                .rotate_view(sheet_index, &args.name, args.angle_deg)
                .map_err(api_error)?;
            Ok("Rotate View")
        }
        wire::METHOD_DRAWING_VIEWS_SET_LABEL => {
            let args: SetViewLabelArgs = crate::decode(method, request)?;
            drawing
                .set_view_label(
                    sheet_index,
                    &args.name,
                    DrawingViewLabelEdit {
                        text: args.text.as_deref().map(String::as_str),
                        show_label: args.show_label.map(|show| *show),
                        show_name: args.show_name.map(|show| *show),
                        show_scale: args.show_scale.map(|show| *show),
                        position_mm: args.label_xmm.zip(args.label_ymm).map(|(x, y)| [*x, *y]),
                    },
                )
                .map_err(api_error)?;
            Ok("Set View Label")
        }
        _ => Err(Error(format!(
            "method {method:?}: expected a drawing view edit"
        ))),
    }
}
fn active_drawing<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
) -> Result<&DrawingDocument, Error> {
    files
        .workspace()
        .session(owner)
        .map_err(api_error)?
        .snapshot()
        .drawing
        .as_ref()
        .ok_or_else(|| Error(format!("document {owner}: expected drawing content")))
}

fn response<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    owner: u64,
    drawing: &DrawingDocument,
) -> Result<Vec<u8>, Error> {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    if sheet.views.is_empty() {
        return crate::encode(&ListDrawingViewsResult { views: None });
    }
    let counts = if let Some(frozen) = files.deferred_drawing_views(owner).map_err(api_error)? {
        sheet
            .views
            .iter()
            .map(|view| frozen_counts(frozen, &sheet.name, &view.name))
            .collect()
    } else {
        live_counts(files, backend, replay, owner, drawing)?
    };
    let views = sheet
        .views
        .iter()
        .zip(counts)
        .map(|(view, counts)| view_info(view, counts))
        .collect::<Result<Vec<_>, _>>()?;
    crate::encode(&ListDrawingViewsResult { views: Some(views) })
}

fn live_counts<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    owner: u64,
    drawing: &DrawingDocument,
) -> Result<Vec<[usize; 2]>, Error> {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    let name = drawing.model_reference();
    let source = match files.drawing_reference(owner, name) {
        Some(id) => Some(id),
        None => files.resolve_parameter_source(name).map_err(api_error)?,
    };
    let (Some(backend), Some(source)) = (backend, source) else {
        return Ok(vec![[0; 2]; sheet.views.len()]);
    };
    if files.document_type(source).map_err(api_error)? != DOCUMENT_PART {
        return Err(Error(format!("drawing source {source}: expected a part")));
    }
    with_current_drawing_part(files, backend, replay, source, |history, guid, part| {
        sheet
            .views
            .iter()
            .map(|view| backend.counts_current(history, guid, view, part))
            .collect()
    })
    .map_err(Error)
}

fn frozen_counts(
    frozen: &oblikovati_persistence::package::SavedDrawingViews,
    sheet: &str,
    view: &str,
) -> [usize; 2] {
    let mut counts = [0; 2];
    if let Some(saved) = frozen
        .views
        .iter()
        .find(|row| row.sheet == sheet && row.view == view)
    {
        for line in &saved.lines {
            counts[usize::from(line.hidden)] += 1;
        }
    }
    counts
}
