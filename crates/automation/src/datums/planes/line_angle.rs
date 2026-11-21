// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::datums::scalars::OwnedDatumScalar;
pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let [line, base] = reference_only::references(request)?;
    let line = input::axis_reference(snapshot, line)?;
    let base = plane_reference(snapshot, base)?;
    OwnedDatumScalar::Angle.validate(snapshot, &request.angle)?;
    Ok(PreparedGeometry::LineAngle {
        line,
        base,
        expression: request.angle.clone(),
    })
}
pub(super) fn slots() -> Vec<WorkPlaneRefSlot> {
    reference_only::slots(&[("Line", "axis"), ("Plane", "plane")])
}
