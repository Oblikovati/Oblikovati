// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: MoveSketchDimensionArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let (dimension, position) = prepare_position(snapshot, sketch, &request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .move_sketch_dimension_projected(sketch, dimension, position, |reply| {
            reply.map_err(api_error)?;
            encode(&OKResult { ok: true })
        })
        .map_err(api_error)
}

fn prepare_position(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    request: &MoveSketchDimensionArgs,
) -> Result<(oblikovati_model::sketch::SketchConstraintId, [f64; 2]), Error> {
    let dimension = super::dimensions::selected_constraint(
        snapshot.sketches.get(sketch).map_err(api_error)?.geometry(),
        request.dimension_index,
    )?;
    let position = super::coordinates::working_pair(
        request.text_point.as_deref().unwrap_or_default(),
        snapshot.parameters.units().working_scale(),
        "dimension text point",
    )?;
    Ok((dimension, position))
}
