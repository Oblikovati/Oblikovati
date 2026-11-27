// SPDX-License-Identifier: GPL-2.0-only
//! Read and write entity presentation overrides in the active sketch.
use super::*;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};

fn owner<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<(u64, DocumentSketchId), Error> {
    let document = super::owner::active_sketch_host(files)?;
    editing.filter(|(id, _)| *id == document).ok_or_else(|| Error(format!("sketch entity format {editing:?}: expected an active sketch edit in document {document}")))
}
fn view(geometry: &Sketch2D, entity: u64) -> Result<SketchEntityFormatView, Error> {
    let format = geometry
        .entity_format(SketchEntityId(entity))
        .map_err(api_error)?;
    Ok(SketchEntityFormatView {
        entity_id: entity,
        has_format: format.is_some(),
        format: format.cloned().unwrap_or_default(),
    })
}
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchEntityFormatArgs,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = owner(files, editing)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let placed = session.snapshot().sketches.get(sketch).map_err(api_error)?;
    encode(&view(placed.geometry(), request.entity_id)?)
}
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetSketchEntityFormatArgs,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = owner(files, editing)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set sketch entity format",
            |snapshot, _| {
                snapshot
                    .sketches
                    .set_entity_format(
                        sketch,
                        SketchEntityId(request.entity_id),
                        request.format.clone(),
                    )
                    .map_err(api_error)
            },
            |snapshot, _| {
                view(
                    snapshot.sketches.get(sketch).map_err(api_error)?.geometry(),
                    request.entity_id,
                )
                .and_then(|reply| encode(&reply))
            },
        )
        .map_err(api_error)
}
