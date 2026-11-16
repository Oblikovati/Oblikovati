// SPDX-License-Identifier: GPL-2.0-only
//! Selection and context-menu interaction on a rectangle edge (#4033).
use super::super::super::ApplicationWorld;

#[test]
fn rectangle_selection_survives_context_menu_and_escape() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::super::rectangle_two_point::typed_rectangle(&mut world, "80", "40");
    let edge = world.edited_geometry().lines()[0].id;
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let baseline = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let position = world.line_point(0, 0.25);
    super::native_click(&mut world, position);
    let selected = vec![(edit.sketch, edge)];
    assert_eq!(world.state().sketch_selection.entities, selected);
    open_rectangle_menu(&mut world);
    assert_eq!(world.state().sketch_selection.entities, selected);
    assert_selected_edge_color(&mut world, &baseline);
    dismiss_rectangle_menu(&mut world);
    assert_eq!(world.state().sketch_selection.entities, selected);
    clear_rectangle_selection(&mut world, &baseline);
    world.close();
}

fn clear_rectangle_selection(
    world: &mut ApplicationWorld,
    baseline: &oblikovati_viewport::ViewportCapture,
) {
    super::native_click(world, [1100, 650]);
    assert!(world.state().sketch_selection.entities.is_empty());
    let sample = world.line_point(0, 0.8);
    let cleared = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(super::same_neighborhood(baseline, &cleared, sample));
}

fn open_rectangle_menu(world: &mut ApplicationWorld) {
    super::super::super::x11_input::right_click([1100, 650]);
    world.pump(Vec::new());
    world.expect_text("Repeat Rectangle");
    world.expect_text("Region Properties");
    world.save_capture("rectangle-edge-context-4033");
}

fn dismiss_rectangle_menu(world: &mut ApplicationWorld) {
    super::super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert!(!world.text().contains("Region Properties"));
}

fn assert_selected_edge_color(
    world: &mut ApplicationWorld,
    baseline: &oblikovati_viewport::ViewportCapture,
) {
    use oblikovati_api::types::{TOKEN_SKETCH_GEOMETRY, TOKEN_SKETCH_SELECTED};
    let sample = world.line_point(0, 0.8);
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let normal = &palette[&TOKEN_SKETCH_GEOMETRY];
    let selected = &palette[&TOKEN_SKETCH_SELECTED];
    let colors = [
        [normal.r, normal.g, normal.b],
        [selected.r, selected.g, selected.b],
    ];
    let after = super::super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(super::selection_color_changed(
        baseline, &after, sample, colors
    ));
}
