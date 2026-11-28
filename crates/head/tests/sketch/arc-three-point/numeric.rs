// SPDX-License-Identifier: GPL-2.0-only
//! Arc-three-point scenarios that read recorded sizes: typed fields, dimensions and their edits.
use super::*;
// arc-three-point-11
#[given("an editable sketch containing the ellipse and tangent-circle fixture")]
fn fixture_sketch(world: &mut ApplicationWorld) {
    ellipse_and_circle(world);
}
#[when("the user select Three Point Arc and supply the specified start, end and arc point")]
fn specified_points(_: &mut ApplicationWorld) {
    require_recorded_fixture("recorded minor arc pointer inputs");
}
#[then("the arc commits and Select start of arc returns")]
fn arc_commits(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 1, "arcs");
    world.expect_status("Select start of arc");
}
#[when("the user exit and select the arc curve in Measure")]
fn exit_measure(world: &mut ApplicationWorld) {
    exit_tool(world);
    measure_arc(world);
}
#[then("radius is 4.294 mm and diameter is 8.588 mm")]
fn minor_radius(world: &mut ApplicationWorld) {
    expect_rows(world, &["Radius 4.294 mm", "Diameter 8.588 mm"]);
}
#[then("length is 12.559 mm and angle is 167.57 degrees")]
fn minor_length(world: &mut ApplicationWorld) {
    expect_rows(world, &["Length 12.559 mm", "Sweep 167.57 deg"]);
}
#[then("center is (0.000, -9.513, 36.783) mm")]
fn minor_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y -9.513 Z 36.783 mm");
}
// arc-three-point-12 and arc-three-point-13
#[given("the new arc radius dimension d3 shows 4.2941329 mm")]
fn dimension_shows(world: &mut ApplicationWorld) {
    dimensioned_minor_arc(world);
    world.open_dimension_label("4.294");
    let text = world.window_text("Edit Dimension");
    assert!(
        text.contains("4.2941329mm"),
        "expected the value in {text:?}"
    );
}
#[when("the user enter 6 and cancel with Escape")]
fn six_escape(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.escape();
    world.pump(Vec::new());
}
#[then("the placed radius dimension remains with annotation 4.294")]
fn dimension_remains(world: &mut ApplicationWorld) {
    world.expect_annotation("4.294");
}
#[when("the user reopen d3")]
fn reopen(world: &mut ApplicationWorld) {
    world.open_dimension_label("4.294");
}
#[then("the stored expression is 4.2941329 mm")]
fn stored(world: &mut ApplicationWorld) {
    let text = world.window_text("Edit Dimension");
    assert!(
        text.contains("4.2941329mm"),
        "expected the value in {text:?}"
    );
}
#[given("d3 retains the original arc radius")]
#[given(
    "the recorded arc's radial dimension editor is open with its original 4.2941329 mm value selected"
)]
fn dimension_retained(world: &mut ApplicationWorld) {
    dimensioned_minor_arc(world);
}
#[when("the user enter 6 and accept with Return")]
fn six_return(world: &mut ApplicationWorld) {
    world.open_dimension_label("4.294");
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("the user exit General Dimension and measure the arc")]
fn exit_dimension_measure(world: &mut ApplicationWorld) {
    exit_tool(world);
    measure_arc(world);
}
#[then("radius is 6.000 mm and diameter is 12.000 mm")]
fn six_radius(world: &mut ApplicationWorld) {
    expect_rows(world, &["Radius 6.000 mm", "Diameter 12.000 mm"]);
}
#[then("length is 17.548 mm and angle is 167.57 degrees")]
fn six_length(world: &mut ApplicationWorld) {
    expect_rows(world, &["Length 17.548 mm", "Sweep 167.57 deg"]);
}
#[then("the displayed center remains (0.000, -9.513, 36.783) mm")]
fn center_stays(world: &mut ApplicationWorld) {
    minor_center(world);
}
#[when("the user close Measure and Undo")]
fn close_and_undo(world: &mut ApplicationWorld) {
    close_measure(world);
    world.undo();
}
#[then("annotation 4.294 and the smaller arc return")]
fn smaller(world: &mut ApplicationWorld) {
    world.expect_annotation("4.294");
}
#[when("the user Redo")]
fn redo(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("annotation 6 and the enlarged arc return")]
fn enlarged(world: &mut ApplicationWorld) {
    world.expect_annotation("6");
    world.expect_no_annotation("4.294");
}
// arc-three-point-14
#[given("an editable sketch with 9 dimensions needed")]
fn ready_9(world: &mut ApplicationWorld) {
    seeded_ready(world, 9);
}
#[when(
    "the user start an arc and enter endpoint distance 6 and angle 0 through the numeric input fields"
)]
fn keyboard_end(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(ROW_START);
    world.hover(ROW_END);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_status("Select point on arc");
}
#[when("the user enter 4 at point-on-arc selection and press Return")]
fn four_return(world: &mut ApplicationWorld) {
    world.type_text("4");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("Select end of arc is shown and no new arc is visible")]
fn end_again(world: &mut ApplicationWorld) {
    world.expect_status("Select end of arc");
    assert_eq!(arcs(world), 0, "arcs");
}
#[then("Ready returns with 9 dimensions needed and earlier geometry retained")]
fn ready_nine(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(9);
}
// arc-three-point-15
#[given("distinct mouse-selected endpoints")]
fn locked_endpoints(_: &mut ApplicationWorld) {
    require_recorded_fixture("recorded locked-radius endpoint inputs");
}
#[when("the user enter 6 in the radius field and press Tab")]
fn radius_tab(world: &mut ApplicationWorld) {
    world.type_text("6");
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
}
#[then("the field shows 6.000 mm with a lock icon")]
fn field_locked(world: &mut ApplicationWorld) {
    let text = world.window_text("##sketch-hud");
    assert!(text.contains("6.000mm"), "expected 6.000 mm in {text:?}");
    let (position, size) =
        diagnostics::window_geometry("##sketch-hud").expect("expected the fields");
    let capture = capture_framebuffer(&mut world.app().window.native);
    let row = (position[1] + size[1] / 2.) as usize;
    // Past the value's text, inside the field: only the padlock draws there.
    let columns = (position[0] + 70.) as usize..(position[0] + size[0]) as usize;
    let width = capture.extent[0] as usize;
    let pixel = |column: usize| &capture.rgba[(row * width + column) * 4..][..3];
    let first = pixel(columns.start);
    assert!(
        columns.clone().any(|column| pixel(column) != first),
        "expected a padlock in the field's end, columns {columns:?}"
    );
}
#[when("the user click a suitable canvas location and exit repetition")]
fn click_and_exit(_: &mut ApplicationWorld) {
    require_recorded_fixture("recorded locked-radius canvas input");
}
#[then("an upper-bowed arc and radial dimension 6 are retained")]
fn upper_bowed(_: &mut ApplicationWorld) {
    require_recorded_fixture("recorded locked-radius chord");
}
#[when("the user measure its curve")]
fn measure_its_curve(world: &mut ApplicationWorld) {
    measure_arc(world);
}
#[then("length is 9.459 mm and sweep is 90.33 degrees")]
fn locked_length(world: &mut ApplicationWorld) {
    expect_rows(world, &["Length 9.459 mm", "Sweep 90.33 deg"]);
}
#[then("center is (0.000, 1.534, 0.604) mm")]
fn locked_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 1.534 Z 0.604 mm");
}
