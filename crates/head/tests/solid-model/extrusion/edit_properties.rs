// SPDX-License-Identifier: GPL-2.0-only
use super::super::diagnostics;
use super::acceptance::{feature_count, feature_distance, open_extrusion_editor};
use super::*;

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-rect-20mm.opd";

#[given(
    "`fixtures/cad/extrusion/extrusion-rect-20mm.opd` is open with Extrusion1 and one solid body"
)]
fn open_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join(FIXTURE);
    open_copy(world, &std::fs::read(source).unwrap(), 20.);
}

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-rect-20mm.opd` is open with Edit Feature active for Extrusion1"
)]
pub(super) fn start_published_edit(world: &mut ApplicationWorld) {
    open_part(world);
    open_extrusion_editor(world);
}

#[given("Edit Feature is open for the captured solid extrusion")]
pub(super) fn start_pinned_edit(world: &mut ApplicationWorld) {
    open_copy(
        world,
        include_bytes!("../../../../../test-utilities/fixtures/extrusion/extrusion-rect-8mm.opd"),
        8.,
    );
    open_extrusion_editor(world);
}

fn open_copy(world: &mut ApplicationWorld, package: &[u8], depth: f64) {
    let path = world.scratch_folder().join("extrusion-properties.opd");
    std::fs::write(&path, package).unwrap();
    world.open_fixture(&path);
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), depth);
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    world.observed_rows = Some(
        browser
            .extrusion_edits
            .values()
            .next()
            .unwrap()
            .distance
            .clone(),
    );
}

#[when("the user double-clicks Extrusion1 in the model browser")]
fn double_click_feature(world: &mut ApplicationWorld) {
    world.double_click(diagnostics::pointer_pixels(
        world.run("Extrusion1").center(),
    ));
    world.pump(Vec::new());
}

#[then("the Extrusion panel restores 1 Profile from 1 Sketch Plane")]
fn input_restored(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "1Profile");
    world.run_in_window("Extrude", "1SketchPlane");
    world.run_in_window("Extrude", "Profile");
}

#[then("Distance A is 20 mm and a body name is displayed")]
fn output_restored(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "20mm");
    let state = world.state();
    let body = &state.browser.unwrap().bodies[0];
    world.run_in_window("Extrude", "BodyName");
    world.run_in_window("Extrude", &body.name);
}

#[then("Match Shape is visible and disabled")]
fn fixed_shape(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "MatchShape");
    assert!(world.checked("Match Shape"));
    assert!(!world.run_is_enabled("Match Shape"));
}

#[then("Taper A is 0 degrees with Match Shape checked and disabled")]
fn fixed_shape_and_taper(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "TaperA");
    world.run_in_window("Extrude", "0deg");
    fixed_shape(world);
}

#[then("a 20 mm viewport preview appears")]
fn depth_preview(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(super::direction::preview_depth(world), [0., 2.]);
}

#[when("the user cancels the edit")]
fn cancel_edit(world: &mut ApplicationWorld) {
    for window in ["Shell", "Split"] {
        if diagnostics::window_geometry(window).is_some() {
            let center = world.run_in_window(window, "Cancel").center();
            world.click(diagnostics::pointer_pixels(center));
            world.pump(Vec::new());
            world.pump(Vec::new());
            return;
        }
    }
    super::cancel_extrusion(world);
}

#[then("the panel closes and the committed solid remains unchanged")]
fn solid_is_unchanged(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Extrude").is_none());
    assert!(!world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 20.);
    assert_eq!(super::direction::preview_depth(world), [0., 2.]);
}
