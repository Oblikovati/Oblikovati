// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{feature::work_plane::WorkPlaneResolver, sketch::SketchPlane};
pub(in crate::datums) mod axis_point;
mod display;
mod inputs;
pub(in crate::datums) mod line_angle;
mod metadata;
mod origins;
pub(in crate::datums) mod parallel;
pub(in crate::datums) mod reference_only;
pub(in crate::datums) mod three_points;
pub(in crate::datums) mod two_axes;
pub(super) use inputs::plane_reference;
pub(super) use inputs::prepare;
type PlaneEntry = AuthoredFeature<WorkGeometryDefinition>;
pub(super) fn current_info(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
    index: i64,
) -> Result<WorkPlaneInfo, Error> {
    let evaluation = rows::evaluate(snapshot)?;
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    plane_info(snapshot, feature, index, &evaluation)
}

pub(super) fn list(
    snapshot: &DocumentSnapshot,
    request: ListWorkPlanesArgs,
) -> Result<Vec<u8>, Error> {
    rows::require_local_occurrence(request.occurrence.as_deref())?;
    let planes = rows::current_rows(
        snapshot,
        DatumCollection::Planes,
        request.include_construction,
        origins::planes()?,
        plane_info,
    )?;
    encode(&ListWorkPlanesResult {
        planes: Some(planes),
    })
}
pub(super) fn plane_info(
    snapshot: &DocumentSnapshot,
    feature: &PlaneEntry,
    index: i64,
    evaluation: &WorkGeometryEvaluation,
) -> Result<WorkPlaneInfo, Error> {
    let mut info = identity(snapshot, feature, index)?;
    let geometry = evaluation.plane(feature.id());
    info.healthy = geometry.is_ok();
    info.reason = health::reason(evaluation, feature.id(), geometry.as_ref().err())?;
    display::append(
        &mut info,
        snapshot,
        feature,
        geometry.as_ref().ok().copied(),
    )?;
    info.origin = geometry
        .as_ref()
        .ok()
        .map(|frame| frame.recipe().origin.to_vec());
    info.normal = geometry.ok().map(|frame| frame.normal().to_vec());
    metadata::append_inputs(&mut info, snapshot, *feature.definition())?;
    Ok(info)
}
fn identity(
    snapshot: &DocumentSnapshot,
    feature: &PlaneEntry,
    index: i64,
) -> Result<WorkPlaneInfo, Error> {
    Ok(WorkPlaneInfo {
        index,
        ref_: reference::encode_reference(snapshot, feature.id())?,
        name: feature.name().into(),
        kind: metadata::constructor(*feature.definition())?,
        visible: feature.is_visible(),
        construction: feature.is_construction_datum(),
        auto_resize: feature.plane_display_settings().auto_resize(),
        grounded: feature.plane_display_settings().grounded(),
        ..Default::default()
    })
}
pub(super) fn creation_reply(
    snapshot: &DocumentSnapshot,
    feature: &PlaneEntry,
    index: i64,
    evaluation: &WorkGeometryEvaluation,
) -> Result<Vec<u8>, Error> {
    let info = plane_info(snapshot, feature, index, evaluation)?;
    encode(&CreateWorkPlaneResult {
        index: info.index,
        ref_: info.ref_,
        name: info.name,
        healthy: info.healthy,
    })
}

#[cfg(test)]
mod tests;
