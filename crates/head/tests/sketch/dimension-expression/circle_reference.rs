// SPDX-License-Identifier: GPL-2.0-only
//! A line dimension references an earlier circle diameter parameter.
use super::*;
use oblikovati_api::wire::AddDimensionArgs;

#[given("the earlier d1 dimension editor is open and d0 identifies the circle diameter parameter")]
fn circle_and_line(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.click_text("Circle");
    world.sketch_click([820, 520]);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.escape();
    world.select_line_tool();
    for point in [[900, 500], [1040, 500]] {
        world.sketch_click(point);
    }
    world.escape();
    add_line_dimension(world);
    world.open_dimension_label("8");
    world.expect_text("d1");
}
fn add_line_dimension(world: &mut ApplicationWorld) {
    let endpoints = world.edited_geometry().lines()[0].endpoints.map(|id| id.0);
    let text_point = world.sketch_point_of([970, 560]);
    let client = Client::new(world.app().owner.document_host());
    let added = client
        .dimension()
        .add(&AddDimensionArgs {
            kind: "distance".into(),
            entities: Some(endpoints.to_vec()),
            orientation: "aligned".into(),
            expression: "8 mm".into(),
            text_point: Some(text_point.to_vec()),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(added.parameter, "d1");
    world.pump(Vec::new());
    world.expect_annotation("6");
    world.expect_annotation("8");
}
#[when("the user type d then 0 and press Enter")]
fn enter_circle_reference(world: &mut ApplicationWorld) {
    world.type_text("d");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the expression is accepted")]
fn accepted_circle_reference(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    world.expect_annotation("fx:6");
}
#[when("the user reopen d1")]
fn reopen_line_dimension(world: &mut ApplicationWorld) {
    world.open_dimension_label("fx:6");
}
#[then("its retained expression is d0")]
fn retained_circle_reference(world: &mut ApplicationWorld) {
    assert!(world.window_text("Edit Dimension").contains("d0"));
    world.escape();
    world.open_dimension_label("6");
    world.type_text("10");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:10");
    let geometry = world.edited_geometry();
    let [first, second] = geometry.lines()[0].endpoints;
    let first = geometry.point(first).unwrap().position;
    let second = geometry.point(second).unwrap().position;
    let length = (second[0] - first[0]).hypot(second[1] - first[1]);
    assert!((length - 1.).abs() < 1e-6, "line length {length} cm");
    assert!((geometry.circles()[0].radius - 0.5).abs() < 1e-6);
}
