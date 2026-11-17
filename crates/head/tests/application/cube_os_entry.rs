// SPDX-License-Identifier: GPL-2.0-only
//! Native input for the beginning of the parameter-driven cube journey (#3857).
use super::{ApplicationWorld, diagnostics, x11_input};

fn click_label(world: &mut ApplicationWorld, label: &str) {
    let center = super::quick_access_icons::standard_icon_center(label)
        .unwrap_or_else(|| world.run(label).center());
    let point = diagnostics::pointer_pixels(center);
    x11_input::click(point);
    world.pump(Vec::new());
}

#[test]
fn native_mouse_creates_part_and_draws_rectangle() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    for label in ["New", "Metric", "Standard metric part", "Create"] {
        click_label(&mut world, label);
    }
    assert_eq!(world.state().documents.len(), 1);
    click_label(&mut world, "Start 2D Sketch");
    click_label(&mut world, "XY Plane");
    let edit = world
        .app()
        .owner
        .active_sketch_edit()
        .expect("expected XY sketch edit");
    assert_eq!(edit.plane, oblikovati_model::sketch::SketchPlane::xy());
    click_label(&mut world, "Rectangle");
    for corner in [[0., 0.], [1., 1.]] {
        let point = world.sketch_pixel_of(corner);
        x11_input::click(point);
        world.pump(Vec::new());
    }
    assert_eq!(world.edited_geometry().lines().len(), 4);
    click_label(&mut world, "Finish Sketch");
    assert!(world.app().owner.active_sketch_edit().is_none());
    assert_eq!(world.state().browser.unwrap().sketches.len(), 1);
    world.close();
}
