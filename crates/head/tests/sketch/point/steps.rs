// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/point`: repeated placement, center points, and their history.
use super::super::ApplicationWorld;
use super::fixtures::PointMark;
use cucumber::{given, then, when};

/// Clear of the origin, the toolbar and each other.
const FIRST: [i32; 2] = [560, 620];
const SECOND: [i32; 2] = [980, 640];
const EMPTY: [i32; 2] = [1100, 700];
const CANDIDATE_LIST_ABOVE_ROW: f32 = 30.;

fn toggle_center_point_format(world: &mut ApplicationWorld) {
    let panel = world.run("Format").center();
    world.click([panel[0] as i32, panel[1] as i32 - 35]);
    world.pump(Vec::new());
    world.click_text("Center Point");
    world.pump(Vec::new());
}
fn set_center_point_format(world: &mut ApplicationWorld, enabled: bool) {
    assert!(
        world.state().sketch_selection.entities.is_empty(),
        "expected no selected geometry"
    );
    if world.state().center_point_format != enabled {
        toggle_center_point_format(world);
    }
    assert_eq!(
        world.state().center_point_format,
        enabled,
        "Center Point mode"
    );
}
fn place_regular_points(world: &mut ApplicationWorld, points: &[[i32; 2]]) {
    set_center_point_format(world, false);
    world.select_point_tool();
    for point in points {
        world.sketch_click(*point);
    }
    world.escape();
    world.pump(Vec::new());
}
fn open_context_menu(world: &mut ApplicationWorld, at: [i32; 2]) {
    world.right_click(at);
    world.pump(Vec::new());
}
fn choose_center_point(world: &mut ApplicationWorld) {
    world.click_text("Center Point");
    world.pump(Vec::new());
    world.hover(EMPTY);
}
fn expect_placed_points(world: &mut ApplicationWorld, count: usize) {
    // The projected sketch origin is a point too, and no one placed it.
    let geometry = world.edited_geometry();
    let placed = geometry
        .points()
        .iter()
        .filter(|point| point.position != [0., 0.])
        .count();
    assert_eq!(placed, count, "placed points");
}
#[given("an editable planar sketch is active")]
fn sketch_active(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
}
#[given("an editable planar sketch has no active selection and Center Point is enabled in Format")]
fn empty_center_point_fixture(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    set_center_point_format(world, true);
    world.remember_edited_sketch(Vec::new());
}
#[when("the user press Escape without placing a point")]
fn cancel_unplaced_point(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("no new point marker is visible")]
fn no_marker_after_cancel(world: &mut ApplicationWorld) {
    world.expect_edited_sketch_retained();
    world.expect_point_mark(FIRST, PointMark::Absent);
    world.expect_point_mark(SECOND, PointMark::Absent);
}
#[then("the menu closes and Ready remains displayed")]
fn menu_closed_and_ready(world: &mut ApplicationWorld) {
    world.expect_no_text("Repeat Point");
    world.expect_ready();
}
// point-03
#[given("Point is active with Select center point")]
fn point_active(world: &mut ApplicationWorld) {
    if world.state().dimensions_needed.is_none() {
        world.start_empty_sketch();
    }
    world.select_point_tool();
    world.expect_status("Select center point");
}
#[when("I press Escape twice without clicking the canvas")]
#[when("the user press Escape twice without clicking the canvas")]
fn escape_twice(world: &mut ApplicationWorld) {
    world.escape();
    world.escape();
    world.pump(Vec::new());
}
#[when("the user right-click empty canvas to inspect the refreshed state")]
#[when("I right-click empty canvas to inspect state")]
#[when("the user right-click empty canvas to inspect state")]
fn right_click_empty(world: &mut ApplicationWorld) {
    open_context_menu(world, EMPTY);
}
#[then("Ready is displayed and the menu offers Repeat Point, Center Point")]
#[then("the status is Ready and the menu offers Repeat Point, Center Point")]
fn ready_with_menu(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_text("Repeat Point");
    world.expect_text("Center Point");
}
#[then("no new point marker is retained")]
fn no_new_point(world: &mut ApplicationWorld) {
    expect_placed_points(world, 0);
}
#[then("the Ready-state menu closes")]
fn menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Repeat Point");
}
// point-04
#[given("a center point remains after placement and Escape")]
fn center_point_placed(world: &mut ApplicationWorld) {
    world.select_point_tool();
    world.sketch_click(FIRST);
    world.escape();
    world.pump(Vec::new());
    world.expect_point_mark(FIRST, PointMark::Cross);
}
#[then("the point cross disappears")]
fn cross_gone(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Absent);
}
#[then("the point cross returns at its prior visible location")]
fn cross_back(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Cross);
}
// point-05
#[given("the Format Center Point toggle is off")]
fn format_off(world: &mut ApplicationWorld) {
    set_center_point_format(world, false);
}
#[when("I invoke Point")]
#[when("the user invoke Point")]
fn invoke_point(world: &mut ApplicationWorld) {
    world.select_point_tool();
}
#[then("the prompt is Select Sketch Point")]
fn prompt_regular(world: &mut ApplicationWorld) {
    world.expect_status("Select Sketch Point");
}
#[when("I click two separate canvas positions")]
#[when("the user click two separate canvas positions")]
fn click_two(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.sketch_click(SECOND);
    world.hover(EMPTY);
}
#[then("both dot-style markers appear")]
fn both_dots(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Dot);
    world.expect_point_mark(SECOND, PointMark::Dot);
}
#[then("Point remains available after each placement")]
fn point_stays(world: &mut ApplicationWorld) {
    world.expect_status("Select Sketch Point");
}
#[then("Ready is displayed with both point markers retained")]
fn ready_with_both(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_point_mark(FIRST, PointMark::Dot);
    world.expect_point_mark(SECOND, PointMark::Dot);
}
fn expect_point_role(world: &mut ApplicationWorld, at: [i32; 2], center: bool) {
    let geometry = world.edited_geometry();
    let point = geometry
        .points()
        .iter()
        .find(|point| world.sketch_pixel_of(point.position) == at)
        .unwrap_or_else(|| panic!("point at {at:?}: expected an authored point"));
    assert_eq!(
        geometry.is_center_point(point.id),
        center,
        "point role at {at:?}"
    );
}
// point-06 and point-07
#[given("a regular dot-style sketch point and a separate unchanged point")]
fn two_regular(world: &mut ApplicationWorld) {
    place_regular_points(world, &[FIRST, SECOND]);
}
#[when("the user right-click the first point and choose Center Point")]
#[when("I right-click the first point and choose Center Point")]
fn convert_first(world: &mut ApplicationWorld) {
    open_context_menu(world, FIRST);
    choose_center_point(world);
}
#[then("its selected marker is a cross")]
#[then("the point is a Center Point again")]
fn first_is_cross(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Cross);
    expect_point_role(world, FIRST, true);
}
#[then("reopening its context menu shows Center Point checked")]
fn reopened_checked(world: &mut ApplicationWorld) {
    open_context_menu(world, FIRST);
    assert!(
        world.checked("Center Point"),
        "expected Center Point checked"
    );
}
#[when("the user choose Center Point again")]
#[when("I choose Center Point again")]
fn choose_again(world: &mut ApplicationWorld) {
    choose_center_point(world);
}
#[then("the first point returns to a dot marker")]
#[then("the point returns to its dot marker")]
fn first_is_dot(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Dot);
    expect_point_role(world, FIRST, false);
}
#[then("the separate point retains its dot marker at its displayed location")]
fn second_is_dot(world: &mut ApplicationWorld) {
    world.expect_point_mark(SECOND, PointMark::Dot);
    expect_point_role(world, SECOND, false);
}
#[given("the user have converted a regular sketch point to Center Point")]
#[given("I have converted a regular sketch point to Center Point")]
fn converted(world: &mut ApplicationWorld) {
    place_regular_points(world, &[FIRST]);
    convert_first(world);
    world.expect_point_mark(FIRST, PointMark::Cross);
}
#[when("the user reopen the point context menu")]
#[when("I reopen the point context menu")]
fn reopen_menu(world: &mut ApplicationWorld) {
    open_context_menu(world, FIRST);
}
#[then("Center Point is checked")]
fn is_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Center Point"),
        "expected Center Point checked"
    );
}
// point-08
#[given("no geometry is selected and Center Point is enabled in Format")]
fn format_on(world: &mut ApplicationWorld) {
    place_regular_points(world, &[SECOND]);
    set_center_point_format(world, true);
}
#[then("the prompt is Select center point")]
fn prompt_center(world: &mut ApplicationWorld) {
    world.expect_status("Select center point");
}
#[when("I click empty canvas and press Escape")]
#[when("the user click empty canvas and press Escape")]
fn click_and_escape(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.escape();
    world.pump(Vec::new());
    world.hover(EMPTY);
}
#[then("a center point remains at that location")]
fn center_remains(world: &mut ApplicationWorld) {
    world.expect_point_mark(FIRST, PointMark::Cross);
}
#[then("existing regular points retain their dot-style type")]
fn regular_retained(world: &mut ApplicationWorld) {
    world.expect_point_mark(SECOND, PointMark::Dot);
    let geometry = world.edited_geometry();
    let points: Vec<_> = geometry
        .points()
        .iter()
        .filter(|point| point.position != [0., 0.])
        .collect();
    assert_eq!(points.len(), 2, "placed point count");
    for point in points {
        let pixel = world.sketch_pixel_of(point.position);
        match pixel {
            FIRST => assert!(geometry.is_center_point(point.id), "first point role"),
            SECOND => assert!(!geometry.is_center_point(point.id), "second point role"),
            _ => panic!("point at {pixel:?}: expected {FIRST:?} or {SECOND:?}"),
        }
    }
}
// point-10
#[when("I click the same canvas location twice and press Escape")]
fn click_twice(world: &mut ApplicationWorld) {
    world.sketch_click(FIRST);
    world.sketch_click(FIRST);
    world.escape();
    world.pump(Vec::new());
}
#[then("no validation dialog appears")]
fn no_dialog(world: &mut ApplicationWorld) {
    world.expect_no_text("Warning");
}
#[then("the sketch shows 210 dimensions needed")]
#[then("the sketch still shows 210 dimensions needed")]
fn shows_210(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(210);
}
#[when("I undo the most recent placement")]
#[when("I undo the earlier placement")]
fn undo_placement(world: &mut ApplicationWorld) {
    world.undo();
}
#[then("the sketch shows 208 dimensions needed")]
fn shows_208(world: &mut ApplicationWorld) {
    world.expect_dimensions_needed(208);
}
#[when("I redo each placement separately")]
fn redo_each(world: &mut ApplicationWorld) {
    world.observed_counts.clear();
    for _ in 0..2 {
        world.redo();
        world.observed_counts.push(world.status_dimensions_needed());
    }
}
#[then("the count changes to 210 and remains at 210")]
fn count_210(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_counts, [210, 210], "counts after each redo");
}
#[when("I open Select Other at the overlapping location and expand its candidate list")]
fn open_select_other(world: &mut ApplicationWorld) {
    open_context_menu(world, FIRST);
    world.click_text("Select Other");
    world.pump(Vec::new());
}
#[then("the list contains two point candidates")]
fn two_candidates(world: &mut ApplicationWorld) {
    // The list opens to the right of its row; the ribbon's own Point sits above the canvas.
    let row = world.run("Select Other").clone();
    let candidates = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Point" && run.minimum[0] > row.maximum[0])
        .filter(|run| run.minimum[1] > row.minimum[1] - CANDIDATE_LIST_ABOVE_ROW)
        .count();
    assert_eq!(candidates, 2, "candidates in {:?}", world.text());
}

