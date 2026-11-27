// SPDX-License-Identifier: GPL-2.0-only
//! Two-point coincidence through the active sketch tool and history.
use super::*;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{Sketch2D, SketchPlane};
use oblikovati_viewport::ViewportPhysicalKey;

const FIRST: [i32; 2] = [560, 400];
const SECOND: [i32; 2] = [1000, 550];

#[given("two separate standalone points")]
#[given("two separate standalone points at the recorded upper-left and lower-right positions")]
fn separate_points(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.select_point_tool();
    world.sketch_click(FIRST);
    world.sketch_click(SECOND);
    world.escape();
    world.key(ViewportPhysicalKey::F8, false);
    world.record_initial_sketch();
}
#[when("the user invoke Coincident and select the upper-left point")]
#[when("the user invokes Coincident and selects the upper-left point")]
fn first_pick(world: &mut ApplicationWorld) {
    super::coincident_segments::invoke_coincident(world);
    world.sketch_click(FIRST);
}
#[then("the prompt requests a second curve or point")]
fn second_prompt(world: &mut ApplicationWorld) {
    world.expect_status("Coincident: Select second curve or point");
    assert_eq!(world.state().sketch_selection.entities.len(), 1);
}
#[when("the user select the lower-right point")]
#[when("the user selects the lower-right point")]
fn second_pick(world: &mut ApplicationWorld) {
    world.sketch_click(SECOND);
}
#[then("it moves onto the first point and a coincidence marker appears there")]
#[then("the points share a visible position and a coincidence marker appears there")]
#[then("their shared visible position returns")]
fn points_joined(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.geometric_constraints().count(), 1);
    let pixels = point_pixels(world, &geometry);
    assert!((pixels[0][0] - pixels[1][0]).abs() <= 1 && (pixels[0][1] - pixels[1][1]).abs() <= 1);
    world.expect_annotation("C");
}
fn point_pixels(world: &mut ApplicationWorld, geometry: &Sketch2D) -> [[i32; 2]; 2] {
    let ids = world.observed_sketch.as_ref().unwrap().geometry.points();
    let points = [ids[ids.len() - 2].id, ids[ids.len() - 1].id];
    points.map(|id| {
        crate::HeadWindow::sketch_pixel(
            &world.app().window,
            SketchPlane::xy(),
            geometry.point(id).unwrap().position,
        )
        .unwrap()
    })
}
#[then("the prompt returns to the first curve or point")]
fn first_prompt(world: &mut ApplicationWorld) {
    world.expect_status("Coincident: Select first curve or point");
    assert!(world.state().sketch_selection.entities.is_empty());
    assert!(!world.state().sketch_at_ready);
}
#[then("the placement remains")]
fn placement_remains(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_at_ready);
    points_joined(world);
}
#[when("the user undo the relation")]
#[when("the user undoes the relation")]
fn undo_relation(world: &mut ApplicationWorld) {
    world.undo();
}
#[when("the user redo the relation")]
#[when("the user redoes the relation")]
fn redo_relation(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("both points appear separately again")]
fn points_separate(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.geometric_constraints().count(), 0);
    let pixels = point_pixels(world, &geometry);
    assert_ne!(pixels[0], pixels[1]);
    world.expect_no_annotation("C");
}
