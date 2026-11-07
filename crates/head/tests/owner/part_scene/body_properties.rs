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
