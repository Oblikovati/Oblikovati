// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchPrimitiveProjectionInput;
mod associative;
mod chain_reply;
mod corner;
mod inputs;
mod line_reply;
mod slot;
mod three_point;
use inputs::PreparedEntity;
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddSketchEntityArgs,
    construction: Option<crate::file_host::SketchConstructionAccess<'_>>,
    corner_construction: Option<(
        &dyn oblikovati_model::sketch::SketchCornerConstruction,
        oblikovati_model::sketch::SketchConstructionPolicy,
    )>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    if matches!(request.kind.as_str(), "fillet" | "chamfer") {
        return corner::create(files, document, sketch, request, corner_construction);
    }
    let input = inputs::prepare(snapshot, &request)?;
    let bindings = associative::bindings(snapshot, &request)?;
    let slot_width = if request.kind == "slot" && associative::named(snapshot, &request.width)? {
        Some(request.width)
    } else {
        None
    };
    create_prepared(
        files,
        document,
        sketch,
        input,
        construction,
        bindings,
        slot_width,
    )
}
fn create_prepared<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    input: PreparedEntity,
    construction: Option<crate::file_host::SketchConstructionAccess<'_>>,
    bindings: Vec<associative::RadiusBinding>,
    slot_width: Option<String>,
) -> Result<Vec<u8>, Error> {
    let mut commands = files.commands(document).map_err(api_error)?;
    match input {
        PreparedEntity::Slot(input) => {
            slot::create(files, document, sketch, input, construction, slot_width)
        }
        PreparedEntity::ThreePoint(input) => {
            three_point::create(files, document, sketch, input, construction)
        }
        PreparedEntity::Primitive(input, kind) if !bindings.is_empty() => {
            associative::create(files, document, sketch, input, kind, bindings)
        }
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
        point_i_ds: (!report.points.is_empty())
            .then(|| report.points.into_iter().map(|id| id.0).collect()),
        ..Default::default()
    })
}
