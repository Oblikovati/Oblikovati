// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    assets::{AssignmentReference, DocumentAssets},
    identity::SemanticReferenceName,
};
fn entity_key(owner: &TestOwner, document: u64, feature: u64, role: &str) -> String {
    let guid = owner
        .documents()
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    let kind = if role == "body" { 4 } else { 1 };
    SemanticReferenceName::new(guid, feature, kind, "sketch-prism", role, &[1])
        .unwrap()
        .encode()
        .unwrap()
        .iter()
        .map(|byte| format!("{byte:02x}"))
        .collect()
}
fn assigned_assets(owner: &TestOwner, document: u64) -> DocumentAssets {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .assets
        .clone()
}
fn assign_look(
    owner: &mut TestOwner,
    scope: &str,
    key: &str,
    id: &str,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: scope.into(),
            key: key.into(),
            appearance_id: id.into(),
        })
}
#[test]
fn exact_body_and_face_commands_are_atomic_undoable_and_canonical() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let body = entity_key(&owner, document, 1, "body");
    let face = entity_key(&owner, document, 1, "start");
    assign_look(&mut owner, "body", &body.to_uppercase(), "default").unwrap();
    let body_state = assigned_assets(&owner, document);
    assert_eq!(
        body_state
            .assignments
            .body_appearances()
            .next()
            .unwrap()
            .0
            .reference_key(),
        body
    );
    assign_look(&mut owner, "face", &face, "default").unwrap();
    let face_state = assigned_assets(&owner, document);
    assert_eq!(
        face_state
            .assignments
            .face_appearances()
            .next()
            .unwrap()
            .0
            .reference_key(),
        face
    );
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(assigned_assets(&owner, document), body_state);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(assigned_assets(&owner, document), face_state);
    assert_eq!(
        owner.take_scene().unwrap().unwrap().status.visible_bodies,
        2
    );
}
#[test]
fn wrong_kind_unknown_namespace_and_failed_feature_never_create_assignments() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    let body = entity_key(&owner, document, 1, "body");
    let face = entity_key(&owner, document, 1, "start");
    let missing = entity_key(&owner, document, 99, "start");
    let foreign = SemanticReferenceName::new(
        "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee".parse().unwrap(),
        1,
        1,
        "sketch-prism",
        "start",
        &[1],
    )
    .unwrap()
    .encode()
    .unwrap()
    .iter()
    .map(|byte| format!("{byte:02x}"))
    .collect::<String>();
    let generation = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation();
    for (scope, key) in [
        ("body", &face),
        ("face", &body),
        ("face", &missing),
        ("face", &foreign),
    ] {
        assert!(assign_look(&mut owner, scope, key, "default").is_err());
        assert!(assigned_assets(&owner, document).is_empty());
        assert_eq!(
            owner
                .documents()
                .workspace()
                .session(document)
                .unwrap()
                .generation(),
            generation
        );
    }
    set_parameter(&mut owner, "depth", "0 cm");
    assert!(assign_look(&mut owner, "face", &face, "default").is_err());
    assert!(assigned_assets(&owner, document).is_empty());
    let independent = entity_key(&owner, document, 2, "start");
    assign_look(&mut owner, "face", &independent, "default").unwrap();
    assert_eq!(
        assigned_assets(&owner, document)
            .assignments
            .face_appearances()
            .count(),
        1
    );
}
#[test]
fn rejected_asset_lookup_does_not_consume_the_pending_first_scene() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let body = entity_key(&owner, document, 1, "body");
    assert!(assign_look(&mut owner, "body", &body, "missing").is_err());
    assert!(assigned_assets(&owner, document).is_empty());
    let scene = owner.take_scene().unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 1);
    assert!(!scene.scene.triangles.is_empty());
    assert!(owner.take_scene().unwrap().is_none());
}
#[test]
fn physical_assignment_binds_a_body_and_keeps_explicit_visual_intent() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let body = entity_key(&owner, document, 1, "body");
    assign_look(&mut owner, "part", "", "default").unwrap();
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: body.clone(),
            material_id: "steel".into(),
        })
        .unwrap();
    let assigned = assigned_assets(&owner, document);
    let reference = AssignmentReference::new(&body, vec![]).unwrap();
    assert_eq!(
        assigned.assignments.effective_material_id(&reference),
        Some("steel")
    );
    assert_eq!(assigned.assignments.part_appearance_id(), Some("default"));
}
