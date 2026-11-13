// SPDX-License-Identifier: GPL-2.0-only
//! Inspect the standard ribbon locations of an available command.
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

#[when("the user clicks \"2D Sketches\" in the available command list")]
fn click_sketch_command(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketches");
}

#[when("the user presses End and then Home while the pointer remains over that row")]
fn navigate_and_return(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::End, false);
    world.key(ViewportPhysicalKey::Home, false);
    world.pump(Vec::new());
}

#[then("a tooltip for \"2D Sketches\" lists its Part and Assembly locations")]
#[then("both locations identify the View tab and Visibility panel")]
fn shows_ribbon_locations(world: &mut ApplicationWorld) {
    world.expect_text("Part | View | Visibility");
    world.expect_text("Assembly | View | Visibility");
}

#[then("the target custom panel remains empty")]
fn target_remains_empty(world: &mut ApplicationWorld) {
    empty_tools_target(world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}
