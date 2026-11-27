// SPDX-License-Identifier: GPL-2.0-only
//! Drive both published multi-profile scenarios through native viewport clicks.
use super::super::distance_editor::dimension_pick::{
    open_dimension_package, open_rectangle_preview,
};
use super::super::viewport_interaction::click_at;
use super::{ApplicationWorld, apply_next::click_settled, click_in_extrude, feature_count};
use crate::HeadWindow;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportCamera;

const RECTANGLE: [f64; 3] = [1., 0.4, 0.];
const CIRCLE: [f64; 3] = [3., 0., 0.];

#[given(
    "the rectangular and circular loops in `fixtures/cad/extrusion/extrusion-dimension-pick.opd` are visible in an active Extrude command"
)]
fn open_rect_and_circle(world: &mut ApplicationWorld) {
    let fixture = super::super::super::scenario_root()
        .join("fixtures/cad/extrusion/extrusion-dimension-pick.opd");
    open_dimension_package(world, &std::fs::read(fixture).unwrap());
    open_rectangle_preview(world);
}

#[when(
    "the user chooses solid output and selects the interiors away from their dimension annotations"
)]
fn select_rect_and_circle(world: &mut ApplicationWorld) {
    click_in_extrude(world, "NewSolid");
    world.click_text("New Solid");
    world.pump(Vec::new());
    let before = world.state();
    pick(world, RECTANGLE);
    pick(world, CIRCLE);
    pick(world, CIRCLE);
    let after = world.state();
    assert_eq!(after.document_generation, before.document_generation);
    assert_eq!(after.undo_labels, before.undo_labels);
    assert_eq!(feature_count(world), 0);
    assert_eq!(
        after.extrude_draft_preview.unwrap().additional_profiles,
        [1]
    );
}

#[then("Profiles reports two selections and both regions have an extrusion preview")]
fn rect_and_circle_preview(world: &mut ApplicationWorld) {
    two_profiles_reported(world);
    assert_two_preview_solids(world, 0, 2.);
    world.save_capture("extrude-rectangle-and-circle-preview");
}

#[when("the user clears Profiles")]
fn clear_profiles(world: &mut ApplicationWorld) {
    click_settled(world, "Clear");
}

#[then("Profiles requests selection again and OK is disabled")]
fn waiting_after_clear(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "SelectProfile");
    assert!(!world.run_is_enabled("OK"));
    assert!(world.state().extrude_draft_preview.is_none());
    assert!(world.app().window.scene_triangles.is_empty());
}

#[when("the user selects only the rectangle interior again")]
fn pick_rectangle_again(world: &mut ApplicationWorld) {
    pick(world, RECTANGLE);
}

#[then("exactly one profile and its preview return")]
fn rectangle_only(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "1Profile");
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("repicked rectangle");
    assert_eq!(draft.profile, 0);
    assert!(draft.additional_profiles.is_empty());
    let triangles = &world.app().window.scene_triangles;
    assert!(!triangles.is_empty());
    assert!(
        triangles
            .iter()
            .flat_map(|triangle| triangle.positions)
            .all(|point| (-1.5..=1.5).contains(&point[0]))
    );
    assert!(world.run_is_enabled("OK"));
}

