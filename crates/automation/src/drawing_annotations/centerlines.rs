// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{DrawingViewProjectionAccess, PartUpdateAccess, with_current_drawing_part};
use oblikovati_model::drawing::{DrawingCenterlines, drawing_view_bounds};
use std::collections::{BTreeMap, BTreeSet};

pub(super) type Counts = BTreeMap<String, usize>;

pub(super) fn add<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn PartUpdateAccess<O, W, F>>,
    owner: u64,
    mut drawing: DrawingDocument,
    args: wire::AddCenterlinesArgs,
) -> Result<Vec<u8>, Error> {
    let annotation = drawing
        .add_centerlines(&args.name, &args.view_name)
        .map_err(api_error)?;
    let count = counts(
        files,
        backend,
        replay,
        owner,
        &drawing,
        Some(&args.view_name),
    )?
    .get(&args.view_name)
    .copied()
    .ok_or_else(|| {
        Error(format!(
            "centerline view {:?}: expected projected geometry",
            args.view_name
        ))
    })?;
    let reply = crate::encode(&AnnotationResult {
        annotation: annotation_info(&annotation, None, count)?,
    })?;
    files
        .commands(owner)
        .map_err(api_error)?
        .run("Add Drawing Centerlines", |snapshot, _| {
            snapshot.drawing = Some(drawing);
            Ok::<(), Error>(())
        })
        .map_err(api_error)?;
    Ok(reply)
}

pub(super) fn counts<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn DrawingViewProjectionAccess>,
    replay: Option<&mut dyn PartUpdateAccess<O, W, F>>,
    owner: u64,
    drawing: &DrawingDocument,
    only: Option<&str>,
) -> Result<Counts, Error> {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    let names: BTreeSet<_> = match only {
        Some(view) => BTreeSet::from([view]),
        None => sheet
            .annotations
            .iter()
            .filter_map(annotation_view_name)
            .collect(),
    };
    let views: Vec<_> = sheet
        .views
        .iter()
        .filter(|view| names.contains(view.name.as_str()))
        .collect();
    if views.is_empty() {
        return Ok(Counts::new());
    }
    if let Some(frozen) = files.deferred_drawing_views(owner).map_err(api_error)? {
        return Ok(frozen
            .views
            .iter()
            .filter(|view| view.sheet == sheet.name && names.contains(view.view.as_str()))
            .filter_map(|view| count_lines(&view.lines).map(|count| (view.view.clone(), count)))
            .collect());
    }
    let reference = drawing.model_reference();
    let source = match files.drawing_reference(owner, reference) {
        Some(source) => Some(source),
        None => files
            .resolve_parameter_source(reference)
            .map_err(api_error)?,
    };
    let (Some(source), Some(backend)) = (source, backend) else {
        return Ok(Counts::new());
    };
    if files.document_type(source).map_err(api_error)? != DOCUMENT_PART {
        return Ok(Counts::new());
    }
    let result =
        with_current_drawing_part(files, backend, replay, source, |history, guid, part| {
            let mut counts = Counts::new();
            for view in views {
                let lines = match backend.saved_lines_current(history, guid, view, part) {
                    Ok(lines) => lines,
                    Err(_) if only.is_none() => continue,
                    Err(error) => return Err(error),
                };
                if let Some(count) = count_lines(&lines) {
                    counts.insert(view.name.clone(), count);
                }
            }
            Ok(counts)
        });
    match result {
        Err(_) if only.is_none() => Ok(Counts::new()),
        result => result.map_err(Error),
    }
}

fn count_lines(lines: &[oblikovati_persistence::package::SavedDrawingLine]) -> Option<usize> {
    drawing_view_bounds(lines.iter().flat_map(|line| line.points.iter().copied()))
        .map(|bounds| DrawingCenterlines::from_view_bounds(bounds).curve_count())
}
