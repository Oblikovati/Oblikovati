// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPointReference;

pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let [first, second, third] = reference_only::references(request)?;
    let points = [
        input::point_reference(snapshot, first)?,
        input::point_reference(snapshot, second)?,
        input::point_reference(snapshot, third)?,
    ];
    Ok(PreparedGeometry::Definition(
        WorkGeometryDefinition::ThroughThreePoints(points),
    ))
}
pub(in crate::datums) fn revise(
    snapshot: &DocumentSnapshot,
    mut points: [WorkPointReference; 3],
    args: &RedefineWorkPlaneArgs,
) -> Result<WorkGeometryDefinition, Error> {
    reference_only::validate(args, "three-point", 3)?;
    for edit in args.repick.as_deref().unwrap_or_default() {
        points[edit.slot as usize] = input::point_reference(snapshot, &edit.ref_)?;
    }
    Ok(WorkGeometryDefinition::ThroughThreePoints(points))
}
pub(super) fn slots() -> Vec<WorkPlaneRefSlot> {
    reference_only::slots(&[
        ("Point 1", "point"),
        ("Point 2", "point"),
        ("Point 3", "point"),
    ])
}
