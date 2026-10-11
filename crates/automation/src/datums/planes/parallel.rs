// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::work_geometry::WorkPlaneThroughPoint;

pub(in crate::datums) fn create(
    request: &CreateWorkPlaneArgs,
    snapshot: &DocumentSnapshot,
) -> Result<PreparedGeometry, Error> {
    let refs = request.refs.as_deref().unwrap_or_default();
    let [base, point] = refs else {
        return Err(Error(format!(
            "plane-point references {refs:?}: expected exactly two references: base plane, through point"
        )));
    };
    let source = WorkPlaneThroughPoint {
        base: plane_reference(snapshot, base)?,
        point: input::point_reference(snapshot, point)?,
    };
    Ok(PreparedGeometry::Definition(
        WorkGeometryDefinition::ParallelThroughPoint(source),
    ))
}
pub(in crate::datums) fn revise(
    snapshot: &DocumentSnapshot,
    mut source: WorkPlaneThroughPoint,
    args: &RedefineWorkPlaneArgs,
) -> Result<WorkGeometryDefinition, Error> {
    reference_only::validate(args, "plane-point", 2)?;
    let repicks = args.repick.as_deref().unwrap_or_default();
    for edit in repicks {
        if edit.slot == 0 {
            source.base = plane_reference(snapshot, &edit.ref_)?;
        } else {
            source.point = input::point_reference(snapshot, &edit.ref_)?;
        }
    }
    Ok(WorkGeometryDefinition::ParallelThroughPoint(source))
}
pub(super) fn slots() -> Vec<WorkPlaneRefSlot> {
    vec![
        WorkPlaneRefSlot {
            index: 0,
            label: "Base plane".into(),
            kind: "plane".into(),
        },
        WorkPlaneRefSlot {
            index: 1,
            label: "Through point".into(),
            kind: "point".into(),
        },
    ]
}
