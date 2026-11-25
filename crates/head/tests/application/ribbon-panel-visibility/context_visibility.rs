// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[given("Modify is visible while Parameters is hidden")]
fn default_model_panels(world: &mut ApplicationWorld) {
    modify_still_visible(world);
    parameters_hidden(world);
}

#[then("Modify remains visible")]
fn modify_still_visible(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "Modify").next().is_some());
}

#[when("the user opens the ribbon context menu and expand Show Panels")]
#[when("the user reopens Show Panels from the ribbon context menu")]
#[when("the user opens Show Panels on the left-docked ribbon")]
#[when("the user opens Show Panels on the floating ribbon")]
#[when("the user opens Show Panels on the top ribbon")]
fn show_context_panels(world: &mut ApplicationWorld) {
    expand_show_panels(world);
}

#[when("the user hides Modify through Show Panels")]
fn hide_modify_in_context(world: &mut ApplicationWorld) {
    hide_modify(world);
}

#[when("the user docks the ribbon on the left")]
#[when("the user docks the floating ribbon at the left")]
fn dock_ribbon_left(world: &mut ApplicationWorld) {
    dock_left(world);
}

#[when("the user selects Modify in Show Panels")]
fn select_modify_with_reflow(world: &mut ApplicationWorld) {
    let was_checked = world.checked("Modify");
    if was_checked {
        world.ribbon_height = side_selector::side_group_y(world, "Work Features");
    }
    toggle_panel(world, "Modify");
    if !was_checked {
        assert!(side_selector::side_group_y(world, "Modify") > 0.);
        expand_show_panels(world);
    }
}

#[then("the remaining ribbon groups return to their earlier positions")]
fn groups_return_to_their_positions(world: &mut ApplicationWorld) {
    world.escape();
    world.escape();
    assert_eq!(
        side_selector::side_group_y(world, "Work Features"),
        world.ribbon_height
    );
}

#[then(
    "the context menu offers Ribbon Appearance, Show Panels, Customize User Commands, Undock Ribbon and Docking Positions"
)]
fn panel_context_commands(world: &mut ApplicationWorld) {
    for label in [
        "Ribbon Appearance",
        "Show Panels",
        "Customize User Commands",
        "Undock Ribbon",
        "Docking Positions",
    ] {
        world.expect_text(label);
    }
}
