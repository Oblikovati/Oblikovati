// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::{
    work_geometry::DatumLength,
    work_plane::{WorkPlaneOrigin, WorkPlaneReference},
};
fn offset(distance: DatumLength) -> WorkGeometryDefinition {
    WorkGeometryDefinition::Offset {
        base: WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        distance,
    }
}
#[test]
fn invalid_offset_display_and_wrong_definition_kinds_are_named_declines() {
    let snapshot = crate::datums::tests::snapshot();
    let mut info = WorkPlaneInfo::default();
    let error = metadata::append_inputs(
        &mut info,
        &snapshot,
        offset(DatumLength::Constant(f64::MAX)),
    )
    .unwrap_err();
    assert!(error.to_string().contains("expected a finite scalar"));
    let wrong = WorkGeometryDefinition::Point([DatumLength::Constant(0.0); 3]);
    assert!(
        metadata::constructor(wrong)
            .unwrap_err()
            .to_string()
            .contains("supported plane definition")
    );
}
#[test]
fn unresolved_scalar_retains_slot_metadata_without_inventing_a_value() {
    let snapshot = crate::datums::tests::snapshot();
    let mut info = WorkPlaneInfo::default();
    metadata::append_inputs(&mut info, &snapshot, offset(DatumLength::Parameter(99))).unwrap();
    assert!(info.scalars.is_none());
    assert_eq!(info.slots.unwrap()[0].label, "Base plane");
}
