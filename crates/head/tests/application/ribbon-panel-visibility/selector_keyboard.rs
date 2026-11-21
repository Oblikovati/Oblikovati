// SPDX-License-Identifier: GPL-2.0-only
//! Keyboard focus remains on the chosen panel when Home or End is pressed.
use super::super::diagnostics::window_geometry;
use super::*;
use oblikovati_viewport::ViewportPhysicalKey as Key;

fn focus_id() -> u32 {
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected GUI");
    context.NavId
}

#[given("the full top-docked 3D Model ribbon is visible")]
fn top_model_ribbon(world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-column").is_none());
    world.expect_text("Start 2D Sketch");
}

#[given("the direct panel selector is open with Create focused and checked")]
fn create_focused(world: &mut ApplicationWorld) {
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    world.click([size[0] as i32 - 22, 45]);
    world.key(Key::DownArrow, false);
    world.key(Key::DownArrow, false);
    assert!(world.checked("Create"));
    world.key(Key::Space, false);
    assert!(
        !world.checked("Create"),
        "expected focused Create to toggle"
    );
    world.key(Key::Space, false);
    assert!(world.checked("Create"));
    let focused = focus_id();
    assert_ne!(focused, 0, "expected a focused panel row");
    world.panel_selector_focus = Some(focused);
}

#[when("the user presses Home")]
fn press_home(world: &mut ApplicationWorld) {
    world.key(Key::Home, false);
}

#[when("the user presses End")]
fn press_end(world: &mut ApplicationWorld) {
    world.key(Key::End, false);
}

#[then("Create remains focused and checked")]
fn create_still_focused(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(focus_id()),
        world.panel_selector_focus,
        "expected Create focus"
    );
    assert!(world.checked("Create"));
}

#[then("the panel selector remains open")]
fn selector_open(_: &mut ApplicationWorld) {
    assert!(super::side_selector::popup_depth() > 0);
}

#[then("the panel selector closes")]
fn selector_closed(_: &mut ApplicationWorld) {
    assert_eq!(super::side_selector::popup_depth(), 0);
}
