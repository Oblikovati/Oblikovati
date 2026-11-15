// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const SHIPPED_BLACK: &str = "abs-black";
fn only_body(owner: &mut TestOwner, document: u64) -> oblikovati_gui::GuiBrowserBody {
    // The scene evaluates the part first, as a frame would.
    owner.take_scene().unwrap();
    let browser = owner.gui_browser_snapshot(document).unwrap();
    assert_eq!(browser.bodies.len(), 1, "bodies");
    browser.bodies[0].clone()
}
#[test]
fn physical_values_wait_for_update_and_go_stale_with_the_model() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    assert_eq!(only_body(&mut owner, document).physical, None);
    assert!(owner.update_body_physical(document, 0).unwrap());
    let shown = only_body(&mut owner, document).physical.unwrap();
    assert!(shown.volume.ends_with("mm^3"), "{shown:?}");
    set_parameter(&mut owner, "depth", "3 cm");
    assert_eq!(only_body(&mut owner, document).physical, None);
    let error = owner.update_body_physical(document, 9).unwrap_err();
    assert!(error.contains("body index 9"), "{error}");
}
#[test]
fn a_body_appearance_can_be_set_cleared_and_undone() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let key = only_body(&mut owner, document).key;
    assert_eq!(only_body(&mut owner, document).appearance, "");
    owner
        .set_body_appearance(document, &key, SHIPPED_BLACK)
        .unwrap();
    assert_eq!(only_body(&mut owner, document).appearance, SHIPPED_BLACK);
    let browser = owner.gui_browser_snapshot(document).unwrap();
    assert!(
        browser
            .appearances
            .iter()
            .any(|row| row.id == SHIPPED_BLACK)
    );
    owner.set_body_appearance(document, &key, "").unwrap();
    assert_eq!(only_body(&mut owner, document).appearance, "");
}

#[test]
fn applying_a_body_appearance_reveals_it_without_selection_tint() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let body = only_body(&mut owner, document);
    owner
        .files
        .commands(document)
        .unwrap()
        .set_browser_body_selection(&body.key, body.feature);
    owner
        .set_body_appearance_from_gui(document, &body.key, SHIPPED_BLACK)
        .unwrap();
    assert_eq!(
        owner.gui_browser_snapshot(document).unwrap().selected_body,
        None
    );
    owner
        .files
        .commands(document)
        .unwrap()
        .set_browser_body_selection(&body.key, body.feature);
    owner
        .set_body_appearance_from_gui(document, &body.key, "")
        .unwrap();
    assert_eq!(
        owner.gui_browser_snapshot(document).unwrap().selected_body,
        None
    );
    let scene = owner.take_scene().unwrap().unwrap();
    assert!(!scene.scene.triangles.is_empty());
    let default = owner
        .files
        .builtin_catalog()
        .unwrap()
        .appearances()
        .get("default")
        .unwrap();
    let color = oblikovati_api::contract::Appearance::base(default).color;
    assert!(
        scene
            .scene
            .triangles
            .iter()
            .all(|triangle| triangle.color == [color.r, color.g, color.b, 1.])
    );
}

#[test]
fn refusing_an_unknown_appearance_keeps_the_body_selected() {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, false);
    let body = only_body(&mut owner, document);
    owner
        .files
        .commands(document)
        .unwrap()
        .set_browser_body_selection(&body.key, body.feature);
    assert!(
        owner
            .set_body_appearance_from_gui(document, &body.key, "missing-appearance")
            .is_err()
    );
    let browser = owner.gui_browser_snapshot(document).unwrap();
    assert_eq!(browser.selected_body, Some(body.feature));
    assert_eq!(browser.bodies[0].appearance, "");
}

