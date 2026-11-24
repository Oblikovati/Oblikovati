// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::diagnostics::{pointer_pixels, window_geometry};
use super::*;

#[given("the Modify panel is visible with its additional command area open and unpinned")]
fn modify_unpinned(world: &mut ApplicationWorld) {
    main_modify_visible(world);
    super::super::super::ribbon_panel_expansion::opened_unpinned(world);
}

#[then("the selector and the Modify additional commands are visible together")]
fn full_expansion_with_selector(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    world.run_in_window(EXPANSION, "MoveBodies");
}

#[when("the user opens the panel selector and clears Modify")]
fn open_selector_and_hide_modify(world: &mut ApplicationWorld) {
    side_selector::open_side_selector(world);
    clear_modify(world);
}

#[when("the user clears Modify in the panel selector")]
fn clear_modify(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"));
    let (_, size) = window_geometry(EXPANSION).expect("expected open Modify expansion");
    world.observed_view = Some([0, size[1] as i32]);
    world.click_text("Modify");
}

#[then("Modify is unchecked and its main panel and additional commands are hidden")]
fn unchecked_modify_without_commands(world: &mut ApplicationWorld) {
    assert!(!world.checked("Modify"));
    no_modify_commands(world);
}

#[then("the Modify main panel and additional commands are hidden")]
#[then("the Modify panel remains hidden")]
fn no_modify_commands(world: &mut ApplicationWorld) {
    world.expect_no_text("Fillet");
    world.expect_no_text("Move Bodies");
}

#[then("a small Modify title strip remains beneath the ribbon")]
#[then("a small Modify title strip with a pin remains visible beneath the ribbon")]
fn hidden_title_strip(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry(EXPANSION).expect("expected the retained Modify title");
    let (ribbon, ribbon_size) = window_geometry("##application-ribbon").unwrap();
    let expanded_height = world.observed_view.expect("expected full expansion size")[1] as f32;
    assert!(size[1] < expanded_height * 0.5);
    assert!(position[1] >= ribbon[1] + ribbon_size[1]);
    world.run_in_window(EXPANSION, "Modify");
    world.run_in_window(EXPANSION, "Pin");
}

#[then("Modify is checked and its main panel and additional commands are visible again")]
fn modify_and_expansion_restored(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"));
    main_modify_visible(world);
    full_expansion_with_selector(world);
}

#[when("the user clicks the remaining Modify title strip")]
fn dismiss_title(world: &mut ApplicationWorld) {
    let title = world.run_in_window(EXPANSION, "Modify");
    world.click(pointer_pixels(title.center()));
}

#[then("the title strip and panel selector close")]
fn strip_and_selector_close(world: &mut ApplicationWorld) {
    expansion_closed(world);
    assert!(!side_selector::selector_popup_open());
}

#[when("the user opens the panel selector and selects Modify")]
fn open_selector_and_restore_modify(world: &mut ApplicationWorld) {
    side_selector::open_side_selector(world);
    assert!(!world.checked("Modify"));
    world.click_text("Modify");
}

#[then("the Modify main panel returns")]
fn modify_returns(world: &mut ApplicationWorld) {
    main_modify_visible(world);
}

#[when("the user clicks the panel selector control")]
fn dismiss_selector_with_its_button(world: &mut ApplicationWorld) {
    side_selector::click_selector(world);
}

#[then("the selector closes and the Modify main panel remains visible")]
fn modify_remains_after_selector(world: &mut ApplicationWorld) {
    assert!(!side_selector::selector_popup_open());
    main_modify_visible(world);
}
