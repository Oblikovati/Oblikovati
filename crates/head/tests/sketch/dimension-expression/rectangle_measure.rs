// SPDX-License-Identifier: GPL-2.0-only
//! Recover an invalid height expression and measure the rectangle's opposite sides.
use super::super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::ParameterRenameArgs};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("the earlier d50 fixture evaluates to 10")]
fn rectangle_with_ten_height(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    super::super::rectangle_two_point::typed_rectangle(world, "6", "10");
    let height = world
        .state()
        .parameters
        .into_iter()
        .find(|row| row.value.starts_with("10 "))
        .expect("expected the 10 mm height parameter");
    Client::new(world.app().owner.document_host())
        .parameters()
        .rename(&ParameterRenameArgs {
            name: height.name,
            new_name: "d50".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("10");
    world.observed_counts = vec![model_height(world)];
}
fn model_height(world: &mut ApplicationWorld) -> usize {
    let geometry = world.edited_geometry();
    let mut heights = geometry.points().iter().map(|point| point.position[1]);
    let first = heights.next().expect("expected rectangle points");
    let (min, max) = heights.fold((first, first), |(min, max), y| (min.min(y), max.max(y)));
    ((max - min) * 1_000_000.).round() as usize
}
#[when("the user enter incomplete 2* and attempt acceptance")]
fn reject_incomplete(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    world.click(world.exact_run_center("10"));
    world.expect_text("d50");
    world.type_text("2*");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the expression-evaluation warning appears")]
fn warning_shown(world: &mut ApplicationWorld) {
    world.expect_text("The expression cannot be evaluated.");
}
#[when("the user acknowledge it and complete 2*6 and accept")]
fn recover_expression(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.key(ViewportPhysicalKey::End, false);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the annotation reads 12 and the geometry changes")]
fn changed_height(world: &mut ApplicationWorld) {
    world.expect_annotation("12");
    assert!(model_height(world) > world.observed_counts[0]);
}
#[when("the user exit General Dimension and measure its top and bottom curves")]
fn measure_opposite_sides(world: &mut ApplicationWorld) {
    world.escape();
    world.key(ViewportPhysicalKey::M, false);
    let geometry = world.edited_geometry();
    for (index, line) in geometry.lines().iter().enumerate() {
        let start = geometry.point(line.endpoints[0]).unwrap().position;
        let end = geometry.point(line.endpoints[1]).unwrap().position;
        if (start[1] - end[1]).abs() < 1e-9 {
            let point = world.line_point(index, 0.3);
            world.sketch_click(point);
        }
    }
}
#[then("Measure reports minimum separation 12.000 mm")]
fn measured_height(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Selection 2: Curve");
    world.expect_measure("Minimum distance 12.000 mm");
}
