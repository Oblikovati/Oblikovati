// SPDX-License-Identifier: GPL-2.0-only
//! Native distance-and-angle Move with retained mode and history.
use super::{ApplicationWorld, diagnostics, move_precise};
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_viewport::ViewportPhysicalKey;

fn select_retained_polar_mode(world: &mut ApplicationWorld) {
    move_precise::select_ellipse(world);
    let mode = world.run_in_window("Precise Input", "Mode").center();
    world.click(diagnostics::pointer_pixels(mode));
    world.click_text("Distance and X axis angle");
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    move_precise::open_move(world);
    world.sketch_click(move_precise::EDGE);
    let text = world.window_text("Precise Input");
    assert!(
        text.contains("Distance") && text.contains("Angle"),
        "{text:?}"
    );
}

fn enter_polar_move(world: &mut ApplicationWorld) {
    world.sketch_click(move_precise::CENTER);
    world.click_precise_field("Distance");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("20");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
}

fn expect_polar_displacement(world: &mut ApplicationWorld, original: [f64; 2]) {
    let geometry = world.edited_geometry();
    let moved = geometry
        .point(geometry.ellipses()[0].center())
        .unwrap()
        .position;
    assert!(
        (moved[0] - original[0] - 2.).abs() < 1e-8 && (moved[1] - original[1]).abs() < 1e-8,
        "20 mm at 0 degrees: {original:?} -> {moved:?}"
    );
    let old_pixel = world.sketch_pixel_of(original);
    let new_pixel = world.sketch_pixel_of(moved);
    let edge = [
        move_precise::EDGE[0] + new_pixel[0] - old_pixel[0],
        move_precise::EDGE[1] + new_pixel[1] - old_pixel[1],
    ];
    world.expect_segment_drawn(edge, edge);
}

#[test]
fn polar_move_retains_mode_and_undoes_the_exact_displacement() {
    let mut world = ApplicationWorld::new();
    move_precise::editable_ellipse(&mut world);
    let before = world.edited_geometry();
    let original = before
        .point(before.ellipses()[0].center())
        .unwrap()
        .position;
    select_retained_polar_mode(&mut world);
    enter_polar_move(&mut world);
    expect_polar_displacement(&mut world, original);
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    world.undo();
    assert!(world.edited_geometry().same_recipe(&before));
    world.expect_segment_drawn(move_precise::EDGE, move_precise::EDGE);
    world.close();
}
