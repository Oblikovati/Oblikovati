// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
use oblikovati_api::contract::Material;
use oblikovati_model::assets::AssetAssignments;
#[test]
fn go_assignment_precedence_uses_explicit_visual_intent_before_material_defaults() {
    let fixture = AssignmentFixture::new();
    let (body, face) = (reference("bodykey", ""), reference("facekey", ""));
    let mut assignments = AssetAssignments::default();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "default");
    assignments.set_part_material("steel").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "steel");
    assignments
        .set_body_material(&body, "aluminum-6061")
        .unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "aluminum");
    assignments.set_part_appearance("oak").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "oak");
    assignments.set_body_appearance(&body, "steel").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "steel");
    assignments.set_face_appearance(&face, "abs-black").unwrap();
    assert_eq!(
        fixture.visual_id(&assignments, &body, Some(&face)),
        "abs-black"
    );
    assert_eq!(
        fixture.visual_id(&assignments, &body, Some(&reference("other-face", ""))),
        "steel"
    );
}
#[test]
fn go_1103_clear_part_override_restores_the_effective_material_appearance() {
    let fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("steel").unwrap();
    assignments.set_part_appearance("oak").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "oak");
    assignments.set_part_appearance("").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "steel");
}
#[test]
fn physical_intent_stays_independent_of_every_visual_override() {
    let fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let mut assignments = AssetAssignments::default();
    assert_eq!(assignments.effective_material_id(&body), None);
    assert!(
        assignments
            .resolve_material(&fixture.material_scopes(), &body)
            .unwrap()
            .is_none()
    );
    assignments.set_part_material("steel").unwrap();
    assignments.set_part_appearance("oak").unwrap();
    assignments
        .set_body_material(&body, "aluminum-6061")
        .unwrap();
    assert_eq!(
        assignments.effective_material_id(&body),
        Some("aluminum-6061")
    );
    assert_eq!(
        assignments
            .resolve_material(&fixture.material_scopes(), &body)
            .unwrap()
            .unwrap()
            .id(),
        "aluminum-6061"
    );
    assignments.set_body_material(&body, "").unwrap();
    assert_eq!(assignments.effective_material_id(&body), Some("steel"));
    assignments.set_part_material("").unwrap();
    assert_eq!(assignments.effective_material_id(&body), None);
    assert_eq!(fixture.visual_id(&assignments, &body, None), "oak");
}
#[test]
fn clearing_entity_overrides_exposes_the_next_declared_intent() {
    let fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let face = reference("face", "");
    let mut assignments = AssetAssignments::default();
    assignments.set_part_appearance("oak").unwrap();
    assignments.set_body_appearance(&body, "steel").unwrap();
    assignments.set_face_appearance(&face, "abs-black").unwrap();
    assignments.set_face_appearance(&face, "").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, Some(&face)), "steel");
    assignments.set_body_appearance(&body, "").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, Some(&face)), "oak");
}
#[test]
fn physically_assigned_material_without_visual_intent_uses_the_neutral_default() {
    let mut fixture = AssignmentFixture::new();
    let body = reference("body", "");
    let mut material = fixture.materials[2].get("steel").unwrap().info();
    material.appearance_id.clear();
    fixture.materials[0]
        .insert(oblikovati_model::assets::MaterialAsset::from_info(material).unwrap());
    let mut assignments = AssetAssignments::default();
    assignments.set_part_material("steel").unwrap();
    assert_eq!(fixture.visual_id(&assignments, &body, None), "default");
}
