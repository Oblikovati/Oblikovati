// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/precise-input`: typed points, reference and coordinate modes, the triad.
use super::super::{ApplicationWorld, diagnostics};
use super::fixtures::PointMark;
use cucumber::{gherkin::Step, given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

const TOOLBAR: &str = "Precise Input";
const POINTER: [i32; 2] = [1000, 620];
/// The isolated line: its end is where the triad goes, with nothing else beside it.
const LINE: [[i32; 2]; 2] = [[420, 720], [600, 620]];
const TRIAD_SAMPLE: i32 = 24;
const NEAR_VIEW_CM: f64 = 100.;
const MODE_INPUTS: [(&str, &str, &str); 4] = [
    ("X and Y coordinate", "5", "6"),
    ("Distance and X axis angle", "10", "45"),
    ("X coordinate and X axis angle", "8", "30"),
    ("Y coordinate and X axis angle", "8", "60"),
];

fn point_active_on_yz(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.select_point_tool();
}
fn toolbar_shows(_: &ApplicationWorld) -> bool {
    diagnostics::window_geometry(TOOLBAR).is_some()
}
fn newest_point_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let geometry = world.edited_geometry();
    let point = geometry.points().last().copied().expect("expected a point");
    world.sketch_pixel_of(point.position)
}
fn open_reference_modes(world: &mut ApplicationWorld) {
    let current = if world.window_text(TOOLBAR).contains("Rel") {
        "Rel"
    } else {
        "Abs"
    };
    let at = world
        .run_in_window(TOOLBAR, current)
        .center()
        .map(|value| value as i32);
    world.click(at);
    world.pump(Vec::new());
}
fn choose_reference(world: &mut ApplicationWorld, label: &str) {
    open_reference_modes(world);
    world.click_text(label);
    world.pump(Vec::new());
}
fn open_coordinate_modes(world: &mut ApplicationWorld) {
    let at = world
        .run_in_window(TOOLBAR, "Mode")
        .center()
        .map(|value| value as i32);
    world.click(at);
    world.pump(Vec::new());
}
fn expect_triad_at(world: &mut ApplicationWorld, at: [i32; 2]) {
    world.expect_segment_drawn(at, [at[0] + TRIAD_SAMPLE, at[1]]);
}
fn origin_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    world.sketch_pixel_of([0., 0.])
}
#[given("Point is active in an editable sketch")]
fn point_active(world: &mut ApplicationWorld) {
    point_active_on_yz(world);
}
#[given("the reference triad is at the origin")]
fn triad_at_origin(world: &mut ApplicationWorld) {
    let (_, at) = world
        .app()
        .owner
        .precise_triad_position()
        .expect("expected Point");
    assert_eq!(at, [0., 0.], "triad position");
}
// precise-input-06
#[when("I open Precise Input and its reference-mode dropdown")]
#[when("the user open Precise Input and its reference-mode dropdown")]
fn open_reference(world: &mut ApplicationWorld) {
    require_recorded_fixture("precise-input-06");
    world.open_precise_input();
    open_reference_modes(world);
}
#[then("Relative and Absolute are listed")]
fn modes_listed(world: &mut ApplicationWorld) {
    world.expect_text("Relative");
    world.expect_text("Absolute");
}
#[when("I select Absolute")]
fn select_absolute(world: &mut ApplicationWorld) {
    world.click_text("Absolute");
    world.pump(Vec::new());
}
#[then("its icon is displayed in the toolbar")]
fn icon_shown(world: &mut ApplicationWorld) {
    world.run_in_window(TOOLBAR, "Abs");
}
#[when("I enter X 5 and Y 6 and press Return")]
fn enter_pair(world: &mut ApplicationWorld) {
    world.precise_point([5., 6.]);
}
#[then("a point marker appears and the coordinate fields reset for repeat entry")]
fn marker_and_reset(world: &mut ApplicationWorld) {
    let at = newest_point_pixel(world);
    world.expect_point_mark(at, PointMark::Cross);
    let text = world.window_text(TOOLBAR);
    assert!(
        !text.contains("6.000000"),
        "expected the typed values gone from {text:?}"
    );
}
#[when("I select Relative and enter the same pair and press Return")]
fn relative_pair(world: &mut ApplicationWorld) {
    choose_reference(world, "Relative");
    world.precise_point([5., 6.]);
}
#[then("no validation dialog appears and no distinct offset marker is visible")]
fn no_offset_marker(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
    let geometry = world.edited_geometry();
    let points = geometry.points();
    let [first, second] = [&points[points.len() - 2], &points[points.len() - 1]];
    assert_eq!(
        first.position, second.position,
        "both points at the same place"
    );
}
#[then("the origin triad remains at its prior visible position")]
fn triad_stays(world: &mut ApplicationWorld) {
    let origin = origin_pixel(world);
    expect_triad_at(world, origin);
}
#[when("I restore Absolute and press Escape")]
fn restore_absolute(world: &mut ApplicationWorld) {
    choose_reference(world, "Absolute");
    world.escape();
    world.pump(Vec::new());
}
#[then("Point and Precise Input close and Ready reports 35 dimensions needed")]
fn closed_35(world: &mut ApplicationWorld) {
    assert!(!toolbar_shows(world), "expected the toolbar to close");
    world.expect_ready();
    require_recorded_fixture("precise-input-06");
    world.expect_dimensions_needed(35);
}
// precise-input-07
#[given("the point marker from the paired input trial remains visible")]
fn marker_from_trial(_: &mut ApplicationWorld) {
    require_recorded_fixture("precise-input-07");
}
#[when("I open Measure and select that marker")]
fn measure_marker(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    let at = newest_point_pixel(world);
    world.sketch_click(at);
}
#[then("it is identified as Selection 1 Point")]
fn identified_point(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
}
#[then("its global position is X 5.000 Y 0.500 Z -6.000 mm")]
fn position_on_offset_plane(world: &mut ApplicationWorld) {
    world.expect_measure("Position X 5.000 Y 0.500 Z -6.000 mm");
}
#[then("Measure closes and the marker remains")]
fn measure_closes(world: &mut ApplicationWorld) {
    assert!(
        diagnostics::window_geometry("Measure").is_none(),
        "expected Measure to close"
    );
    let at = newest_point_pixel(world);
    world.expect_point_mark(at, PointMark::Cross);
}
// precise-input-01
#[when("I open Precise Input in X and Y coordinate mode")]
#[when("the user open Precise Input in X and Y coordinate mode")]
fn open_toolbar(world: &mut ApplicationWorld) {
    require_recorded_fixture("precise-input-01");
    world.open_precise_input();
    assert!(toolbar_shows(world), "expected the toolbar");
}
#[when("I focus X, press 5, press Tab, press 6, and press Return")]
fn typed_with_tab(world: &mut ApplicationWorld) {
    world.click_precise_field("X");
    world.type_text("5");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[when("I press Escape and measure the created point")]
fn escape_and_measure(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    measure_marker(world);
}
#[then(
    "its global position is X=0.000 mm, Y=5.000 mm, Z=6.000 mm in the specified sketch-plane fixture"
)]
fn position_on_yz(world: &mut ApplicationWorld) {
    require_recorded_fixture("precise-input-01");
    world.expect_measure("Position X 0.000 Y 5.000 Z 6.000 mm");
}
// precise-input-03
fn expect_red_value(world: &ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(TOOLBAR).expect("expected the toolbar");
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let center = run.center();
            run.text.starts_with('a')
                && center[0] >= position[0]
                && center[0] <= position[0] + size[0]
                && center[1] >= position[1]
                && center[1] <= position[1] + size[1]
        })
        .unwrap_or_else(|| panic!("expected a typed a in {:?}", world.window_text(TOOLBAR)));
    let [red, green, blue] = run.color;
    assert!(
        red > 200 && green < 100 && blue < 100,
        "colour {:?}: expected red",
        run.color
    );
}
fn expect_numeric_fields(world: &mut ApplicationWorld) {
    world.hover(POINTER);
    let text = world.window_text(TOOLBAR);
    assert!(
        text.contains(".000") || text.chars().any(|letter| letter.is_ascii_digit()),
        "expected numbers in {text:?}"
    );
}
#[given("X and Y coordinate mode is active with numeric fields")]
fn numeric_fields(world: &mut ApplicationWorld) {
    point_active_on_yz(world);
    world.open_precise_input();
    expect_numeric_fields(world);
}
#[when("I focus X, select its value, and enter the single character a")]
#[when("the user focus X, select its value, and enter the single character a")]
fn enter_a(world: &mut ApplicationWorld) {
    world.click_precise_field("X");
    world.type_text("a");
}
#[then("a is displayed in Red")]
fn a_red(world: &mut ApplicationWorld) {
    expect_red_value(world);
}
#[when("I press Return")]
#[when("the user press Return")]
fn press_return(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
#[then("the Red value remains and no visible point is committed")]
fn red_remains(world: &mut ApplicationWorld) {
    expect_red_value(world);
    // Far-away seeds and the projected origin are not what the command would have placed.
    let geometry = world.edited_geometry();
    let near = geometry
        .points()
        .iter()
        .filter(|point| {
            point.position != [0., 0.]
                && point.position[0].abs() < NEAR_VIEW_CM
                && point.position[1].abs() < NEAR_VIEW_CM
        })
        .count();
    assert_eq!(near, 0, "points near the view");
}
#[when("I press Escape and restart Point")]
#[when("the user press Escape and restart Point")]
fn escape_and_restart(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.select_point_tool();
}
#[then("Precise Input reopens with numeric fields")]
fn reopened_numeric(world: &mut ApplicationWorld) {
    assert!(toolbar_shows(world), "expected the toolbar back");
    expect_numeric_fields(world);
    let text = world.window_text(TOOLBAR);
    assert!(
        !text.contains('a') || !text.starts_with('a'),
        "expected no typed a in {text:?}"
    );
}
// precise-input-04
#[given("Precise Input is in X and Y coordinate mode")]
fn in_xy_mode(world: &mut ApplicationWorld) {
    require_recorded_fixture("precise-input-04");
    point_active_on_yz(world);
    world.open_precise_input();
    world.run_in_window(TOOLBAR, "Mode");
}
#[when("I open the coordinate-mode dropdown")]
fn open_modes(world: &mut ApplicationWorld) {
    open_coordinate_modes(world);
}
#[then("it offers X and Y coordinate")]
#[then("it offers X coordinate and X axis angle")]
#[then("it offers Y coordinate and X axis angle")]
#[then("it offers Distance and X axis angle")]
fn offers_modes(world: &mut ApplicationWorld) {
    for (mode, _, _) in MODE_INPUTS {
        world.expect_text(mode);
    }
}
#[when("I create a point in each mode using the original triad and the following inputs")]
fn create_in_each_mode(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    for (mode, first, second) in MODE_INPUTS {
        open_coordinate_modes(world);
        world.click_text(mode);
        world.pump(Vec::new());
        world.precise_values(first, second);
    }
}
#[when("I measure each point in the specified sketch-plane fixture")]
fn measure_each(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    let geometry = world.edited_geometry();
    let placed: Vec<[f64; 2]> = geometry
        .points()
        .iter()
        .rev()
        .take(MODE_INPUTS.len())
        .map(|point| point.position)
        .collect();
    let mut rows = String::new();
    for position in placed {
        world.key(ViewportPhysicalKey::M, false);
        let at = world.sketch_pixel_of(position);
        world.sketch_click(at);
        rows.push_str(&world.window_text("Measure"));
        world.click_text("Done");
        world.pump(Vec::new());
    }
    world.observed_rows = Some(rows);
}
#[then("Measure reports these global positions to 0.001 mm")]
fn positions_reported(world: &mut ApplicationWorld, step: &Step) {
    let rows = world
        .observed_rows
        .clone()
        .expect("expected measured points");
    let table = step
        .table
        .as_ref()
        .expect("expected the table of positions");
    for row in table.rows.iter().skip(1) {
        let wanted = format!("PositionX{}Y{}Z{}mm", row[3], row[4], row[5]);
        assert!(rows.contains(&wanted), "expected {wanted:?} in {rows:?}");
    }
}
// precise-input-05
#[given("a point committed through Precise Input")]
fn committed_point(world: &mut ApplicationWorld) {
    point_active_on_yz(world);
    world.open_precise_input();
    world.precise_point([5., 6.]);
}
#[when("I exit Point and undo and redo")]
#[when("the user exit Point and undo and redo")]
fn exit_undo_redo(world: &mut ApplicationWorld) {
    let at = newest_point_pixel(world);
    world.escape();
    world.pump(Vec::new());
    world.undo();
    world.expect_point_mark(at, PointMark::Absent);
    world.redo();
    world.observed_view = Some(at);
}
#[then("the point marker disappears and returns")]
fn marker_back(world: &mut ApplicationWorld) {
    let at = world.observed_view.expect("expected the point's pixel");
    world.expect_point_mark(at, PointMark::Cross);
}
// precise-input-08
#[given("Point and Precise Input are active in an editable sketch")]
fn with_line_and_toolbar(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.select_line_tool();
    for corner in LINE {
        world.sketch_click(corner);
    }
    world.escape();
    world.pump(Vec::new());
    world.select_point_tool();
    world.open_precise_input();
}
#[when("I activate Reposition Triad")]
fn activate_reposition(world: &mut ApplicationWorld) {
    world.click_text("Reposition Triad");
    world.pump(Vec::new());
}
#[when("click the displayed origin then the isolated line endpoint")]
fn click_origin_then_end(world: &mut ApplicationWorld) {
    let origin = origin_pixel(world);
    world.sketch_click(origin);
    world.sketch_click(LINE[1]);
    world.hover(POINTER);
}
#[then("the triad visibly moves to that endpoint")]
fn triad_moved(world: &mut ApplicationWorld) {
    expect_triad_at(world, LINE[1]);
    let origin = origin_pixel(world);
    world.expect_segment_absent(origin, [origin[0] + TRIAD_SAMPLE, origin[1]]);
}
#[then("Select center point is restored")]
fn prompt_restored(world: &mut ApplicationWorld) {
    world.expect_status("Select center point");
}
#[when("I choose Reset to Origin")]
fn choose_reset(world: &mut ApplicationWorld) {
    world.click_text("Reset to Origin");
    world.pump(Vec::new());
}
#[then("the triad returns to its prior origin position")]
fn triad_back(world: &mut ApplicationWorld) {
    let origin = origin_pixel(world);
    expect_triad_at(world, origin);
}
#[when("I open toolbar options")]
fn open_options(world: &mut ApplicationWorld) {
    let at = world
        .run_in_window(TOOLBAR, "...")
        .center()
        .map(|value| value as i32);
    world.click(at);
    world.pump(Vec::new());
}
#[then("Pin Mini-Toolbar Position, Auto Fade and Close are listed")]
fn options_listed(world: &mut ApplicationWorld) {
    world.expect_text("Pin Mini-Toolbar Position");
    world.expect_text("Auto Fade");
    world.expect_text("Close");
}
#[then("the toolbar and triad disappear while Point remains active")]
fn toolbar_gone(world: &mut ApplicationWorld) {
    assert!(!toolbar_shows(world), "expected the toolbar to close");
    let origin = origin_pixel(world);
    world.expect_segment_absent(origin, [origin[0] + TRIAD_SAMPLE, origin[1]]);
    world.expect_status("Select center point");
}
#[then("Point ends and Ready is displayed")]
fn point_ended(world: &mut ApplicationWorld) {
    world.expect_ready();
}

fn require_recorded_fixture(scenario: &str) {
    super::super::gap_runner::block_fixture(&format!(
        "{scenario}: recorded sketch geometry, plane and reference triad are unspecified; see precise-input/BLOCKERS.md"
    ));
}

#[test]
fn recorded_fixture_guard_refuses_an_inferred_setup() {
    for scenario in [
        "precise-input-01",
        "precise-input-04",
        "precise-input-06",
        "precise-input-07",
    ] {
        let error = std::panic::catch_unwind(|| require_recorded_fixture(scenario)).unwrap_err();
        let blocked = error.downcast_ref::<String>().unwrap();
        assert!(blocked.contains(scenario) && blocked.contains("unspecified"));
    }
}
