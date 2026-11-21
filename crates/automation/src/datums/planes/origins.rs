// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn planes() -> Result<Vec<WorkPlaneInfo>, Error> {
    [
        origin(0, "XY Plane", WORK_REF_XY_PLANE, SketchPlane::xy()),
        origin(1, "XZ Plane", WORK_REF_XZ_PLANE, SketchPlane::xz()),
        origin(2, "YZ Plane", WORK_REF_YZ_PLANE, SketchPlane::yz()),
    ]
    .into_iter()
    .collect()
}
// Origin kind names follow Go fixedPlaneDef, not the user constructor (#3744).
fn origin(
    index: i64,
    name: &str,
    reference: &str,
    frame: SketchPlane,
) -> Result<WorkPlaneInfo, Error> {
    Ok(WorkPlaneInfo {
        index,
        name: name.into(),
        ref_: reference.into(),
        kind: "fixed".into(),
        origin: Some(frame.recipe().origin.to_vec()),
        normal: Some(frame.normal().to_vec()),
        is_origin: true,
        grounded: true,
        healthy: true,
        size: origin_size(frame)?,
        ..Default::default()
    })
}
fn origin_size(frame: SketchPlane) -> Result<Option<Vec<Option<Vec<f64>>>>, Error> {
    oblikovati_model::feature::work_geometry::WorkPlaneDisplaySettings::default()
        .current_corners(Some(frame), &Default::default())
        .map(|corners| corners.map(super::display::wire_corners))
        .map_err(api_error)
}
