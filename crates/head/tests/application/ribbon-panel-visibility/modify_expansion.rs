// SPDX-License-Identifier: GPL-2.0-only
use super::*;

const EXPANSION: &str = "##ribbon-additional-PartModify";

#[when("the user opens Show Panels from the active tab context menu")]
#[when("the user reopens Show Panels from the active tab context menu")]
fn model_tab_show_panels(world: &mut ApplicationWorld) {
    open_show_panels_at(world, "3D Model");
}

#[then("Modify is checked in the panel list")]
fn modify_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"));
}

#[then("the Modify panel and its additional command area are hidden")]
fn modify_and_expansion_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Modify");
    world.expect_no_text("Fillet");
    expansion_closed(world);
}

#[then("the main Modify panel is visible")]
fn main_modify_visible(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "Modify").next().is_some());
    world.expect_text("Fillet");
}

#[then("the additional command area remains closed")]
fn expansion_closed(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(EXPANSION).is_none());
    world.expect_no_text("Pin");
}
