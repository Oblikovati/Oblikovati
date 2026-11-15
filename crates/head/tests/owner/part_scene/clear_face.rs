// SPDX-License-Identifier: GPL-2.0-only
//! Named face clear must preserve unrelated assignments and selection identity.
use super::*;
use oblikovati_api::wire::AssignAppearanceArgs;
use oblikovati_model::assets::DocumentAssets;

fn assign(owner: &mut TestOwner, scope: &str, key: &str) {
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: scope.into(),
            key: key.into(),
            appearance_id: "abs-black".into(),
        })
        .unwrap();
}
fn assets(owner: &TestOwner, document: u64) -> DocumentAssets {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .assets
        .clone()
}
fn face_fixture() -> (TestOwner, u64, String, String) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let body = super::assignments::entity_key(&owner, document, 1, "body");
    let end = super::assignments::entity_key(&owner, document, 1, "end");
    let start = super::assignments::entity_key(&owner, document, 1, "start");
    assign_face_fixture_assets(&mut owner, &body, &start, &end);
    owner.take_scene().unwrap();
    owner
        .click_model(document, [[1., 1., 10.], [0., 0., -1.]])
        .unwrap();
    (owner, document, end, start)
}
fn assign_face_fixture_assets(owner: &mut TestOwner, body: &str, start: &str, end: &str) {
    assign(owner, "part", "");
    Client::new(owner.document_host())
        .materials()
        .assign(&AssignMaterialArgs {
            body_key: body.into(),
            material_id: "steel".into(),
        })
        .unwrap();
    assign(owner, "body", body);
    assign(owner, "face", start);
    assign(owner, "face", end);
}
#[test]
fn clear_selected_named_face_is_one_reversible_edit() {
    let (mut owner, document, end, start) = face_fixture();
    let before = assets(&owner, document);
    let history = history_count(&owner, document);
    assert!(
        owner
            .clear_selected_face_appearance(document, &end)
            .unwrap()
    );
    let cleared = assets(&owner, document);
    assert_only_face_cleared(&before, &cleared, &end, &start);
    assert_eq!(history_count(&owner, document), history + 1);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_eq!(assets(&owner, document), before);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_eq!(assets(&owner, document), cleared);
    let reopened = save_and_reopen(&mut owner, document);
    assert_eq!(assets(&owner, reopened), cleared);
}
fn history_count(owner: &TestOwner, document: u64) -> usize {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len()
}
fn assert_only_face_cleared(
    before: &DocumentAssets,
    cleared: &DocumentAssets,
    end: &str,
    start: &str,
) {
    let mut expected = before.clone();
    let reference = oblikovati_model::assets::AssignmentReference::new(end, vec![]).unwrap();
    expected
        .assignments
        .set_face_appearance(&reference, "")
        .unwrap();
    assert_eq!(cleared, &expected);
    assert_eq!(cleared.assignments.body_appearances().count(), 1);
    let remaining: Vec<_> = cleared
        .assignments
        .face_appearances()
        .map(|(key, _)| key.reference_key())
        .collect();
    assert_eq!(remaining, [start]);
}
#[test]
fn clear_selected_named_face_rejects_forged_and_stale_selection() {
    let (mut owner, document, end, start) = face_fixture();
    let before = assets(&owner, document);
    assert!(
        owner
            .clear_selected_face_appearance(document, &start)
            .is_err()
    );
    assert_eq!(assets(&owner, document), before);
    set_parameter(&mut owner, "depth", "3 cm");
    let after_edit = assets(&owner, document);
    assert!(
        owner
            .clear_selected_face_appearance(document, &end)
            .is_err()
    );
    assert_eq!(assets(&owner, document), after_edit);
}

fn save_and_reopen(owner: &mut TestOwner, document: u64) -> u64 {
    let path = "/memory/cleared-face.opd";
    Client::new(owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    close_face_fixture(owner, document);
    Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}

fn close_face_fixture(owner: &mut TestOwner, document: u64) {
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
}
