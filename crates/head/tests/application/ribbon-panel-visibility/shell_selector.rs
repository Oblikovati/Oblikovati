// SPDX-License-Identifier: GPL-2.0-only
//! Observe Shell while its originating panel is part of the visibility workflow.
use super::*;

#[given("the full 3D Model ribbon displays the Modify panel")]
fn modify_panel_shown(world: &mut ApplicationWorld) {
    world.run_in_window("##application-ribbon", "Shell");
    assert!(ribbon_runs(world, "Modify").next().is_some());
}

#[when("the user starts Shell from Modify")]
fn start_shell(world: &mut ApplicationWorld) {
    let command = world.run_in_window("##application-ribbon", "Shell");
    world.click(pointer_pixels(command.center()));
    world.pump(Vec::new());
}

#[then("the Shell properties panel and model preview appear")]
fn shell_panel_and_preview(world: &mut ApplicationWorld) {
    world.run_in_window("Shell", "Thickness");
    assert!(
        world.state().extrude_preview_active,
        "Shell properties are visible, expected the model preview before any face is selected"
    );
}
