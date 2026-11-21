// SPDX-License-Identifier: GPL-2.0-only
//! Interaction with an expanded Modify command's context menu (#3856).
use super::{ApplicationWorld, diagnostics};
use cucumber::{then, when};

#[when("the user right-clicks \"Move Bodies\" in the additional area")]
#[when("the user right-clicks Move Bodies in the additional area")]
fn right_click_move_body(world: &mut ApplicationWorld) {
    let button = world
        .run_in_window("##ribbon-additional-PartModify", "MoveBodies")
        .center();
    world.right_click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
}
#[then("a context menu offers \"Move to Main Panel\"")]
fn context_has_placement(world: &mut ApplicationWorld) {
    world.expect_text("Move to Main Panel");
}
#[then("the menu offers \"Add to Quick Access Toolbar\"")]
fn context_has_quick_access(world: &mut ApplicationWorld) {
    world.expect_text("Add to Quick Access Toolbar");
}
#[then("its command context menu is visible")]
fn context_is_visible(world: &mut ApplicationWorld) {
    context_has_placement(world);
    context_has_quick_access(world);
}
#[then("the command context menu and Modify additional command area close")]
fn command_context_and_expansion_close(world: &mut ApplicationWorld) {
    world.expect_no_text("Add to Quick Access Toolbar");
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
}
