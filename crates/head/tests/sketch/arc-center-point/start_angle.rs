// SPDX-License-Identifier: GPL-2.0-only
//! Typed start directions stay editable across history and document storage.
use super::*;

fn create_directed_arc(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    draw_directed_arc(world, "45");
    world.expect_annotation("45.00");
}
fn draw_directed_arc(world: &mut ApplicationWorld, direction: &str) {
    select_tool(world);
    world.sketch_click(CENTER);
    world.type_text("4");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text(direction);
    world.key(ViewportPhysicalKey::Tab, false);
    world.sketch_click(START);
    world.type_text("90");
    world.key(ViewportPhysicalKey::Tab, false);
    world.sketch_click(ABOVE);
    world.escape();
}
fn assert_direction(world: &mut ApplicationWorld, degrees: f64) {
    let geometry = world.edited_geometry();
    let arc = geometry.arcs()[0];
    let center = geometry.point(arc.center).unwrap().position;
    let start = geometry.point(arc.start).unwrap().position;
    let end = geometry.point(arc.end).unwrap().position;
    assert_radius_and_sweep(center, start, end, arc.counter_clockwise);
    let angle = (start[1] - center[1])
        .atan2(start[0] - center[0])
        .to_degrees()
        .rem_euclid(360.);
    assert!(
        (angle - degrees).abs() < 0.001,
        "start angle {angle}, expected {degrees}"
    );
}
fn assert_radius_and_sweep(center: [f64; 2], start: [f64; 2], end: [f64; 2], ccw: bool) {
    let radial = |point: [f64; 2]| [point[0] - center[0], point[1] - center[1]];
    let [start, end] = [radial(start), radial(end)];
    let radius = start[0].hypot(start[1]);
    let from = start[1].atan2(start[0]);
    let to = end[1].atan2(end[0]);
    let sweep = if ccw { to - from } else { from - to }.rem_euclid(std::f64::consts::TAU);
    assert!((radius - 0.4).abs() < 1e-6, "radius {radius} cm");
    assert!(
        (sweep - std::f64::consts::FRAC_PI_2).abs() < 1e-6,
        "sweep {sweep} rad"
    );
}
fn edit_direction(world: &mut ApplicationWorld, old: &str, new: &str) {
    world.open_dimension_label(old);
    world.type_text(new);
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation(&format!("{new}.00"));
}
fn reopen_saved_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
    world.press_save_shortcut();
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected the saved sketch row");
    let point = diagnostics::pointer_pixels(row.center());
    world.double_click(point);
    world.pump(Vec::new());
}
#[test]
fn typed_start_direction_drives_history_and_reopens() {
    let mut world = ApplicationWorld::new();
    create_directed_arc(&mut world);
    assert_direction(&mut world, 45.);
    edit_direction(&mut world, "45.00", "60");
    assert_direction(&mut world, 60.);
    world.undo();
    assert_direction(&mut world, 45.);
    world.redo();
    assert_direction(&mut world, 60.);
    reopen_saved_sketch(&mut world);
    assert_direction(&mut world, 60.);
    edit_direction(&mut world, "60.00", "30");
    assert_direction(&mut world, 30.);
    world.close();
}

fn set_turn(world: &mut ApplicationWorld, value: &str, add: bool) {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    let client = Client::new(world.app().owner.document_host());
    let argument = ParameterSetArgs {
        name: "turn".into(),
        expression: value.into(),
        ..Default::default()
    };
    if add {
        client.parameters().add(&argument).unwrap();
    } else {
        client.parameters().set(&argument).unwrap();
    }
    world.pump(Vec::new());
}
#[test]
fn formula_drives_start_direction_through_history_and_storage() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    set_turn(&mut world, "22.5 deg", true);
    draw_directed_arc(&mut world, "turn * 2");
    world.expect_annotation("fx:45.00");
    assert_direction(&mut world, 45.);
    set_turn(&mut world, "30 deg", false);
    world.expect_annotation("fx:60.00");
    assert_direction(&mut world, 60.);
    world.undo();
    assert_direction(&mut world, 45.);
    world.redo();
    assert_direction(&mut world, 60.);
    reopen_saved_sketch(&mut world);
    world.expect_annotation("fx:60.00");
    set_turn(&mut world, "15 deg", false);
    world.expect_annotation("fx:30.00");
    assert_direction(&mut world, 30.);
    world.close();
}
