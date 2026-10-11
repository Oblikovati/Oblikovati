// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchPrimitiveProjectionInput;
mod inputs;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddSketchEntityArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let (input, kind) = inputs::prepare(snapshot, &request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .create_sketch_primitive_projected(sketch, &input, |result| creation_reply(result, kind))
        .map_err(api_error)
}
fn creation_reply(
    result: SketchPrimitiveProjectionInput,
    kind: &'static str,
) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    encode(&AddSketchEntityResult {
        entity_id: report.entity.0,
        kind: kind.into(),
        point_i_ds: Some(report.points.into_iter().map(|id| id.0).collect()),
        ..Default::default()
    })
}
