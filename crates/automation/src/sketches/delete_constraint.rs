// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{Sketch2D, SketchConstraintId};
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: DeleteConstraintArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let constraint = selected_constraint(geometry, request.constraint_index)?;
    files
        .commands(document)
        .map_err(api_error)?
        .delete_sketch_geometric_constraint_projected(sketch, constraint, |result| {
            result.map_err(api_error)?;
            encode(&OKResult { ok: true })
        })
        .map_err(api_error)
}
fn selected_constraint(geometry: &Sketch2D, index: i64) -> Result<SketchConstraintId, Error> {
    let position = usize::try_from(index).ok();
    position
        .and_then(|position| geometry.geometric_constraints().nth(position))
        .map(|constraint| constraint.id)
        .ok_or_else(|| {
            Error(format!(
                "geometric constraint index {index}: expected an owned row in 0..{}",
                geometry.geometric_constraints().count()
            ))
        })
}
