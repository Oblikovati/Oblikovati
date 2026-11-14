// SPDX-License-Identifier: GPL-2.0-only
//! Delete a coincidence, move one segment, and replay each edit.
use super::*;
use cucumber::{then, when};
use oblikovati_model::sketch::{Sketch2D, SketchPlane};

fn joined(world: &ApplicationWorld) -> &Sketch2D {
    &world
        .observed_sketch
        .as_ref()
        .expect("expected joined sketch")
        .geometry
}
fn endpoints(geometry: &Sketch2D, line: usize) -> [[f64; 2]; 2] {
    geometry.lines()[line]
        .endpoints
        .map(|id| geometry.point(id).unwrap().position)
}
fn pixel(world: &mut ApplicationWorld, point: [f64; 2]) -> [i32; 2] {
    crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), point).unwrap()
}
fn first_midpoint(world: &mut ApplicationWorld) -> [i32; 2] {
    let ends = endpoints(joined(world), 0);
    pixel(
        world,
        std::array::from_fn(|axis| (ends[0][axis] + ends[1][axis]) * 0.5),
    )
}
#[when("I right-click the first segment interior")]
fn open_line_menu(world: &mut ApplicationWorld) {
    let point = first_midpoint(world);
    world.right_click(point);
    world.pump(Vec::new());
}
#[then("the menu includes Delete Coincident Constraint")]
fn delete_is_offered(world: &mut ApplicationWorld) {
    world.expect_text("Delete Coincident Constraint");
}
#[when("I choose that command")]
fn delete_relation(world: &mut ApplicationWorld) {
    world.click_text("Delete Coincident Constraint");
}
#[then("the joint marker disappears and both lines retain their positions")]
#[then("the marker disappears without moving either line")]
fn deleted_without_movement(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.geometric_constraints().count(), 0);
    assert_eq!(geometry.points(), joined(world).points());
    assert_eq!(geometry.lines(), joined(world).lines());
    assert!(world.app().owner.sketch_constraint_marks().is_empty());
}
#[when("I move only the first segment using its start as base point")]
fn move_segment(world: &mut ApplicationWorld) {
    let middle = first_midpoint(world);
    let start = endpoints(joined(world), 0)[0];
    let base = pixel(world, start);
    world.click_text("Move");
    world.sketch_click(middle);
    world.sketch_click(base);
    let target = [base[0] + 120, base[1] + 80];
    world.hover(target);
    assert!(crate::HeadSketchWork::sketch_preview(&world.app().owner).is_some());
    world.sketch_click(target);
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
}
#[then("it separates from the second segment which remains in place")]
#[then("the first segment separates again")]
fn first_separates(world: &mut ApplicationWorld) {
    let current = world.edited_geometry();
    assert_ne!(endpoints(&current, 0), endpoints(joined(world), 0));
    assert_eq!(endpoints(&current, 1), endpoints(joined(world), 1));
    assert_ne!(endpoints(&current, 0)[1], endpoints(&current, 1)[0]);
}
#[then("no constraint-relaxation or conflict dialog appears")]
fn no_conflict(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Constraint Relaxation").is_none());
    assert!(diagnostics::window_geometry("Create Constraint").is_none());
}
#[when("I undo the movement")]
#[when("I undo the deletion")]
fn undo(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("the endpoints overlap visually without the joint marker")]
fn overlap_without_marker(world: &mut ApplicationWorld) {
    deleted_without_movement(world);
}
#[then("the joint marker returns")]
fn marker_returns(world: &mut ApplicationWorld) {
    let current = world.edited_geometry();
    assert!(current.same_recipe(joined(world)));
    assert!(
        world
            .app()
            .owner
            .sketch_constraint_marks()
            .iter()
            .any(|mark| mark.text == "C")
    );
}
#[when("I redo the deletion")]
#[when("I redo the movement")]
fn redo(world: &mut ApplicationWorld) {
    world.redo();
}
