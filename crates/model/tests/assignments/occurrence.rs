// SPDX-License-Identifier: GPL-2.0-only
use super::fixture::*;
use oblikovati_api::{
    contract::{Appearance, Material},
    types::{ASSET_DOCUMENT, ASSET_PROJECT},
};
use oblikovati_model::assets::AssetAssignments;
#[test]
fn equal_entity_text_in_different_occurrences_does_not_share_overrides() {
    let fixture = AssignmentFixture::new();
    let mut assignments = AssetAssignments::default();
    let left = reference("body", "left");
    let right = reference("body", "right");
    let left_face = reference("face", "left");
    let right_face = reference("face", "right");
    assignments.set_part_material("steel").unwrap();
    assignments
        .set_body_material(&left, "aluminum-6061")
        .unwrap();
    assignments.set_body_appearance(&left, "oak").unwrap();
    assignments
        .set_face_appearance(&left_face, "abs-black")
        .unwrap();
    assert_eq!(assignments.effective_material_id(&right), Some("steel"));
    assert_eq!(
        fixture.visual_id(&assignments, &left, Some(&left_face)),
        "abs-black"
    );
    assert_eq!(
        fixture.visual_id(&assignments, &right, Some(&right_face)),
        "steel"
    );
    let failure = assignments
        .resolve_appearance(
            &fixture.appearance_scopes(),
            &fixture.material_scopes(),
            &right,
            Some(&left_face),
        )
        .unwrap_err();
    assert!(failure.contains("same occurrence"));
    assert_eq!(left.occurrence_path(), ["left"]);
}
#[test]
fn document_and_project_definitions_keep_their_resolution_priority() {
    let mut fixture = AssignmentFixture::new();
    let mut assignments = AssetAssignments::default();
    let body = reference("body", "");
    let project = fixture.appearances[2]
        .get("steel")
        .unwrap()
        .duplicate("steel", "Project steel", ASSET_PROJECT)
        .unwrap();
    fixture.appearances[1].insert(project);
    assignments.set_part_material("steel").unwrap();
    assert_eq!(
        resolved_visual_name(&fixture, &assignments, &body),
        "Project steel"
    );
    let document = fixture.appearances[2]
        .get("steel")
        .unwrap()
        .duplicate("steel", "Document steel", ASSET_DOCUMENT)
        .unwrap();
    fixture.appearances[0].insert(document);
    assert_eq!(
        resolved_visual_name(&fixture, &assignments, &body),
        "Document steel"
    );
    assert_eq!(
        assignments
            .resolve_material(&fixture.material_scopes(), &body)
            .unwrap()
            .unwrap()
            .id(),
        "steel"
    );
}
fn resolved_visual_name(
    fixture: &AssignmentFixture,
    assignments: &AssetAssignments,
    body: &oblikovati_model::assets::AssignmentReference,
) -> String {
    assignments
        .resolve_appearance(
            &fixture.appearance_scopes(),
            &fixture.material_scopes(),
            body,
            None,
        )
        .unwrap()
        .display_name()
}
