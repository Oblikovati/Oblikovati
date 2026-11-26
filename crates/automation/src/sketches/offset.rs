// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::{PrismResolutionPolicy, offset_sketch_profile};
use oblikovati_model::{
    history::SketchPolylineCreation,
    quantity::Dimension,
    sketch::{SketchEntityId, SketchPrimitiveCreation},
};

mod entities;

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: OffsetSketchArgs,
    policy: Option<PrismResolutionPolicy>,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let distance = snapshot
        .parameters
        .evaluate_scalar_expression(&request.distance, Dimension::Length)
        .map_err(api_error)?
        .value;
    let offset = prepare(snapshot, sketch, &request, distance, policy)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run("Offset sketch", |snapshot, ids| {
            let (created, kind) = match offset {
                PreparedOffset::Primitive(input, kind) => (
                    vec![
                        snapshot
                            .create_sketch_primitive(sketch, &input, ids)
                            .map_err(api_error)?
                            .entity,
                    ],
                    kind,
                ),
                PreparedOffset::Chains(chains) => {
                    let mut created = Vec::new();
                    for chain in chains {
                        created.extend(
                            snapshot
                                .create_sketch_polyline(sketch, &chain, ids)
                                .map_err(api_error)?
                                .lines,
                        );
                    }
                    (created, "line")
                }
            };
            let entity = created
                .first()
                .ok_or_else(|| Error("sketch offset: expected an authored curve".into()))?;
            encode(&OffsetSketchResult {
                entity_id: entity.0,
                kind: kind.into(),
                created: Some(created.into_iter().map(|id| id.0).collect()),
            })
        })
        .map_err(api_error)
}

enum PreparedOffset {
    Primitive(SketchPrimitiveCreation, &'static str),
    Chains(Vec<SketchPolylineCreation>),
}

fn prepare(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    request: &OffsetSketchArgs,
    distance: f64,
    policy: Option<PrismResolutionPolicy>,
) -> Result<PreparedOffset, Error> {
    if let Some(profile) = &request.profile_index {
        let profile = usize::try_from(**profile).map_err(api_error)?;
        let segments = arc_segments(request.arc_segments)?;
        let policy = policy.ok_or_else(|| {
            Error("sketch profile offset requires a certified extrusion policy".into())
        })?;
        let loops = offset_sketch_profile(snapshot, sketch, policy, profile, distance, segments)
            .map_err(Error)?;
        return Ok(PreparedOffset::Chains(
            loops
                .into_iter()
                .map(|points| SketchPolylineCreation {
                    points,
                    closed: true,
                    construction: false,
                })
                .collect(),
        ));
    }
    let index = snapshot
        .sketches
        .get(sketch)
        .map_err(api_error)?
        .geometry()
        .read_index();
    if let Some(ids) = request.entities.as_deref().filter(|ids| !ids.is_empty()) {
        return Ok(PreparedOffset::Chains(vec![entities::chain(
            &index, ids, distance,
        )?]));
    }
    let (primitive, kind) = entities::single(&index, SketchEntityId(request.entity), distance)?;
    Ok(PreparedOffset::Primitive(primitive, kind))
}

fn arc_segments(requested: i64) -> Result<u32, Error> {
    if requested <= 0 {
        return Ok(8);
    }
    if requested > 4096 {
        return Err(Error(format!(
            "sketch offset arcSegments {requested}: expected at most 4096 spans per curve"
        )));
    }
    u32::try_from(requested).map_err(api_error)
}
