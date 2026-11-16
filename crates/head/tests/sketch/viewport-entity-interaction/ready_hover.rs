// SPDX-License-Identifier: GPL-2.0-only
//! Ready-state hover feedback through native mouse input (#4043).
use super::super::super::ApplicationWorld;
use oblikovati_viewport::ViewportPhysicalKey;

#[test]
fn ready_hover_identifies_point_and_line_without_selecting_them() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::add_isolated_line(&mut world);
    world.remember_edited_sketch(Vec::new());
    super::super::super::x11_input::move_pointer([1100, 650]);
    world.pump(Vec::new());
    let baseline = super::super::super::capture_framebuffer(&mut world.app().window.native);
    move_to(&mut world, [-3., 1.]);
    world.expect_text("1. Point (Sketch1)");
    assert!(world.state().sketch_selection.entities.is_empty());
    world.save_capture("ready-point-hover-4043");
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(false)]);
    assert!(world.state().sketch_pick_cue.is_none());
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(true)]);
    move_to(&mut world, [0., 1.]);
    world.expect_text("1. Line (Sketch1)");
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    world.expect_text("1. Line (Sketch1)");
    assert_line_highlight(&mut world, &baseline);
    assert!(world.state().sketch_selection.entities.is_empty());
    world.save_capture("ready-line-hover-4043");
    super::super::super::x11_input::move_pointer([1100, 650]);
    world.pump(Vec::new());
    assert!(world.state().sketch_pick_cue.is_none());
    world.expect_edited_sketch_retained();
    move_to(&mut world, [0., 1.]);
    world.key(ViewportPhysicalKey::D, false);
    assert!(world.state().sketch_pick_cue.is_none());
    let after_tool = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let sample = world.sketch_pixel_of([0.75, 1.]);
    assert!(super::same_neighborhood(&baseline, &after_tool, sample));
    world.escape();
    let midpoint = world.sketch_pixel_of([0., 1.]);
    super::native_click(&mut world, midpoint);
    let selected = world.state().sketch_selection.entities;
    assert_eq!(selected.len(), 1);
    move_to(&mut world, [-3., 1.]);
    world.expect_text("1. Point (Sketch1)");
    assert_eq!(world.state().sketch_selection.entities, selected);
    world.expect_edited_sketch_retained();
    world.close();
}

fn move_to(world: &mut ApplicationWorld, point: [f64; 2]) {
    let pixel = world.sketch_pixel_of(point);
    super::super::super::x11_input::move_pointer(pixel);
    world.pump(Vec::new());
}

fn assert_line_highlight(
    world: &mut ApplicationWorld,
    before: &oblikovati_viewport::ViewportCapture,
) {
    use oblikovati_api::types::{TOKEN_SKETCH_CANDIDATE, TOKEN_SKETCH_GEOMETRY};
    let point = world.sketch_pixel_of([0.75, 1.]);
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let normal = &palette[&TOKEN_SKETCH_GEOMETRY];
    let candidate = &palette[&TOKEN_SKETCH_CANDIDATE];
    let colors = [
        [normal.r, normal.g, normal.b],
        [candidate.r, candidate.g, candidate.b],
    ];
    let after = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(super::selection_color_changed(
        before, &after, point, colors
    ));
}

#[test]
fn ready_hover_is_retired_when_the_sketch_is_undone() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::add_isolated_line(&mut world);
    move_to(&mut world, [0., 1.]);
    world.expect_text("1. Line (Sketch1)");
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    world.pump(Vec::new());
    assert!(world.state().sketch_pick_cue.is_none());
    assert!(world.edited_geometry().lines().is_empty());
    world.close();
}
