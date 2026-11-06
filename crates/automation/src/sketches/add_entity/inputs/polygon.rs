// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{history::SketchPolygonCreation, sketch::SketchPolygonSeed};
pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    let positions = points::resolve(snapshot, request)?;
    let [center, through] = positions.as_slice() else {
        return Err(Error(format!(
            "polygon points {positions:?}: expected exactly 2 positions"
        )));
    };
    let sides = usize::try_from(request.sides).map_err(|_| {
        Error(format!(
            "polygon sides {}: expected a representable count of at least 3",
            request.sides
        ))
    })?;
    Ok(PreparedEntity::Polygon(SketchPolygonCreation {
        seed: SketchPolygonSeed {
            center: *center,
            through: *through,
            sides,
            inscribed: request.variant != "circumscribed",
        },
        construction: request.construction,
    }))
}
