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
    let scalars = args.scalars.as_deref().unwrap_or_default();
    if !scalars.is_empty() {
        return Err(Error(format!(
            "plane-point scalar edits {scalars:?}: expected no scalar slots"
        )));
    }
    let repicks = args.repick.as_deref().unwrap_or_default();
    validate_slots(repicks)?;
    for edit in repicks {
        if edit.slot == 0 {
            source.base = plane_reference(snapshot, &edit.ref_)?;
        } else {
            source.point = input::point_reference(snapshot, &edit.ref_)?;
        }
    }
    Ok(WorkGeometryDefinition::ParallelThroughPoint(source))
}
fn validate_slots(edits: &[SlotRepick]) -> Result<(), Error> {
    if edits.iter().any(|edit| !matches!(edit.slot, 0 | 1))
        || edits.len() > 2
        || (edits.len() == 2 && edits[0].slot == edits[1].slot)
    {
        return Err(Error(format!(
            "plane-point reference edits {edits:?}: expected unique base-plane slot 0 and through-point slot 1"
        )));
    }
    Ok(())
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
