// SPDX-License-Identifier: GPL-2.0-only
//! Switching ribbons while a part's Measure panel is open.
use super::*;

#[when("the user selects Measure")]
fn select_measure(world: &mut ApplicationWorld) {
    let command = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Measure")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("Measure: expected the ribbon command");
    world.click(super::super::diagnostics::pointer_pixels(command.center()));
}
#[then("the Measure panel opens over the workspace")]
#[then("the Measure panel remains open")]
fn measure_open(world: &mut ApplicationWorld) {
    assert!(window_geometry("Measure").is_some());
    world.expect_text("Select a solid face or edge to measure");
}
#[when("the user selects the View tab")]
fn select_view(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("the View ribbon command groups are displayed")]
#[then("the View ribbon remains visible")]
fn view_visible(world: &mut ApplicationWorld) {
    world.expect_text("User Interface");
    world.expect_text("Navigate");
}
#[when("the user selects Done in the Measure panel")]
fn finish_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
}
#[then("the Measure panel closes")]
fn measure_closed(_world: &mut ApplicationWorld) {
    assert!(window_geometry("Measure").is_none());
}
