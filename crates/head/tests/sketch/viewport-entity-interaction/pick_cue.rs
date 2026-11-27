// SPDX-License-Identifier: GPL-2.0-only
//! Native point-on-line pick cue and clearance (#4042).
use super::super::super::ApplicationWorld;
use oblikovati_model::sketch::SketchEntityId;

#[test]
fn overlap_pick_cue_names_the_point_and_clears_when_pointer_moves() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::add_isolated_line(&mut world);
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let point = super::add_overlap_point(&mut world);
    check_point_cue_and_clearance(&mut world, edit, point);
    check_line_cue_and_undo(&mut world);
    world.close();
}

fn check_point_cue_and_clearance(
    world: &mut ApplicationWorld,
    edit: crate::HeadSketchEdit,
    point: SketchEntityId,
) {
    let pixel = world.sketch_pixel_of([0., 1.]);
    super::native_click(world, pixel);
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, point)]
    );
    world.expect_text("1. Point (Sketch1)");
    world.save_capture("sketch-point-pick-cue-4042");
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(false)]);
    assert!(world.state().sketch_pick_cue.is_none());
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(true)]);
    super::native_click(world, pixel);
    world.expect_text("1. Point (Sketch1)");
    super::super::super::x11_input::move_pointer([1100, 650]);
    world.pump(Vec::new());
    assert!(!world.text().contains("1.Point(Sketch1)"));
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, point)]
    );
}

fn check_line_cue_and_undo(world: &mut ApplicationWorld) {
    let line = world.line_point(0, 0.25);
    super::native_click(world, line);
    world.expect_text("1. Line (Sketch1)");
    check_camera_motion_hides_cue(world);
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    world.pump(Vec::new());
    assert!(world.state().sketch_pick_cue.is_none());
}

fn check_camera_motion_hides_cue(world: &mut ApplicationWorld) {
    use crate::HeadWindow;
    world
        .app()
        .window
        .native
        .navigate_view(oblikovati_viewport::ViewportNavigation::Pan([120, 0]))
        .unwrap();
    world.pump(Vec::new());
    assert!(!world.shows("1. Line (Sketch1)"));
}
