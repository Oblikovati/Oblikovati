// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: DeleteSketchDimensionArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let constraint = super::dimensions::selected_constraint(
        snapshot.sketches.get(sketch).map_err(api_error)?.geometry(),
        request.dimension_index,
    )?;
    files
        .commands(document)
        .map_err(api_error)?
        .delete_sketch_dimension_projected(sketch, constraint, encode_deleted)
        .map_err(api_error)
}
fn encode_deleted(
    reply: oblikovati_session::SketchDimensionDeleteProjectionInput,
) -> Result<Vec<u8>, Error> {
    let dof = super::indices::constraint_wire_index(reply.map_err(api_error)?, "dimension DOF")?;
    encode(&DeleteSketchDimensionResult { dof })
}
