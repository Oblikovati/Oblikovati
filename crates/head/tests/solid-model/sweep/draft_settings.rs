// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[when("Advanced Properties is expanded")]
fn show_advanced(world: &mut ApplicationWorld) {
    click_panel(world, "AdvancedProperties");
}

#[then("Optimize for Single Selection is shown unchecked")]
fn optimization_defaults_unchecked(world: &mut ApplicationWorld) {
    let checkbox = world.run_in_window("Sweep", "OptimizeforSingleSelection");
    assert!(!world.checked_run(checkbox));
}

#[given("Sweep1 is open for editing with a valid preview")]
fn open_valid_sweep(world: &mut ApplicationWorld) {
    sweep_is_open_for_editing(world);
    assert!(world.run_is_enabled("OK"));
}

#[when("Optimize for Single Selection is checked and OK is clicked")]
fn accept_with_single_selection_checked(world: &mut ApplicationWorld) {
    reveal_advanced(world);
    click_panel(world, "OptimizeforSingleSelection");
    let checkbox = world.run_in_window("Sweep", "OptimizeforSingleSelection");
    assert!(world.checked_run(checkbox));
    accept(world);
    assert!(super::super::diagnostics::window_geometry("Sweep").is_none());
    accepted_sweep(world);
}

#[then("the checkbox is shown unchecked")]
fn reopened_optimization_is_unchecked(world: &mut ApplicationWorld) {
    reveal_advanced(world);
    optimization_defaults_unchecked(world);
    taper_is_retained(world);
}

fn reveal_advanced(world: &mut ApplicationWorld) {
    if !super::super::world::squeezed(&world.window_text("Sweep"))
        .contains("OptimizeforSingleSelection")
    {
        show_advanced(world);
    }
}
