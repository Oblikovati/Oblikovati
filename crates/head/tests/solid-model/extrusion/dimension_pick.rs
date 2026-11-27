// SPDX-License-Identifier: GPL-2.0-only
use super::super::direction::preview_depth;
use super::*;
use crate::HeadWindow;
use oblikovati_viewport::ViewportCamera;

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-dimension-pick.opd";

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-dimension-pick.opd` is open with a rectangle and separate 3 mm circle dimensioned as d2"
)]
fn open_dimension_fixture(world: &mut ApplicationWorld) {
    let source = super::super::super::scenario_root().join(FIXTURE);
    open_dimension_package(world, &std::fs::read(source).unwrap());
}

#[given("two disjoint closed loops and one rectangle selected for extrusion")]
fn pinned_dimension_fixture(world: &mut ApplicationWorld) {
    open_dimension_package(
        world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-dimension-pick.opd"
        ),
    );
    open_rectangle_preview(world);
}

pub(in super::super) fn open_dimension_package(world: &mut ApplicationWorld, bytes: &[u8]) {
    let copy = world.scratch_folder().join("extrusion-dimension-pick.opd");
    std::fs::write(&copy, bytes).unwrap();
    world.open_fixture(&copy);
    let camera = ViewportCamera {
        eye: [0.8, 0., 20.],
        target: [0.8, 0., 0.],
        scale: 6.,
        ..Default::default()
    };
    world.app().window.set_view_camera(camera).unwrap();
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 0);
    world.run("3mm");
}

#[given("the rectangular profile is selected in an active Extrude preview")]
pub(in super::super) fn open_rectangle_preview(world: &mut ApplicationWorld) {
    world.click_text("Sketch1");
    world.click_text("Extrude");
    world.pump(Vec::new());
    assert_eq!(preview_depth(world), [0., 1.]);
    assert_rectangle(world);
}

#[when("the user opens the Distance A editor menu and chooses Measure")]
fn choose_measure(world: &mut ApplicationWorld) {
    open_distance_menu(world);
    click_popup(world, "Measure");
    world.pump(Vec::new());
}

#[then("the status area asks the user to select a dimension to reference the parameter")]
fn measure_prompt(world: &mut ApplicationWorld) {
    world.expect_text("Select a dimension to reference the parameter.");
}

#[when("the circle dimension annotation is picked")]
fn click_circle_dimension(world: &mut ApplicationWorld) {
    click_position(world, world.run("3mm").center());
    world.pump(Vec::new());
}

#[then("Distance A references d2 instead of adding another profile")]
fn preview_uses_measured_parameter(world: &mut ApplicationWorld) {
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("measured preview");
    assert_eq!(draft.distance, "d2");
    assert_eq!(draft.profile, 0);
    world.run_in_window("Extrude", "d2");
    assert_eq!(preview_depth(world), [0., 0.3]);
    assert_rectangle(world);
}

#[then("a later pick produces invalid d2d2 and disabled OK")]
fn repeated_pick_validates_as_authored(world: &mut ApplicationWorld) {
    click_circle_dimension(world);
    let state = world.state();
    let validation = state
        .extrude_distance_validation
        .expect("checked expression");
    assert_eq!(validation.distance, "d2d2");
    assert!(validation.error.is_some());
    assert!(!state.extrude_preview_active);
    world.run_in_window("Extrude", "d2d2");
    assert!(!world.run_is_enabled("OK"));
}

#[then("no new extrusion is accepted")]
fn cancel_keeps_source_only(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Extrude").is_none());
    assert_eq!(feature_count(world), 0);
    let browser = world.state().browser.unwrap();
    assert!(browser.bodies.is_empty());
    assert_eq!(browser.sketches.len(), 1);
    world.run("3mm");
}

fn assert_rectangle(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "1Profile");
    let x: Vec<_> = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions.map(|point| point[0]))
        .collect();
    assert!(x.contains(&-1.5) && x.contains(&1.5));
    assert!(x.iter().all(|value| (-1.5..=1.5).contains(value)));
}
