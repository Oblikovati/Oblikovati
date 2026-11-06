// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::SketchPolylineCreation, sketch::SketchRectangleSeed};
pub(super) fn polyline(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    Ok(PreparedEntity::Chain {
        input: SketchPolylineCreation {
            points: points::resolve(snapshot, request)?,
            closed: request.closed,
            construction: request.construction,
        },
        kind: "polyline",
        label: "Create sketch polyline",
    })
}
pub(super) fn rectangle(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    let points = points::resolve(snapshot, request)?;
    let seed = rectangle_seed(&request.variant, &points)?;
    let vertices = seed.vertices().map_err(api_error)?;
    Ok(PreparedEntity::Chain {
        input: SketchPolylineCreation {
            points: vertices.to_vec(),
            closed: true,
            construction: request.construction,
        },
        kind: "rectangle",
        label: "Create sketch rectangle",
    })
}
fn rectangle_seed(variant: &str, points: &[[f64; 2]]) -> Result<SketchRectangleSeed, Error> {
    match (variant, points) {
        ("threePoint", [first, second, third]) => {
            Ok(SketchRectangleSeed::ThreePoints([*first, *second, *third]))
        }
        ("center", [center, corner]) => Ok(SketchRectangleSeed::Center([*center, *corner])),
        (name, [first, opposite]) if name != "threePoint" => {
            Ok(SketchRectangleSeed::Corners([*first, *opposite]))
        }
        _ => Err(Error(format!(
            "rectangle variant {variant:?}, points {points:?}: expected exactly {} positions",
            if variant == "threePoint" { 3 } else { 2 }
        ))),
    }
}
