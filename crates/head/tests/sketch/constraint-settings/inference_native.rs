// SPDX-License-Identifier: GPL-2.0-only
//! Compare inferred horizontal constraints using repeatable native clicks (#4158).
use super::super::ApplicationWorld;
use oblikovati_model::sketch::PlanarConstraint;

const TEST_CLICKS: [[i32; 2]; 2] = [[620, 300], [840, 302]];

#[test]
fn inference_toggle_changes_the_same_native_line_gesture() {
    let mut world = ApplicationWorld::new();
    world.start_sketch_on("XY Plane");
    set_inference(&mut world, false);
    draw_near_horizontal_line(&mut world);
    assert_eq!(horizontal_constraints(&mut world), 0);
    super::super::x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert!(world.edited_geometry().lines().is_empty());
    set_inference(&mut world, true);
    draw_near_horizontal_line(&mut world);
    assert_eq!(horizontal_constraints(&mut world), 1);
    world.close();
}

fn set_inference(world: &mut ApplicationWorld, enabled: bool) {
    world.open_application_options();
    world.click_text("Constraint Settings");
    world.click_text("Inference");
    if world.checked("Infer constraints") != enabled {
        world.click_text("Infer constraints");
    }
    if enabled && !world.checked("Persist constraints") {
        world.click_text("Persist constraints");
    }
    let point = world.run_in_window("Constraint Settings", "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(point));
    world.close_application_options();
    world.click_text("Sketch");
    world.pump(Vec::new());
}

fn draw_near_horizontal_line(world: &mut ApplicationWorld) {
    super::super::x11_input::focus_window();
    let line = super::super::diagnostics::pointer_pixels(world.run("Line").center());
    super::super::x11_input::click(line);
    world.pump(Vec::new());
    world.expect_status("Select start of line");
    for point in TEST_CLICKS {
        super::super::x11_input::click(point);
        world.pump(Vec::new());
    }
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    assert_eq!(world.edited_geometry().lines().len(), 1);
}

fn horizontal_constraints(world: &mut ApplicationWorld) -> usize {
    world
        .edited_geometry()
        .constraints()
        .iter()
        .filter(|item| matches!(item.relation, PlanarConstraint::LineHorizontal { .. }))
        .count()
}
