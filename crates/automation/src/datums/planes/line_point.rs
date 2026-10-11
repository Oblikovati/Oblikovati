// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPlaneLinePoint;

pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [line, point] = refs else {
        return Err(Error(format!(
            "line-point references {refs:?}: expected exactly two references: axis, through point"
        )));
    };
    let source = WorkPlaneLinePoint {
        line: input::axis_reference(snapshot, line)?,
        point: input::point_reference(snapshot, point)?,
    };
    Ok(PreparedGeometry::Definition(
        WorkGeometryDefinition::LineThroughPoint(source),
    ))
}
pub(in crate::datums) fn revise(
    snapshot: &DocumentSnapshot,
    mut source: WorkPlaneLinePoint,
    args: &RedefineWorkPlaneArgs,
) -> Result<WorkGeometryDefinition, Error> {
    reference_only::validate(args, "line-point", 2)?;
    for edit in args.repick.as_deref().unwrap_or_default() {
        if edit.slot == 0 {
            source.line = input::axis_reference(snapshot, &edit.ref_)?;
        } else {
            source.point = input::point_reference(snapshot, &edit.ref_)?;
        }
    }
    Ok(WorkGeometryDefinition::LineThroughPoint(source))
}
pub(super) fn slots() -> Vec<WorkPlaneRefSlot> {
    reference_only::slots(&[("Line", "axis"), ("Through point", "point")])
}
