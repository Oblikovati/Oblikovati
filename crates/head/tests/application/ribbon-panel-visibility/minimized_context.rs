// SPDX-License-Identifier: GPL-2.0-only
//! Inspect model panel visibility from a minimized ribbon tab.
use super::*;
use cucumber::{given, then, when};

const SIMULATION: &str = "Simulation";

#[given("the top-docked ribbon is minimized to tabs with 3D Model active")]
fn minimized_model(world: &mut ApplicationWorld) {
    super::super::ribbon_display::click_display_menu(world);
    world.click_text("Minimize to Tabs");
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}
#[given("the Modify panel is visible and Parameters is hidden")]
fn configured_panels(world: &mut ApplicationWorld) {
    right_click_model(world);
    open_show_panels(world);
    panel_checks(world);
    escape_twice(world);
}
#[when("the user right-clicks the 3D Model tab")]
fn right_click_model(world: &mut ApplicationWorld) {
    let position = super::super::diagnostics::pointer_pixels(world.run("3D Model").center());
    world.right_click(position);
}
#[then("the ribbon context menu is displayed")]
#[then("the ribbon context menu opens")]
fn context_open(world: &mut ApplicationWorld) {
    world.expect_text("Show Panels");
}
#[when("the user opens Show Panels")]
fn open_show_panels(world: &mut ApplicationWorld) {
    world.click_text("Show Panels");
}
#[then("the submenu lists the 3D Model panels")]
fn model_panels(world: &mut ApplicationWorld) {
    for name in ["Sketch", "Create", "Modify", "Parameters"] {
        world.expect_text(name);
    }
}
#[then("Modify is checked and Parameters is unchecked")]
fn panel_checks(world: &mut ApplicationWorld) {
    assert!(world.checked("Modify"));
    assert!(!world.checked(PARAMETERS));
}
#[then("the ribbon remains minimized to tabs")]
fn still_minimized(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}
#[when("the user presses Escape twice")]
fn escape_twice(world: &mut ApplicationWorld) {
    world.escape();
    world.escape();
}
#[then("both menus close")]
fn menus_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panels");
}

#[when("the user selects the unchecked Parameters entry")]
fn enable_parameters(world: &mut ApplicationWorld) {
    assert!(!world.checked(PARAMETERS));
    world.click_text(PARAMETERS);
}
#[then("the menus close and the ribbon remains minimized to tabs")]
fn menus_close_without_expanding(world: &mut ApplicationWorld) {
    menus_closed(world);
    still_minimized(world);
}
#[when("the user clicks the 3D Model tab once")]
fn open_temporary_tab(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[then("the temporary command area opens")]
fn temporary_tab_open(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    assert!(super::super::diagnostics::window_geometry("##ribbon-temporary-panel").is_some());
}
#[then("it contains the Parameters panel")]
fn temporary_parameters(world: &mut ApplicationWorld) {
    let (position, size) =
        super::super::diagnostics::window_geometry("##ribbon-temporary-panel").unwrap();
    assert!(world.frame.runs.iter().any(|run| run.text == PARAMETERS
        && run.center()[0] >= position[0]
        && run.center()[0] <= position[0] + size[0]
        && run.center()[1] >= position[1]
        && run.center()[1] <= position[1] + size[1]));
}
#[when("the user double-clicks the 3D Model tab")]
fn restore_full_ribbon(world: &mut ApplicationWorld) {
    let position = super::super::diagnostics::pointer_pixels(world.run("3D Model").center());
    world.double_click(position);
}
#[then("the full ribbon is restored")]
fn full_ribbon(world: &mut ApplicationWorld) {
    assert!(
        world.shows("Start 2D Sketch") || world.shows("User Interface"),
        "expected full 3D Model or View ribbon commands"
    );
    assert!(super::super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the Parameters panel is visible")]
fn parameters_visible(world: &mut ApplicationWorld) {
    expect_parameters_panel(world);
}

#[given("the Simulation panel is enabled for 3D Model")]
fn simulation_enabled(world: &mut ApplicationWorld) {
    right_click_model(world);
    open_show_panels(world);
    simulation_checked(world);
    escape_twice(world);
}

#[then("Simulation is checked in the panel list")]
fn simulation_checked(world: &mut ApplicationWorld) {
    world.expect_text(SIMULATION);
    assert!(world.checked(SIMULATION));
}

#[then("Simulation is unchecked")]
fn simulation_unchecked(world: &mut ApplicationWorld) {
    world.expect_text(SIMULATION);
    assert!(!world.checked(SIMULATION));
}

#[when("the user selects Simulation")]
fn select_simulation(world: &mut ApplicationWorld) {
    world.click_text(SIMULATION);
}

#[then("the temporary ribbon opens without a Simulation panel")]
fn temporary_ribbon_omits_simulation(world: &mut ApplicationWorld) {
    temporary_tab_open(world);
    world.expect_no_text(SIMULATION);
}

#[then("the menus and temporary ribbon close")]
fn menus_and_temporary_ribbon_close(world: &mut ApplicationWorld) {
    menus_close_without_expanding(world);
    assert!(super::super::diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
}

#[then("the full ribbon is visible with the Simulation panel")]
fn full_ribbon_shows_simulation(world: &mut ApplicationWorld) {
    full_ribbon(world);
    assert!(ribbon_runs(world, SIMULATION).next().is_some());
    world.expect_text("Stress Analysis");
}
