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

const PRESET: &str = "FollowPath_0deg_90deg";

#[given(
    "the preset name Follow Path_0 deg_90 deg is available for creation in an isolated preference profile"
)]
fn preset_name_is_available(world: &mut ApplicationWorld) {
    assert!(!preset_file(world).exists());
}

#[given(
    "Sweep1 is open for editing with Follow Path, taper 0 degrees and twist 90 degrees entered"
)]
fn edit_sweep_for_preset(world: &mut ApplicationWorld) {
    open_valid_sweep(world);
    replace_angle(world, "Taper", "0");
    replace_angle(world, "Twist", "90");
    assert!(world.run_is_enabled("OK"));
}

#[when("the preset plus control is clicked")]
fn save_sweep_preset(world: &mut ApplicationWorld) {
    click_panel(world, "+");
}

#[then("a preset named Follow Path_0 deg_90 deg is shown")]
fn sweep_preset_is_saved(world: &mut ApplicationWorld) {
    let file = preset_file(world);
    let bytes = std::fs::read(&file).unwrap_or_else(|cause| {
        panic!("expected the preset plus control to save {file:?}: {cause}")
    });
    let settings: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
    assert_eq!(
        settings["entries"]["Follow Path_0 deg_90 deg"],
        serde_json::json!({"orientation":"follow_path","taper":"0 deg","twist":"90 deg"})
    );
    world.run_in_window("Sweep", PRESET);
}

#[when("No Preset is selected and Twist is changed to 0")]
fn no_preset_and_zero_twist(world: &mut ApplicationWorld) {
    click_panel(world, PRESET);
    click_orientation_option(world, "NoPreset");
    replace_angle(world, "Twist", "0");
    assert_eq!(world.state().sweep_preview.unwrap().request.twist, "0");
}

#[when("the saved preset is selected")]
fn select_saved_preset(world: &mut ApplicationWorld) {
    click_panel(world, "NoPreset");
    click_orientation_option(world, PRESET);
}

#[then("the profile and path remain selected")]
fn preset_keeps_sources(world: &mut ApplicationWorld) {
    chosen_sources(world);
    let draft = world.state().sweep_preview.unwrap().request;
    assert_eq!(draft.taper, "0 deg");
    assert!(!draft.path_cleared);
    assert_eq!(draft.profile, oblikovati_gui::GuiSweepProfile::Saved);
}

fn preset_file(world: &mut ApplicationWorld) -> std::path::PathBuf {
    world
        .app()
        .window
        .native
        .ribbon_settings_directory
        .as_ref()
        .unwrap()
        .join("sweep-presets.json")
}

#[given("the user edits Sweep1, enters 45 degrees for Twist and accepts")]
fn accept_45_degree_twist(world: &mut ApplicationWorld) {
    open_valid_sweep(world);
    replace_angle(world, "Twist", "45");
    assert!(world.run_is_enabled("OK"));
    accept(world);
    accepted_sweep(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(
        browser.sweep_edits[&browser.bodies[0].feature].twist,
        "45 deg"
    );
}

#[given("Sweep1 is reopened for editing and the Twist expression menu contains 45")]
fn show_recent_twist(world: &mut ApplicationWorld) {
    open_valid_sweep(world);
    click_panel(world, "Recent");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "45" && world.frame.is_popup_row(run))
    );
}

#[when("45 is selected from the menu")]
fn reuse_45_degree_twist(world: &mut ApplicationWorld) {
    click_orientation_option(world, "45");
}

#[then("the Twist field displays 45")]
fn shows_recent_twist(world: &mut ApplicationWorld) {
    chosen_sources(world);
    assert_eq!(world.state().sweep_preview.unwrap().request.twist, "45");
    assert!(super::super::world::squeezed(&world.window_text("Sweep")).contains("Twist45"));
}

#[when("the edit is cancelled")]
fn cancel_twist_draft(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    click_panel(world, "Cancel");
    assert_eq!(world.state().document_generation, generation);
    assert!(world.state().sweep_preview.is_none());
}

#[then("the existing Sweep1 solid remains")]
fn accepted_twist_survives_cancel(world: &mut ApplicationWorld) {
    accepted_sweep(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(
        browser.sweep_edits[&browser.bodies[0].feature].twist,
        "45 deg"
    );
    assert!(!world.app().window.scene_triangles.is_empty());
}