#[given(
    "a disposable copy of the native fixture `fixtures/cad/region-properties/two-rectangles.opd` is open"
)]
fn open_two_rectangles(world: &mut ApplicationWorld) {
    let source = super::super::super::scenario_root()
        .join("fixtures/cad/region-properties/two-rectangles.opd");
    let copy = world.scratch_folder().join("two-rectangles.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
}

#[given("the finished sketch has two separate visible closed regions")]
fn two_rectangles_visible(world: &mut ApplicationWorld) {
    let camera = ViewportCamera {
        eye: [1.5, 0.4, 20.],
        target: [1.5, 0.4, 0.],
        scale: 7.,
        ..Default::default()
    };
    world.app().window.set_view_camera(camera).unwrap();
    world.pump(Vec::new());
    let state = world.state();
    assert!(state.active_sketch.is_none());
    let browser = state.browser.unwrap();
    assert_eq!(browser.sketches.len(), 1);
    assert!(browser.sketches[0].visible);
    assert!(browser.bodies.is_empty());
}

#[given("Extrude is awaiting profile selection")]
fn extrude_is_awaiting_profiles(world: &mut ApplicationWorld) {
    world.click_text("Extrude");
    world.pump(Vec::new());
    if world.state().extrude_draft_preview.is_some() {
        clear_profiles(world);
    }
    waiting_after_clear(world);
}

#[when("the user clicks inside each region in turn without dragging")]
fn pick_two_rectangles(world: &mut ApplicationWorld) {
    pick(world, [2.2, -1.3, 0.]);
    let first = world
        .state()
        .extrude_draft_preview
        .expect("first picked rectangle");
    assert!(first.additional_profiles.is_empty());
    assert!(
        world
            .app()
            .window
            .scene_triangles
            .iter()
            .flat_map(|triangle| triangle.positions)
            .all(|point| point[1] < 0.)
    );
    pick(world, [2.2, 2.2, 0.]);
}

#[then("Profiles shows 2 Profiles")]
fn two_profiles_reported(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "2Profiles");
    let draft = world
        .state()
        .extrude_draft_preview
        .expect("two-profile preview");
    assert_eq!(draft.additional_profiles.len(), 1);
    assert_ne!(draft.profile, draft.additional_profiles[0]);
    assert!(world.run_is_enabled("OK"));
}

#[then("the extrusion preview includes both regions")]
fn two_rectangles_preview(world: &mut ApplicationWorld) {
    assert_two_preview_solids(world, 1, 0.);
    world.save_capture("extrude-two-rectangles-preview");
}

fn pick(world: &mut ApplicationWorld, point: [f64; 3]) {
    let pixel = world.app().window.native.view.model_pixel(point).unwrap();
    world.hover(pixel);
    click_at(world, pixel);
    world.pump(Vec::new());
}

fn assert_two_preview_solids(world: &mut ApplicationWorld, axis: usize, split: f32) {
    let mut caps = [false; 2];
    for triangle in &world.app().window.scene_triangles {
        let right = triangle.positions[0][axis] > split;
        assert!(
            triangle
                .positions
                .iter()
                .all(|point| (point[axis] > split) == right)
        );
        if triangle
            .positions
            .iter()
            .all(|point| (point[2] - 1.).abs() < 1e-5)
        {
            caps[usize::from(right)] = true;
        }
    }
    assert_eq!(caps, [true, true]);
}

#[test]
fn native_extrude_commits_multiple_bodies_in_one_feature_and_reopens() {
    let mut world = ApplicationWorld::new();
    super::super::new_metric_part(&mut world);
    open_dimension_package(
        &mut world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-dimension-pick.opd"
        ),
    );
    open_rectangle_preview(&mut world);
    select_rect_and_circle(&mut world);
    rect_and_circle_preview(&mut world);
    click_settled(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 1);
    let state = world.state();
    assert_eq!(state.undo_labels.len(), 1);
    let browser = state.browser.unwrap();
    assert_eq!(browser.bodies.len(), 2);
    let keys = browser
        .bodies
        .into_iter()
        .map(|body| body.key)
        .collect::<Vec<_>>();
    assert_ne!(keys[0], keys[1]);
    let directory = world.scratch_folder();
    super::super::acceptance::save_and_reopen(&mut world, &directory.join("grouped-extrusion.opd"));
    world.pump(Vec::new());
    let browser = world.state().browser.unwrap();
    assert_eq!(
        browser
            .bodies
            .into_iter()
            .map(|body| body.key)
            .collect::<Vec<_>>(),
        keys
    );
    assert_eq!(feature_count(&mut world), 1);
}