#[test]
fn clearing_two_selected_body_overrides_is_one_atomic_history_step() {
    let (mut owner, document, bodies) = clear_two_body_fixture();
    let keys = select_two_bodies(&mut owner, document, &bodies);
    let untouched = assign_unrelated_face(&mut owner, document);
    owner.take_scene().unwrap();
    let before = clear_history_count(&owner, document);
    assert!(
        owner
            .clear_selected_body_appearances(document, &keys)
            .unwrap()
    );
    assert_eq!(clear_history_count(&owner, document), before + 1);
    assert_clear_assets(&owner, document, 0, &untouched);
    owner.files.commands(document).unwrap().undo().unwrap();
    assert_clear_assets(&owner, document, 2, &untouched);
    owner.files.commands(document).unwrap().redo().unwrap();
    assert_clear_assets(&owner, document, 0, &untouched);
}
#[test]
fn clear_rejects_changed_body_selection_without_mutation() {
    let (mut owner, document, bodies) = clear_two_body_fixture();
    let keys: Vec<_> = bodies.iter().map(|body| body.key.clone()).collect();
    owner
        .files
        .commands(document)
        .unwrap()
        .set_browser_body_selection(&keys[0], bodies[0].feature);
    let session = owner.files.workspace().session(document).unwrap();
    let before = session.snapshot().assets.clone();
    let history: Vec<String> = session
        .undo_labels()
        .iter()
        .map(|label| (*label).to_owned())
        .collect();
    assert!(
        owner
            .clear_selected_body_appearances(document, &keys)
            .is_err()
    );
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(session.snapshot().assets, before);
    assert_eq!(session.undo_labels(), history);
}
#[test]
fn clear_rejects_a_stale_or_forged_body_without_changing_valid_overrides() {
    use oblikovati_session::selection::SelectionClickMode::{Replace, Toggle};
    let (mut owner, document, bodies) = clear_two_body_fixture();
    let first = &bodies[0];
    let keys = vec![first.key.clone(), "not-a-current-body".into()];
    let mut selection = owner.files.commands(document).unwrap();
    selection.apply_browser_body_click(&keys[0], first.feature, Replace);
    selection.apply_browser_body_click(&keys[1], FeatureId(99), Toggle);
    let before = clear_asset_snapshot(&owner, document);
    assert!(
        owner
            .clear_selected_body_appearances(document, &keys)
            .is_err()
    );
    assert_eq!(clear_asset_snapshot(&owner, document), before);
}
fn clear_two_body_fixture() -> (TestOwner, u64, Vec<oblikovati_gui::GuiBrowserBody>) {
    let (mut owner, _) = owner();
    let document = open_prism(&mut owner, true);
    owner.take_scene().unwrap();
    let bodies = owner.gui_browser_snapshot(document).unwrap().bodies;
    assert_eq!(bodies.len(), 2);
    for body in &bodies {
        owner
            .assign_body_appearance(document, &body.key, SHIPPED_BLACK)
            .unwrap();
    }
    owner.take_scene().unwrap();
    (owner, document, bodies)
}
fn select_two_bodies(
    owner: &mut TestOwner,
    document: u64,
    bodies: &[oblikovati_gui::GuiBrowserBody],
) -> Vec<String> {
    use oblikovati_session::selection::SelectionClickMode::{Replace, Toggle};
    let keys: Vec<_> = bodies.iter().map(|body| body.key.clone()).collect();
    let mut selection = owner.files.commands(document).unwrap();
    selection.apply_browser_body_click(&keys[0], bodies[0].feature, Replace);
    selection.apply_browser_body_click(&keys[1], bodies[1].feature, Toggle);
    keys
}
fn assign_unrelated_face(
    owner: &mut TestOwner,
    document: u64,
) -> oblikovati_model::assets::AssignmentReference {
    let face = super::assignments::entity_key(owner, document, 1, "start");
    Client::new(owner.document_host())
        .appearances()
        .assign(&AssignAppearanceArgs {
            scope: "face".into(),
            key: face.clone(),
            appearance_id: SHIPPED_BLACK.into(),
        })
        .unwrap();
    oblikovati_model::assets::AssignmentReference::new(&face, vec![]).unwrap()
}
fn assert_clear_assets(
    owner: &TestOwner,
    document: u64,
    count: usize,
    face: &oblikovati_model::assets::AssignmentReference,
) {
    let session = owner.files.workspace().session(document).unwrap();
    let assignments = &session.snapshot().assets.assignments;
    assert_eq!(assignments.body_appearances().count(), count);
    let actual = assignments
        .face_appearances()
        .find(|(key, _)| *key == face)
        .map(|(_, id)| id);
    assert_eq!(actual, Some(SHIPPED_BLACK));
}

fn clear_history_count(owner: &TestOwner, document: u64) -> usize {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len()
}
fn clear_asset_snapshot(
    owner: &TestOwner,
    document: u64,
) -> oblikovati_model::assets::DocumentAssets {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .assets
        .clone()
}
