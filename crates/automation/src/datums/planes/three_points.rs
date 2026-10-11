// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPointReference;

pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [first, second, third] = refs else {
        return Err(Error(format!(
            "three-point references {refs:?}: expected exactly three point references"
        )));
    };
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
    ["Point 1", "Point 2", "Point 3"]
        .into_iter()
        .enumerate()
        .map(|(index, label)| WorkPlaneRefSlot {
            index: index as i64,
            label: label.into(),
            kind: "point".into(),
        })
        .collect()
}
