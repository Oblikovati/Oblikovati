// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetSketchPropertyArgs,
    editing: Option<(u64, DocumentSketchId)>,
) -> Result<Vec<u8>, Error> {
    let visible = visibility_value(&request)?;
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let rank = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .sketch_solve_options()
        .rank_tolerance;
    files
        .commands(document)
        .map_err(api_error)?
        .set_sketch_visibility_projected(sketch, visible, |reply| {
            let snapshot = reply.map_err(api_error)?;
            encode(&inspection::inspect_snapshot(
                snapshot,
                document,
                request.sketch_index,
                sketch,
                editing,
                rank,
            )?)
        })
        .map_err(api_error)
}
fn visibility_value(request: &SetSketchPropertyArgs) -> Result<bool, Error> {
    if request.property != "visible" {
        return Err(Error(format!(
            "{METHOD_SKETCH_SET_PROPERTY} property {:?}: expected supported property visible",
            request.property
        )));
    }
    if request.value.trim() != request.value {
        return Err(Error(format!(
            "{METHOD_SKETCH_SET_PROPERTY} visible value {:?}: expected a Go boolean literal without surrounding whitespace",
            request.value
        )));
    }
    crate::parameter_value::boolean_literal(&request.value)
}
