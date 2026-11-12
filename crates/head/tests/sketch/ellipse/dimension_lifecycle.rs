// SPDX-License-Identifier: GPL-2.0-only
//! The recorded ellipse minor-radius dimension and its cancelled value edit.
use super::super::super::{ApplicationWorld, diagnostics};
use super::choose_ellipse;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client, types::DIM_CONSTRAINT_ELLIPSE_RADIUS, wire::AddDimensionArgs,
};
use oblikovati_viewport::ViewportPhysicalKey;

const CENTER: [f64; 2] = [-6.213, 15.221];
const MINOR: f64 = 2.9788523;
#[given(
    "the recorded ellipse has major-radius annotation 8 and minor radius approximately 2.979 mm"
)]
fn recorded_ellipse(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    choose_ellipse(world);
    world.open_precise_input();
    for point in [
        CENTER,
        [CENTER[0] + 8., CENTER[1]],
        [CENTER[0], CENTER[1] + MINOR],
    ] {
        world.precise_point(point);
    }
    world.escape();
    let geometry = world.edited_geometry();
    let ellipse = geometry.ellipses()[0];
    Client::new(world.app().owner.document_host())
        .dimension()
        .add(&AddDimensionArgs {
            kind: DIM_CONSTRAINT_ELLIPSE_RADIUS.0.to_string(),
            entities: Some(vec![ellipse.id().0]),
            expression: "8 mm".into(),
            text_point: Some(vec![CENTER[0], CENTER[1] - 12.]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("8");
}
#[when("General Dimension selects its curve and places the vertical center-to-axis dimension")]
fn place_minor(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    let geometry = world.edited_geometry();
    let ellipse = geometry.ellipses()[0];
    let center = geometry.point(ellipse.center()).unwrap().position;
    let hit = ellipse.evaluate(center, 0.7).unwrap().position;
    let hit = world.sketch_pixel_of(hit);
    world.sketch_click(hit);
    let label = world.sketch_pixel_of([center[0] + 1.2, center[1]]);
    world.sketch_click(label);
    world.pump(Vec::new());
}
#[then("the minor-radius editor opens with 2.9788523 mm selected")]
fn editor_shows_minor(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Edit Dimension").is_some());
    assert!(world.window_text("Edit Dimension").contains("2.9788523"));
    assert!(world.text_is_selected("2.9788523"));
}
#[when("4 is entered and Escape cancels the editor")]
fn cancel_value(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.escape();
}
#[then("the placed dimension remains annotated 2.979")]
fn annotation_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("2.979");
    assert!(world.state().dimension_tool_active);
}
#[when("its annotation is reopened")]
fn reopen(world: &mut ApplicationWorld) {
    world.click(world.exact_run_center("2.979"));
    world.pump(Vec::new());
}
#[then("the editor retains expression 2.9788523 mm")]
fn retained_expression(world: &mut ApplicationWorld) {
    assert!(world.window_text("Edit Dimension").contains("2.9788523"));
}
#[given("the reopened minor-radius expression is selected")]
fn reopened_minor(world: &mut ApplicationWorld) {
    recorded_ellipse(world);
    place_minor(world);
    editor_shows_minor(world);
    cancel_value(world);
    reopen(world);
    retained_expression(world);
    assert!(world.text_is_selected("2.9788523"));
}
#[when("4 is entered and Return accepts it")]
fn accept_minor(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the ellipse grows vertically and its vertical annotation becomes 4")]
fn minor_changed(world: &mut ApplicationWorld) {
    world.expect_annotation("4");
    let ellipse = world.edited_geometry().ellipses()[0];
    assert!((ellipse.radii()[1] - 0.4).abs() < 1e-8);
}
#[then("its horizontal annotation remains 8")]
fn major_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("8");
    let ellipse = world.edited_geometry().ellipses()[0];
    assert!((ellipse.radii()[0] - 0.8).abs() < 1e-8);
}
#[when("Escape exits General Dimension")]
fn exit_dimension(world: &mut ApplicationWorld) {
    world.escape();
    assert!(!world.state().dimension_tool_active);
}
#[then("the edited ellipse remains visible")]
fn ellipse_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().ellipses().len(), 1);
    world.expect_annotation("4");
    world.expect_annotation("8");
}
