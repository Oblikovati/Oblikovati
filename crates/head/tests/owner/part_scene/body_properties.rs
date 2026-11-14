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
