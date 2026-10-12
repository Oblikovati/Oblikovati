// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
use oblikovati_model::assets::{AssetAssignments, AssignmentReference};
#[test]
fn selected_missing_visual_intent_is_refused_instead_of_using_lower_priority_assets() {
    let fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let face = reference("face", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("steel").unwrap();
    assignments.set_part_appearance("missing-part").unwrap();
    assert!(visual_failure(&fixture, &assignments, &body, Some(&face)).contains("missing-part"));
    assignments
        .set_body_appearance(&body, "missing-body")
        .unwrap();
    assert!(visual_failure(&fixture, &assignments, &body, Some(&face)).contains("missing-body"));
    assignments
        .set_face_appearance(&face, "missing-face")
        .unwrap();
    assert!(visual_failure(&fixture, &assignments, &body, Some(&face)).contains("missing-face"));
}
fn visual_failure(
    fixture: &AssignmentFixture,
    assignments: &AssetAssignments,
    body: &AssignmentReference,
    face: Option<&AssignmentReference>,
) -> String {
    assignments
        .resolve_appearance(
            &fixture.appearance_scopes(),
            &fixture.material_scopes(),
            body,
            face,
        )
        .unwrap_err()
}
#[test]
fn missing_physical_or_material_visual_identity_is_a_named_failure() {
    let mut fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("steel").unwrap();
    assignments
        .set_body_material(&body, "missing-material")
        .unwrap();
    assert!(
        assignments
            .resolve_material(&fixture.material_scopes(), &body)
            .unwrap_err()
            .contains("missing-material")
    );
    assert!(visual_failure(&fixture, &assignments, &body, None).contains("missing-material"));
    assignments.set_body_material(&body, "").unwrap();
    let mut record = fixture.materials[2].get("steel").unwrap().info();
    record.appearance_id = "missing-visual".into();
    fixture.materials[0]
        .insert(oblikovati_model::assets::MaterialAsset::from_info(record).unwrap());
    assert!(visual_failure(&fixture, &assignments, &body, None).contains("missing-visual"));
}
#[test]
fn invalid_assignment_replacement_is_atomic_and_references_have_valid_shapes() {
    let body = reference("body", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("steel").unwrap();
    assignments.set_part_appearance("oak").unwrap();
    assignments.set_body_material(&body, "steel").unwrap();
    assignments.set_body_appearance(&body, "oak").unwrap();
    assignments.set_face_appearance(&body, "steel").unwrap();
    let before = assignments.clone();
    assert!(assignments.set_part_material("bad\0id").is_err());
    assert!(assignments.set_part_appearance("bad\0id").is_err());
    assert!(assignments.set_body_material(&body, "bad\0id").is_err());
    assert!(assignments.set_body_appearance(&body, "bad\0id").is_err());
    assert!(assignments.set_face_appearance(&body, "bad\0id").is_err());
    assert_eq!(assignments, before);
    for key in ["", "bad\0key"] {
        assert!(AssignmentReference::new(key, vec![]).is_err());
    }
    for path in ["", "bad\0path"] {
        assert!(AssignmentReference::new("body", vec![path.into()]).is_err());
    }
    assert_eq!(body.reference_key(), "body");
    assert!(body.occurrence_path().is_empty());
}

#[test]
fn neutral_default_is_required_and_a_visual_override_does_not_repair_missing_physics() {
    let mut fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("missing-physics").unwrap();
    assignments.set_part_appearance("oak").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "oak");
    assert!(
        assignments
            .resolve_material(&fixture.material_scopes(), &body)
            .is_err()
    );
    fixture.appearances[2] = Default::default();
    assert!(
        visual_failure(&fixture, &AssetAssignments::default(), &body, None).contains("default")
    );
}
