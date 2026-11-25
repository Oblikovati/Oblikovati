// SPDX-License-Identifier: GPL-2.0-only
//! Verify a floating asset panel while switching between a blank and a solid Part.
use super::{ApplicationWorld, diagnostics, workspace_layout};
use cucumber::{given, then, when};
use workspace_layout::dockspace_native::open_toolbar_tool;

const APPEARANCE: &str = "Appearance Browser";

#[given("the Model Browser is docked on the left")]
fn model_browser_docked(world: &mut ApplicationWorld) {
    super::browser_scenarios::model_browser_left(world);
    assert!(diagnostics::window_dock_id("Model").is_some_and(|dock| dock != 0));
}

#[given("the Appearance Browser is open as a floating window")]
fn floating_appearance(world: &mut ApplicationWorld) {
    open_toolbar_tool(world, APPEARANCE);
    world.floating_appearance_position = Some(appearance_position());
    workspace_layout::browser_feature::expect_solid_visible(world);
}

#[when("the user switches to `PartOne.opd`")]
fn switch_to_part_one(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.other_document.expect("expected PartOne"));
}

#[then("the viewport and Model Browser show the newly active document")]
fn part_one_is_visible(world: &mut ApplicationWorld) {
    let part = world.other_document.expect("expected PartOne");
    assert_eq!(world.state().document, Some(part));
    world.expect_browser_of(part);
    super::super::viewport_background::expect_empty_part(world);
}

#[then("the floating Appearance Browser remains open in the same position")]
fn floating_appearance_retained(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(appearance_position()),
        world.floating_appearance_position
    );
}

#[when("the user switches back to `docking-visible-solid-current.opd`")]
fn switch_to_solid_part(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.previous_document.expect("expected the solid Part"));
}

#[then("that document is active again")]
fn solid_part_is_visible(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the solid Part");
    assert_eq!(world.state().document, Some(part));
    world.expect_browser_of(part);
    workspace_layout::browser_feature::expect_solid_visible(world);
}

fn appearance_position() -> [f32; 2] {
    assert_eq!(diagnostics::window_dock_id(APPEARANCE), Some(0));
    diagnostics::window_geometry(APPEARANCE)
        .expect("expected the floating Appearance Browser")
        .0
}
