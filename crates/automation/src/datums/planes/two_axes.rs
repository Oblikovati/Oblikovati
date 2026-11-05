// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_axis::WorkAxisReference;

pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let [first, second] = reference_only::references(request)?;
    let axes = [
        input::axis_reference(snapshot, first)?,
        input::axis_reference(snapshot, second)?,
    ];
    Ok(PreparedGeometry::Definition(
        WorkGeometryDefinition::ThroughTwoAxes(axes),
    ))
}
pub(in crate::datums) fn revise(
    snapshot: &DocumentSnapshot,
    mut axes: [WorkAxisReference; 2],
    args: &RedefineWorkPlaneArgs,
) -> Result<WorkGeometryDefinition, Error> {
    reference_only::validate(args, WORK_PLANE_TWO_LINES.0.as_ref(), 2)?;
    for edit in args.repick.as_deref().unwrap_or_default() {
        axes[edit.slot as usize] = input::axis_reference(snapshot, &edit.ref_)?;
    }
    Ok(WorkGeometryDefinition::ThroughTwoAxes(axes))
}
pub(super) fn slots() -> Vec<WorkPlaneRefSlot> {
    reference_only::slots(&[("Line 1", "axis"), ("Line 2", "axis")])
}
