// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn axes() -> Vec<WorkAxisInfo> {
    vec![
        axis(0, "X Axis", WORK_REF_X_AXIS, [1.0, 0.0, 0.0]),
        axis(1, "Y Axis", WORK_REF_Y_AXIS, [0.0, 1.0, 0.0]),
        axis(2, "Z Axis", WORK_REF_Z_AXIS, [0.0, 0.0, 1.0]),
    ]
}
fn axis(index: i64, name: &str, reference: &str, direction: [f64; 3]) -> WorkAxisInfo {
    WorkAxisInfo {
        index,
        name: name.into(),
        ref_: reference.into(),
        kind: WORK_AXIS_LINE.0.into_owned(),
        origin: Some(vec![0.0; 3]),
        direction: Some(direction.to_vec()),
        is_origin: true,
        healthy: true,
        ..Default::default()
    }
}
pub(super) fn point() -> WorkPointInfo {
    // Go reports its immutable center as fixed; it is not a user constructor (#3741).
    WorkPointInfo {
        name: "Center Point".into(),
        ref_: WORK_REF_CENTER.into(),
        kind: "fixed".into(),
        position: Some(vec![0.0; 3]),
        is_origin: true,
        healthy: true,
        ..Default::default()
    }
}
