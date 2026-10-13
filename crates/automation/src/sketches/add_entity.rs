// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchPrimitiveProjectionInput;
mod chain_reply;
mod inputs;
mod line_reply;
use inputs::PreparedEntity;
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
    let input = inputs::prepare(snapshot, &request)?;
    create_prepared(files, document, sketch, input)
}
fn create_prepared<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    input: PreparedEntity,
) -> Result<Vec<u8>, Error> {
    let mut commands = files.commands(document).map_err(api_error)?;
    match input {
        PreparedEntity::Primitive(input, kind) => commands
            .create_sketch_primitive_projected(sketch, &input, |result| {
                creation_reply(result, kind)
            })
            .map_err(api_error),
        PreparedEntity::Chain { input, kind, label } => commands
            .run(label, |snapshot, ids| {
                chain_reply::encode_created(
                    snapshot.create_sketch_polyline(sketch, &input, ids),
                    kind,
                )
            })
            .map_err(api_error),
        PreparedEntity::Polygon(input) => commands
            .run("Create sketch polygon", |snapshot, ids| {
                let report = snapshot
                    .create_sketch_polygon(sketch, &input, ids)
                    .map_err(api_error)?;
                chain_reply::encode_polygon(report)
            })
            .map_err(api_error),
        PreparedEntity::InferredLine(input) => commands
            .create_inferred_sketch_line_projected(sketch, &input, line_reply::encode_created)
            .map_err(api_error),
    }
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
