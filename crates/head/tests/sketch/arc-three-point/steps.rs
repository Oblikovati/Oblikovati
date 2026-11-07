// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/arc-three-point`: arcs through a start, an end and a point on the curve.
use super::super::{ApplicationWorld, capture_framebuffer, diagnostics};
use super::arc_center_point::arc_fraction_pixel;
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchArc2D;
use oblikovati_viewport::ViewportPhysicalKey;

#[path = "numeric.rs"]
mod numeric;

/// The recording's 350,250, 350,350 and 300,300: moved clear of this window's browser panel.
const START: [i32; 2] = [650, 250];
const END: [i32; 2] = [650, 350];
const ON_ARC: [i32; 2] = [600, 300];
/// The recording's 1530,600 and 1530,660 lie past this window's edge: the same places at half scale.
const REPEATED_START: [i32; 2] = [765, 300];
const DISTINCT_END: [i32; 2] = [765, 330];
const ROW_START: [i32; 2] = [560, 500];
const ROW_END: [i32; 2] = [760, 500];
const ROW_BETWEEN: [i32; 2] = [660, 500];
const BELOW_ROW: [i32; 2] = [660, 600];
/// Clear of where the recorded arcs sit, to hold the fixture's ellipse and circle.
const ELLIPSE: [[i32; 2]; 3] = [[1000, 640], [1060, 640], [1000, 680]];
const CIRCLE: [[i32; 2]; 2] = [[920, 640], [940, 640]];
const LABEL_PLACE: [i32; 2] = [500, 600];
/// Radius that rounds to each recorded row: 4.751, diameter 9.501 and length 15.657.
const MAJOR_ARC: RecordedArc = RecordedArc {
    center: [22.838, 33.371],
    radius: 4.7507,
    sweep: 188.83,
};
const MINOR_ARC: RecordedArc = RecordedArc {
    center: [-9.513, 36.783],
    radius: 4.2941329,
    sweep: 167.57,
};
const LOCKED_ARC: RecordedArc = RecordedArc {
    center: [1.534, 0.604],
    radius: 6.,
    sweep: 90.33,
};
/// Measured values of a recorded arc, in millimetres along model Y and Z and in degrees.
struct RecordedArc {
    center: [f64; 2],
    radius: f64,
    sweep: f64,
}
impl RecordedArc {
    /// The two ends, each half the sweep from the arc's middle along model +Y (up on screen).
    fn ends(&self) -> [[f64; 2]; 2] {
        let (sin, cos) = (self.sweep / 2.).to_radians().sin_cos();
        [1., -1.].map(|side| {
            [
                self.center[0] + self.radius * cos,
                self.center[1] + self.radius * sin * side,
            ]
        })
    }
    fn middle(&self) -> [f64; 2] {
        [self.center[0] + self.radius, self.center[1]]
    }
}
fn select_tool(world: &mut ApplicationWorld) {
    world.choose_from_flyout("Center Arc", "Three Point Arc");
}
fn arcs(world: &mut ApplicationWorld) -> usize {
    world.edited_geometry().arcs().len()
}
fn newest_arc(world: &mut ApplicationWorld) -> SketchArc2D {
    *world
        .edited_geometry()
        .arcs()
        .last()
        .expect("expected an arc")
}
fn arc_center_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let arc = newest_arc(world);
    let center = world.edited_geometry().point(arc.center).unwrap().position;
    world.sketch_pixel_of(center)
}
/// Where the arc's middle shows, whichever way it turns.
fn arc_middle_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    arc_fraction_pixel(world, 0.5)
}
fn pointer_arc(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(START);
    world.sketch_click(END);
    world.sketch_click(ON_ARC);
}
fn exit_tool(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
fn typed_arc(world: &mut ApplicationWorld, recorded: &RecordedArc) {
    select_tool(world);
    world.open_precise_input();
    let [start, end] = recorded.ends();
    for point in [start, end, recorded.middle()] {
        world.precise_point(point);
    }
}
fn measure_arc(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = arc_fraction_pixel(world, 0.25);
    world.sketch_click(at);
}
fn close_measure(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.pump(Vec::new());
}
fn major_arc_restored(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    typed_arc(world, &MAJOR_ARC);
    exit_tool(world);
    world.undo();
    world.redo();
}
fn seeded_ready(world: &mut ApplicationWorld, needed: usize) {
    world.start_empty_sketch();
    world.seed_dimensions_needed(needed);
    world.expect_ready();
}
fn ellipse_and_circle(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.click_text("Ellipse");
    world.pump(Vec::new());
    for point in ELLIPSE {
        world.sketch_click(point);
    }
    exit_tool(world);
    world.click_text("Circle");
    world.pump(Vec::new());
    for point in CIRCLE {
        world.sketch_click(point);
    }
    exit_tool(world);
}
/// The recorded arc with a radius dimension holding its measured value.
fn dimensioned_minor_arc(world: &mut ApplicationWorld) {
    ellipse_and_circle(world);
    typed_arc(world, &MINOR_ARC);
    exit_tool(world);
    world.click_text("Dimension");
    world.pump(Vec::new());
    let at = arc_fraction_pixel(world, 0.25);
    world.sketch_click(at);
    world.sketch_click(LABEL_PLACE);
    world.click_text("Apply");
    world.pump(Vec::new());
    world.escape();
    world.pump(Vec::new());
    world.expect_annotation("4.294");
}
fn expect_rows(world: &ApplicationWorld, rows: &[&str]) {
    for row in rows {
        world.expect_measure(row);
    }
}
// arc-three-point-01
#[when("I select Three Point Arc and select a start, end, and noncollinear point on the arc")]
fn three_points(world: &mut ApplicationWorld) {
    pointer_arc(world);
}
#[then("one arc has the chosen endpoints and passes through the third point")]
fn through_all(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 1, "arcs");
    let arc = newest_arc(world);
    let geometry = world.edited_geometry();
    let pixel_of = |world: &mut ApplicationWorld, point| {
        let at = geometry.point(point).unwrap().position;
        world.sketch_pixel_of(at)
    };
    assert_eq!(pixel_of(world, arc.start), START, "start pixel");
    assert_eq!(pixel_of(world, arc.end), END, "end pixel");
    assert_eq!(arc_middle_pixel(world), ON_ARC, "middle of the turn");
}
#[then("the command requests another arc start")]
#[then("the command requests another start point")]
fn another_start(world: &mut ApplicationWorld) {
    world.expect_status("Select start of arc");
}
// arc-three-point-02 and arc-three-point-16
#[when("I select endpoints and then click their screen row between them")]
fn collinear_third(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(ROW_START);
    world.sketch_click(ROW_END);
    world.sketch_click(ROW_BETWEEN);
}
#[then("Select point on arc remains active without a committed arc or error dialog")]
#[then("Select point on arc remains active without an error dialog or new arc")]
fn on_arc_stays(world: &mut ApplicationWorld) {
    world.expect_status("Select point on arc");
    assert_eq!(arcs(world), 0, "arcs");
}
#[given("I selected the start and end on the same screen row")]
fn same_row(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    select_tool(world);
    world.sketch_click(ROW_START);
    world.sketch_click(ROW_END);
}
#[when("I click between them on that row for the third point")]
fn between(world: &mut ApplicationWorld) {
    world.sketch_click(ROW_BETWEEN);
}
#[when("I select a point below that row")]
fn below_row(world: &mut ApplicationWorld) {
    world.sketch_click(BELOW_ROW);
}
#[then("a lower-bowed arc commits and Select start of arc returns")]
fn lower_bowed(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 1, "arcs");
    assert_eq!(arc_middle_pixel(world), BELOW_ROW, "middle of the turn");
    world.expect_status("Select start of arc");
}
#[then("Ready returns with the new arc retained")]
fn ready_retained(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(arcs(world), 1, "arcs");
}
// arc-three-point-03
#[given("Three Point Arc is requesting a point on the arc after both endpoints were selected")]
fn asks_on_arc(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(START);
    world.sketch_click(END);
    world.expect_status("Select point on arc");
}
#[then("the sketch returns to Ready without a new arc")]
fn ready_empty(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(arcs(world), 0, "arcs");
}
// arc-three-point-04
#[given("I have committed one valid change through this feature")]
fn committed(world: &mut ApplicationWorld) {
    world.remember_sketch_baseline();
    pointer_arc(world);
    assert_eq!(arcs(world), 1, "arcs");
}
#[given("I have exited the creation or edit command")]
fn exited(world: &mut ApplicationWorld) {
    exit_tool(world);
}
#[when("I press Ctrl+Z once")]
fn undo_once(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("the last feature transaction is undone")]
fn undone(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 0, "arcs");
}
#[when("I press Ctrl+Y once")]
fn redo_once(world: &mut ApplicationWorld) {
    world.redo();
}
#[then("that transaction is restored")]
fn restored(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 1, "arcs");
}
#[then("the arc and surrounding geometry are visible again")]
fn visible_again(world: &mut ApplicationWorld) {
    let (from, to) = (
        arc_fraction_pixel(world, 0.2),
        arc_fraction_pixel(world, 0.3),
    );
    world.expect_segment_drawn(from, to);
}
// arc-three-point-05 and arc-three-point-09
#[given("the three-point arc has been restored by Redo")]
#[given("the three-point arc has been restored")]
fn restored_by_redo(world: &mut ApplicationWorld) {
    major_arc_restored(world);
}
#[when("I open Measure and select its curve")]
#[when("I open Measure and select its upper curve")]
fn measure_curve(world: &mut ApplicationWorld) {
    measure_arc(world);
}
#[then("radius is 4.751 mm, diameter 9.501 mm, length 15.657 mm and sweep 188.83 degrees")]
fn major_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Radius 4.751 mm",
            "Diameter 9.501 mm",
            "Length 15.657 mm",
            "Sweep 188.83 deg",
        ],
    );
}
// The recording's session wrote decimal commas; this product writes points.
#[then("its displayed radius is 4,751 mm and diameter is 9,501 mm")]
fn comma_radius(world: &mut ApplicationWorld) {
    expect_rows(world, &["Radius 4.751 mm", "Diameter 9.501 mm"]);
}
#[then("its length is 15,657 mm and angle is 188,83 deg")]
fn comma_length(world: &mut ApplicationWorld) {
    expect_rows(world, &["Length 15.657 mm", "Sweep 188.83 deg"]);
}
#[then("its center is X 0,000 mm, Y 22,838 mm, Z 33,371 mm")]
fn comma_center(world: &mut ApplicationWorld) {
    world.expect_measure("Center X 0.000 Y 22.838 Z 33.371 mm");
}
// arc-three-point-06 to arc-three-point-08
#[given("the sketch is Ready with 156 dimensions needed")]
fn ready_156(world: &mut ApplicationWorld) {
    seeded_ready(world, 156);
}
#[when("I choose Arc Three Point and click start 350,250")]
fn tool_and_start(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(START);
}
#[then("the prompt requests the end of the arc")]
fn asks_end(world: &mut ApplicationWorld) {
    world.expect_status("Select end of arc");
}
#[then("156 dimensions are still needed")]
fn still_156(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(156);
}
#[when("I invoke Three Point Arc")]
fn invoke(world: &mut ApplicationWorld) {
    select_tool(world);
}
#[when("I click start 350,250 and end 350,350")]
fn start_and_end(world: &mut ApplicationWorld) {
    world.sketch_click(START);
    world.sketch_click(END);
}
#[then("the prompt requests a point on the arc")]
fn asks_point(world: &mut ApplicationWorld) {
    world.expect_status("Select point on arc");
}
#[when("I press Escape before selecting the third point")]
fn escape_before_third(world: &mut ApplicationWorld) {
    exit_tool(world);
}
#[when("I click start 350,250, end 350,350, and point on arc 300,300")]
fn all_three(world: &mut ApplicationWorld) {
    world.sketch_click(START);
    world.sketch_click(END);
    world.sketch_click(ON_ARC);
}
#[then("an arc opening to the right appears")]
fn opens_right(world: &mut ApplicationWorld) {
    let center = arc_center_pixel(world);
    let middle = arc_middle_pixel(world);
    assert!(
        middle[0] < center[0],
        "middle {middle:?} from center {center:?}"
    );
}
#[then("the arc remains and 161 dimensions are needed")]
#[then("the arc returns and 161 dimensions are needed")]
fn arc_161(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 1, "arcs");
    world.expect_dimensions_needed(161);
}
#[then("the arc disappears and 156 dimensions are needed")]
fn gone_156(world: &mut ApplicationWorld) {
    assert_eq!(arcs(world), 0, "arcs");
    world.expect_dimensions_needed(156);
}
// arc-three-point-10
#[given("the sketch is Ready with 161 dimensions needed")]
fn ready_161(world: &mut ApplicationWorld) {
    seeded_ready(world, 161);
}
#[when("I invoke Three Point Arc and click start 1530,600")]
fn tool_repeated_start(world: &mut ApplicationWorld) {
    select_tool(world);
    world.sketch_click(REPEATED_START);
}
#[when("I click 1530,600 again for the end")]
fn repeated_end(world: &mut ApplicationWorld) {
    world.sketch_click(REPEATED_START);
}
#[then("Select end of arc remains active without an error dialog")]
fn end_stays(world: &mut ApplicationWorld) {
    world.expect_status("Select end of arc");
}
#[when("I click the distinct endpoint 1530,660")]
fn distinct_end(world: &mut ApplicationWorld) {
    world.sketch_click(DISTINCT_END);
}
#[then("Select point on arc becomes active")]
fn point_active(world: &mut ApplicationWorld) {
    world.expect_status("Select point on arc");
}
#[then("the MCAD Software returns to Ready without adding an arc")]
fn ready_no_arc(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(arcs(world), 0, "arcs");
}
#[then("161 dimensions are still needed")]
fn still_161(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(161);
}
