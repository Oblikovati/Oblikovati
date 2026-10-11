// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn planes() -> Vec<WorkPlaneInfo> {
    vec![
        origin(0, "XY Plane", WORK_REF_XY_PLANE, SketchPlane::xy()),
        origin(1, "XZ Plane", WORK_REF_XZ_PLANE, SketchPlane::xz()),
        origin(2, "YZ Plane", WORK_REF_YZ_PLANE, SketchPlane::yz()),
    ]
}
// Origin kind names follow Go fixedPlaneDef, not the user constructor (#3744).
fn origin(index: i64, name: &str, reference: &str, frame: SketchPlane) -> WorkPlaneInfo {
    WorkPlaneInfo {
        index,
        name: name.into(),
        ref_: reference.into(),
        kind: "fixed".into(),
        origin: Some(frame.recipe().origin.to_vec()),
        normal: Some(frame.normal().to_vec()),
        is_origin: true,
        grounded: true,
        healthy: true,
        ..Default::default()
    }
}
