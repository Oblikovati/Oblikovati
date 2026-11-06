// SPDX-License-Identifier: GPL-2.0-only
//! The retained arc fixtures, and comparing a pointer drag with the fields hidden.
use super::super::super::ApplicationWorld;
use super::recorded::{expect_measure, finish_and_save, reopen_and_edit};
use cucumber::{given, then, when};
use oblikovati_model::history::SketchTangentArcCreation;
use oblikovati_viewport::ViewportPhysicalKey;

// Recorded centre, radius and sweep in millimetres; the arc starts a half-turn from sketch +x.
const CENTER: [f64; 2] = [1.203, 38.923];
const RADIUS: f64 = 2.58;
const SWEEP_DEGREES: f64 = 93.27;
const SOURCE_LENGTH: f64 = 5.;
const FOLLOWING: [f64; 2] = [4.4, 40.];
/// The source adds three, the arc and the following line two each, reading the recorded 28.
const SEEDED_BEFORE_ARC_FIXTURE: usize = 21;
/// Working lengths are centimetres; the recording and Precise Input read millimetres.
const MM_PER_CM: f64 = 10.;

fn arc_start() -> [f64; 2] {
    [CENTER[0] - RADIUS, CENTER[1]]
}
fn arc_end() -> [f64; 2] {
    let angle = (180. + SWEEP_DEGREES).to_radians();
    [
        CENTER[0] + RADIUS * angle.cos(),
        CENTER[1] + RADIUS * angle.sin(),
    ]
}
/// A straight source along sketch y, its end being where the arc begins.
fn arc_source(world: &mut ApplicationWorld) {
    let start = arc_start();
    world.select_line_tool();
    world.precise_point([start[0], start[1] + SOURCE_LENGTH]);
    world.precise_point(start);
    world.escape();
    world.pump(Vec::new());
}
fn add_tangent_arc(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let geometry = world.edited_geometry();
    let source = *geometry.lines().last().expect("expected the source line");
    let input = SketchTangentArcCreation {
        line: source.id,
        vertex: source.endpoints[1],
        end: arc_end().map(|value| value / MM_PER_CM),
    };
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_tangent_arc(edit.sketch, &input)
        .unwrap();
    world.pump(Vec::new());
}
/// The following line starts by pointer on the arc's end, which the command acquires.
fn following_line(world: &mut ApplicationWorld) {
    let at = arc_end().map(|value| value / MM_PER_CM);
    let pixel = world.sketch_pixel_of(at);
    world.select_line_tool();
    world.sketch_click(pixel);
    world.precise_point(FOLLOWING);
    world.escape();
    world.pump(Vec::new());
}
fn build_arc_fixture(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.seed_dimensions_needed(SEEDED_BEFORE_ARC_FIXTURE);
    world.open_precise_input();
    arc_source(world);
    add_tangent_arc(world);
    following_line(world);
    world.expect_ready();
    world.expect_dimensions_needed(28);
}
fn measure_arc(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = world.arc_pixel();
    world.sketch_click(at);
}
fn expect_arc_rows(world: &ApplicationWorld) {
    expect_measure(world, "Radius 2.580 mm");
    expect_measure(world, "Diameter 5.160 mm");
    expect_measure(world, "Length 4.200 mm");
    expect_measure(world, "Sweep 93.27 deg");
    expect_measure(world, "Center X 0.000 Y 1.203 Z 38.923 mm");
}
// line-27
#[given("an editable sketch contains a retained arc and shows 28 dimensions needed")]
fn retained_arc_fixture(world: &mut ApplicationWorld) {
    build_arc_fixture(world);
}
#[when("its Curve is selected in Measure")]
fn select_arc(world: &mut ApplicationWorld) {
    measure_arc(world);
}
#[then("radius is 2.580 mm and diameter is 5.160 mm")]
fn radius_diameter(world: &mut ApplicationWorld) {
    expect_measure(world, "Radius 2.580 mm");
    expect_measure(world, "Diameter 5.160 mm");
}
#[then("length is 4.200 mm and sweep is 93.27 degrees")]
fn length_sweep(world: &mut ApplicationWorld) {
    expect_measure(world, "Length 4.200 mm");
    expect_measure(world, "Sweep 93.27 deg");
}
#[then("its center is X 0.000 Y 1.203 Z 38.923 mm")]
fn center_row(world: &mut ApplicationWorld) {
    expect_measure(world, "Center X 0.000 Y 1.203 Z 38.923 mm");
}
// line-28
#[given("the retained source line, arc and following line")]
fn retained_chain(world: &mut ApplicationWorld) {
    build_arc_fixture(world);
    world.key(ViewportPhysicalKey::F8, false);
    world.pump(Vec::new());
    world.expect_annotation("V");
}
#[when("Undo first hides constraint glyphs")]
fn undo_hides_glyphs(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("geometry and the 28-dimension count remain")]
fn count_stays(world: &mut ApplicationWorld) {
    world.expect_no_annotation("V");
    world.expect_dimensions_needed(28);
    assert_eq!(world.edited_geometry().arcs().len(), 1, "expected the arc");
}
#[when("further Undo removes the following line then the arc")]
fn undo_twice(world: &mut ApplicationWorld) {
    world.observed_counts.clear();
    for _ in 0..2 {
        world.undo();
        world.observed_counts.push(world.status_dimensions_needed());
    }
}
#[then("the counts become 26 then 24")]
fn counts_fall(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_counts, [26, 24], "counts after each undo");
    assert!(
        world.edited_geometry().arcs().is_empty(),
        "expected the arc gone"
    );
}
#[when("both creations are redone")]
fn redo_both(world: &mut ApplicationWorld) {
    world.redo();
    world.redo();
}
#[then("the count returns to 28 and all displayed arc measurements match")]
fn count_back(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(28);
    measure_arc(world);
    expect_arc_rows(world);
    world.click_text("Done");
    world.pump(Vec::new());
}
#[then("the history lists Create Line, Create Tangent Arc, Create Line latest first")]
fn history_order(world: &mut ApplicationWorld) {
    world.open_undo_history();
    let rows = world.undo_history_rows();
    assert_eq!(
        rows[..3],
        ["CreateLine", "CreateTangentArc", "CreateLine"],
        "history rows {rows:?}"
    );
}
// line-29
#[given("the measured arc fixture is finished and saved")]
fn arc_saved(world: &mut ApplicationWorld) {
    build_arc_fixture(world);
    finish_and_save(world);
}
#[when("the part is closed, reopened and its sketch edited in the same session")]
fn arc_reopened(world: &mut ApplicationWorld) {
    reopen_and_edit(world);
}
#[then("the chain remains with 28 dimensions needed")]
fn chain_remains(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(28);
}
#[then("arc radius, diameter, length, sweep and center match the baseline")]
fn arc_matches(world: &mut ApplicationWorld) {
    measure_arc(world);
    expect_arc_rows(world);
}
// line-30
const DRAG_END: [i32; 2] = [1000, 520];
const PREVIEW_LENGTH_CM: f64 = 1.0271;
const PREVIEW_ANGLE_DEGREES: f64 = 229.88;
const SEEDED_BEFORE_SOURCE: usize = 24;
fn toggle_sketch_hud(world: &mut ApplicationWorld) {
    world.open_application_options();
    world.click_text("Sketch HUD");
    world.pump(Vec::new());
    world.close_application_options();
}
/// Worked back from where the pointer lands, so the drag reads the recorded values exactly.
fn source_end_for_drag(world: &mut ApplicationWorld) -> [f64; 2] {
    let end = world.sketch_point_of(DRAG_END);
    let direction = PREVIEW_ANGLE_DEGREES.to_radians();
    [
        end[0] - PREVIEW_LENGTH_CM * direction.cos(),
        end[1] - PREVIEW_LENGTH_CM * direction.sin(),
    ]
}
#[given("Sketch HUD is enabled")]
fn hud_enabled(world: &mut ApplicationWorld) {
    world.open_application_options();
    assert!(world.checked("Sketch HUD"), "expected Sketch HUD checked");
    world.close_application_options();
}
#[when("HUD is temporarily disabled and Line acquires the original source endpoint")]
fn hud_off(world: &mut ApplicationWorld) {
    toggle_sketch_hud(world);
    world.start_sketch_on("YZ Plane");
    world.seed_dimensions_needed(SEEDED_BEFORE_SOURCE);
    let source_end = source_end_for_drag(world);
    let in_mm = source_end.map(|value| value * MM_PER_CM);
    world.open_precise_input();
    world.select_line_tool();
    // Clear of the seeds' directions, so no relation moves this end off its exact position.
    world.precise_point([in_mm[0] + 8., in_mm[1] + 1.2]);
    world.precise_point(in_mm);
    world.escape();
    world.pump(Vec::new());
    world.observed_view = Some(world.sketch_pixel_of(source_end));
    world.select_line_tool();
}
#[when("a drag from 1232,355 to 1400,500 is released")]
fn drag_released(world: &mut ApplicationWorld) {
    let from = world.observed_view.expect("expected the source's end");
    world.drag(from, DRAG_END);
}
#[then("a straight pending preview remains with length 10.271 mm and angle 229.88 degrees")]
fn preview_values(world: &mut ApplicationWorld) {
    world.expect_status("Length 10.271 mm");
    world.expect_status("Angle 229.88 deg");
    let from = world.observed_view.expect("expected the source's end");
    world.expect_segment_drawn(from, DRAG_END);
}
#[when("Escape cancels the preview")]
fn escape_preview(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("the retained fixture remains at 28 dimensions needed")]
fn retained_twenty_eight(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_dimensions_needed(28);
}
#[when("HUD is restored and Application Options reopened")]
fn hud_restored(world: &mut ApplicationWorld) {
    toggle_sketch_hud(world);
    world.open_application_options();
}
#[then("HUD is checked again")]
fn hud_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Sketch HUD"), "expected Sketch HUD checked");
    world.close_application_options();
}
