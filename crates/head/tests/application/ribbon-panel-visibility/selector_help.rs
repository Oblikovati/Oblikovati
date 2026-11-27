// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[when("the user closes it with the same selector control")]
fn close_selector_with_its_button(world: &mut ApplicationWorld) {
    click_selector(world);
    assert!(!selector_popup_open());
}

#[then("the Show Panels help tooltip appears")]
fn selector_hint(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
}

#[then("it explains that the down arrow displays more ribbon panels")]
fn arrow_instruction(world: &mut ApplicationWorld) {
    world.expect_text("The down arrow displays more ribbon panels.");
}

#[then("it explains that selecting or clearing a checkbox adds or removes a panel")]
fn checkbox_instruction(world: &mut ApplicationWorld) {
    world.expect_text("Select a checkbox to show its panel.");
    world.expect_text("Clear a checkbox to hide its panel.");
}

#[then("it offers additional help through F1")]
fn f1_instruction(world: &mut ApplicationWorld) {
    world.expect_text("Press F1 for more help.");
}

#[then("the tooltip closes")]
fn selector_hint_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("The down arrow displays more ribbon panels.");
    world.expect_no_text("Press F1 for more help.");
}

#[test]
fn f1_opens_the_panel_selector_help() {
    use oblikovati_viewport::ViewportPhysicalKey::{Escape, F1};
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_side_selector(&mut world);
    close_selector_with_its_button(&mut world);
    f1_instruction(&mut world);
    world.key(F1, false);
    world.expect_text("Ribbon Panel Help");
    world.expect_text("Arrow keys move between panels.");
    world.expect_text("Space changes the focused checkbox.");
    world.key(Escape, false);
    world.expect_no_text("Ribbon Panel Help");
    world.expect_text("Start 2D Sketch");
    world.close();
}
