// SPDX-License-Identifier: GPL-2.0-only
//! Native one-body Extrude Join, including its real replacement preview and history.
use super::super::{capture_framebuffer, diagnostics, solid_visibility};
use super::*;

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-base-circle.opd";
const BASE_TOP: f32 = 0.8;
const BOSS_TOP: f32 = 1.6;

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-base-circle.opd` is open with its diameter-3 mm circle sketch on the top face"
)]
fn published_circle_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join(FIXTURE);
    open_circle_part(world, &std::fs::read(source).unwrap());
}

#[given("a diameter-3 circular sketch on the top face of the existing solid")]
fn pinned_circle_part(world: &mut ApplicationWorld) {
    open_circle_part(
        world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-base-circle.opd"
        ),
    );
}

fn open_circle_part(world: &mut ApplicationWorld, package: &[u8]) {
    let path = world.scratch_folder().join("extrusion-base-circle.opd");
    std::fs::write(&path, package).unwrap();
    world.open_fixture(&path);
    assert_original_shape(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.12 },
    )
    .unwrap();
    world.pump(Vec::new());
}

#[when("Extrude uses Default direction, Distance A 8 mm, zero taper and Join")]
fn preview_joined_boss(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    let original = world.state().browser.unwrap().bodies[0].key.clone();
    world.click_text("Sketch1");
    world.click_text("Extrude");
    super::replace_extrusion_distance(world, "8");
    click_panel(world, "NewSolid");
    world.click_text("Join");
    world.pump(Vec::new());
    let state = world.state();
    let draft = state.extrude_draft_preview.expect("certified Join preview");
    assert_eq!(
        draft.operation,
        Some(oblikovati_model::feature::sketch_prism::PrismBoolean::Join)
    );
    assert_eq!(
        draft.direction,
        oblikovati_gui::GuiExtrudeDirection::Positive
    );
    assert_eq!(state.document_generation, generation);
    assert_eq!(state.browser.unwrap().bodies[0].key, original);
    assert!(world.run_is_enabled("OK"));
    assert_circular_cap(world);
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_replacement_preview_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
}

#[then("a cylindrical boss and Extrusion2 are visible")]
fn joined_result(world: &mut ApplicationWorld) {
    world.expect_text("Extrusion2");
    assert!(diagnostics::window_geometry("Extrude").is_none());
    let browser = world.state().browser.unwrap();
    let joined = browser
        .features
        .iter()
        .find(|row| row.name == "Extrusion2")
        .unwrap();
    assert_eq!(browser.bodies[0].feature, joined.id);
    assert_eq!(browser.bodies[0].faces, 8);
    assert!(browser.sketches.iter().any(|sketch| {
        sketch.name == "Sketch1" && !sketch.visible && sketch.consumed_by == Some(joined.id)
    }));
    assert_circular_cap(world);
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_solid_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
}

#[then("the browser still shows Solid Bodies(1)")]
fn still_one_body(world: &mut ApplicationWorld) {
    world.expect_text("Solid Bodies");
    assert_eq!(world.state().browser.unwrap().bodies.len(), 1);
}

#[then("the circular sketch returns and the boss disappears")]
#[then("Sketch2 returns and the boss disappears")]
fn assert_original_shape(world: &mut ApplicationWorld) {
    still_one_body(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies[0].faces, 6);
    assert!(browser.features.iter().all(|row| row.name != "Extrusion2"));
    // The published native package calls the second sketch Sketch1.
    assert!(browser.sketches.iter().any(|sketch| {
        sketch.name == "Sketch1" && sketch.visible && sketch.consumed_by.is_none()
    }));
    let top = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions.map(|point| point[2]))
        .fold(f32::NEG_INFINITY, f32::max);
    assert!((top - BASE_TOP).abs() < 1e-5, "base cap is at {top} cm");
}

fn click_panel(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run_in_window("Extrude", label).center());
    world.click(point);
}

fn assert_circular_cap(world: &mut ApplicationWorld) {
    let mut min = [f32::INFINITY; 2];
    let mut max = [f32::NEG_INFINITY; 2];
    for point in world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .filter(|point| (point[2] - BOSS_TOP).abs() < 1e-5)
    {
        for axis in 0..2 {
            min[axis] = min[axis].min(point[axis]);
            max[axis] = max[axis].max(point[axis]);
        }
    }
    for axis in 0..2 {
        let span = max[axis] - min[axis];
        assert!(
            (span - 0.3).abs() < 0.01,
            "boss cap {axis} span is {span} cm at {BOSS_TOP} cm"
        );
    }
}