// point-12
#[when("the user expand the Create panel")]
fn expand_create_for_point(world: &mut ApplicationWorld) {
    super::super::ribbon_panel_expansion::click_expansion_arrow(world, "Create");
    world.pump(Vec::new());
}
#[then("Precise Input is available")]
fn precise_input_available_for_point(world: &mut ApplicationWorld) {
    let command = world.run_in_window("##ribbon-additional-Create", "PreciseInput");
    assert_eq!(command.alpha, u8::MAX, "expected Precise Input enabled");
    world.expect_status("Select center point");
}

#[cfg(target_os = "linux")]
#[test]
fn native_point_acquisition_honors_disabled_inference_settings() {
    use crate::HeadSketchWork;
    use oblikovati_api::{client::Client, wire::SetSketchSettingsArgs};
    use oblikovati_model::sketch::{PlanarConstraint, SketchPrimitiveCreation};
    use oblikovati_viewport::ViewportSketchPreview;
    for (infer, auto_apply, persist) in [
        (false, true, false),
        (true, false, false),
        (true, true, false),
    ] {
        let mut world = ApplicationWorld::new();
        world.start_empty_sketch();
        let edit = world.app().owner.active_sketch_edit().unwrap();
        let source = world
            .app()
            .owner
            .document_commands(edit.document)
            .create_sketch_primitive(
                edit.sketch,
                &SketchPrimitiveCreation::Point {
                    position: [2.0, 2.0],
                },
            )
            .unwrap()
            .entity;
        let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
        settings.infer_constraints = infer;
        settings.auto_apply_constraints = auto_apply;
        settings.persist_inferred_constraints = persist;
        Client::new(world.app().owner.document_host())
            .documents()
            .set_sketch_settings(&SetSketchSettingsArgs {
                document: 0,
                settings,
            })
            .unwrap();
        world.pump(Vec::new());
        let point_button = super::super::diagnostics::pointer_pixels(world.run("Point").center());
        super::super::x11_input::click(point_button);
        world.pump(Vec::new());
        let [x, y] = world.sketch_pixel_of([2.0, 2.0]);
        let click = [x + 5, y];
        let requested = world.sketch_point_of(click);
        let expected = if infer && auto_apply {
            [2.0, 2.0]
        } else {
            requested
        };
        let before = world.edited_geometry();
        super::super::x11_input::move_pointer(click);
        world.pump(Vec::new());
        let Some(ViewportSketchPreview::Resolved {
            geometry, members, ..
        }) = world.app().owner.sketch_preview()
        else {
            panic!("expected resolved point preview");
        };
        assert_eq!(members.len(), 1);
        assert_eq!(geometry.point(members[0]).unwrap().position, expected);
        let preview_coincidence = geometry.constraints().iter().any(|entry| matches!(entry.relation,
            PlanarConstraint::Coincident { points } if points.contains(&source) && points.contains(&members[0])
        ));
        assert_eq!(preview_coincidence, infer && auto_apply);
        super::super::x11_input::click(click);
        world.pump(Vec::new());
        super::super::x11_input::press_escape();
        world.pump(Vec::new());
        let geometry = world.edited_geometry();
        let placed = geometry
            .points()
            .iter()
            .find(|point| point.id != source && point.position == expected)
            .expect("expected the new point at the requested or acquired position");
        let coincidence = geometry.constraints().iter().any(|entry| matches!(entry.relation,
            PlanarConstraint::Coincident { points } if points.contains(&source) && points.contains(&placed.id)
        ));
        assert_eq!(coincidence, infer && auto_apply);
        let after = geometry.clone();
        world.undo();
        assert!(before.same_recipe(&world.edited_geometry()));
        world.redo();
        assert!(after.same_recipe(&world.edited_geometry()));
        world.close();
    }
}
