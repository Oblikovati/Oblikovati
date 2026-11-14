// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::SketchPlacement, feature::work_plane::WorkPlaneOrigin};
pub(crate) fn sketch_plane_at_index(
    snapshot: &DocumentSnapshot,
    index: i64,
) -> Result<(SketchPlacement, String), Error> {
    let origins = [
        WorkPlaneOrigin::XY,
        WorkPlaneOrigin::XZ,
        WorkPlaneOrigin::YZ,
    ];
    let position = usize::try_from(index).map_err(|_| invalid_plane_index(snapshot, index))?;
    if let Some(origin) = origins.get(position) {
        return Ok(origin_placement(*origin));
    }
    let feature = rows::entries(snapshot, DatumCollection::Planes)
        .nth(position - origins.len())
        .ok_or_else(|| invalid_plane_index(snapshot, index))?;
    Ok((
        SketchPlacement::WorkPlane(feature.id()),
        feature.name().into(),
    ))
}
fn invalid_plane_index(snapshot: &DocumentSnapshot, index: i64) -> Error {
    let count = DatumCollection::Planes.origin_count()
        + rows::entries(snapshot, DatumCollection::Planes).count();
    Error(format!(
        "sketch.create work plane index {index}: expected an ordered index in 0..{count}"
    ))
}

fn origin_placement(origin: WorkPlaneOrigin) -> (SketchPlacement, String) {
    (
        SketchPlacement::Fixed(origin.plane()),
        format!("{origin:?} Plane"),
    )
}
