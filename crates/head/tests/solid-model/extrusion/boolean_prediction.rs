// SPDX-License-Identifier: GPL-2.0-only
use super::*;

const PREDICTION: &str = "PredictBooleanOperation";

#[given(
    "`fixtures/cad/extrusion/extrusion-base-circle.opd` is open in Extrude with the circular profile selected"
)]
fn circular_preview(world: &mut ApplicationWorld) {
    published_circle_part(world);
    start_circle_extrude(world);
}

#[given("Advanced Settings offers Predict Boolean Operation")]
fn prediction_is_available(world: &mut ApplicationWorld) {
    click_panel(world, "AdvancedSettings");
    assert!(world.checked_run(popup_option(world, PREDICTION)));
}

#[when("the user disables Predict Boolean Operation")]
fn disable_prediction(world: &mut ApplicationWorld) {
    choose_popup_option(world, PREDICTION);
    click_panel(world, "AdvancedSettings");
    assert!(!world.checked_run(popup_option(world, PREDICTION)));
    world.escape();
}

#[when("the user specifies the Boolean operation and direction explicitly")]
fn choose_manual_cut(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    click_panel(world, "Direction");
    choose_popup_option(world, "Dropdown");
    click_panel(world, "Cut");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "Default");
    assert_top_opening(world, false);
    assert!(world.status_text().contains("Commandfailed"));

    click_panel(world, "Default");
    choose_popup_option(world, "Flipped");
    assert_eq!(world.state().document_generation, generation);
}

#[then("the preview uses the specified operation and direction")]
fn preview_uses_explicit_cut(world: &mut ApplicationWorld) {
    flipped_cut_preview(world);
    assert!(!world.status_text().contains("Commandfailed"));
    world.save_capture("extrude-manual-boolean");
}

fn popup_option<'w>(world: &'w ApplicationWorld, label: &str) -> &'w diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == label && world.frame.is_popup_row(run))
        .unwrap_or_else(|| panic!("expected {label:?} in the popup"))
}

fn choose_popup_option(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(popup_option(world, label).center());
    world.click(point);
    world.pump(Vec::new());
}
