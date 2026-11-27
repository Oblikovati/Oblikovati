// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard and mouse coverage for the precise ellipse Move scenario.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) const CENTER: [i32; 2] = [850, 550];
pub(super) const EDGE: [i32; 2] = [1010, 470];

#[given("an editable sketch with eligible geometry")]
pub(super) fn editable_ellipse(world: &mut ApplicationWorld) {
    world.start_sketch_on("YZ Plane");
    world.choose_from_flyout("Circle", "Ellipse");
    for point in [CENTER, EDGE, [880, 610]] {
        world.sketch_click(point);
    }
    world.escape();
    world.remember_edited_sketch(Vec::new());
}

#[given("the ellipse is selected in Move with Copy off and Precise Input on")]
pub(super) fn select_ellipse(world: &mut ApplicationWorld) {
    open_move(world);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    world.sketch_click(EDGE);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    let reference = world.run_in_window("Precise Input", "Abs").center();
    world.click(diagnostics::pointer_pixels(reference));
    world.click_text("Relative");
}

#[given("the selected ellipse is at the precise Base Point stage")]
fn precise_base_point(world: &mut ApplicationWorld) {
    select_ellipse(world);
}

#[when("the user opens the coordinate-mode dropdown")]
fn open_coordinate_modes(world: &mut ApplicationWorld) {
    let button = world.run_in_window("Precise Input", "Mode").center();
    world.click(diagnostics::pointer_pixels(button));
}

#[then("four coordinate and angle combinations are listed")]
fn coordinate_modes(world: &mut ApplicationWorld) {
    for name in [
        "X and Y coordinate",
        "X coordinate and X axis angle",
        "Y coordinate and X axis angle",
        "Distance and X axis angle",
    ] {
        world.expect_text(name);
    }
}

#[when("I choose Distance and X axis angle")]
#[when("the user chooses Distance and X axis angle")]
fn distance_angle(world: &mut ApplicationWorld) {
    world.click_text("Distance and X axis angle");
}

#[then("the toolbar exposes Distance in mm and Angle in degrees")]
fn distance_angle_units(world: &mut ApplicationWorld) {
    let text = world.window_text("Precise Input");
    assert!(
        text.contains("Distance") && text.contains("Angle") && text.contains("mm/deg"),
        "{text:?}"
    );
}

#[given("the selected ellipse is in precise Base Point mode")]
fn reference_base_point(world: &mut ApplicationWorld) {
    select_ellipse(world);
}

#[when("I open the left toolbar dropdown")]
#[when("the user opens the left toolbar dropdown")]
fn reference_menu(world: &mut ApplicationWorld) {
    let current = if world.window_text("Precise Input").contains("Rel") {
        "Rel"
    } else {
        "Abs"
    };
    let button = world.run_in_window("Precise Input", current).center();
    world.click(diagnostics::pointer_pixels(button));
}

#[then("Relative and Absolute choices are visible")]
fn reference_choices(world: &mut ApplicationWorld) {
    world.expect_text("Relative");
    world.expect_text("Absolute");
}

#[when("I choose Absolute")]
#[when("the user chooses Absolute")]
fn choose_absolute(world: &mut ApplicationWorld) {
    world.click_text("Absolute");
}

#[then("the visible triad is at the sketch axes intersection")]
#[then("the triad remains at the axes intersection")]
fn triad_at_origin(world: &mut ApplicationWorld) {
    let origin = world.sketch_pixel_of([0., 0.]);
    world.expect_segment_drawn(origin, [origin[0] + 24, origin[1]]);
    assert_eq!(
        world.app().owner.precise_triad_position().unwrap().1,
        [0., 0.]
    );
}

#[then("the selected ellipse retains its recorded position")]
fn reference_retains_ellipse(world: &mut ApplicationWorld) {
    original_only(world);
}

#[when("I reopen the dropdown and choose Relative")]
#[when("the user reopens the dropdown and chooses Relative")]
fn choose_relative(world: &mut ApplicationWorld) {
    reference_menu(world);
    world.click_text("Relative");
}

#[then("the toolbar icon changes")]
fn relative_icon(world: &mut ApplicationWorld) {
    world.run_in_window("Precise Input", "Rel");
}

#[when("I choose Base Point and click its center")]
#[when("the user chooses Base Point and clicks its center")]
fn choose_base(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
}

#[then("the coordinate toolbar appears and the status requests an endpoint")]
fn endpoint_requested(world: &mut ApplicationWorld) {
    world.run_in_window("Precise Input", "Rel");
    world.expect_status("Move: Select end point");
}

#[when("I type 1 then 0 into X and Tab to Y")]
#[when("the user types 1 then 0 into X and presses Tab to Y")]
fn type_x(world: &mut ApplicationWorld) {
    world.click_precise_field("X");
    world.type_text("10");
    world.key(ViewportPhysicalKey::Tab, false);
}

#[when("I type 0 and press Enter")]
#[when("the user types 0 and presses Enter")]
fn type_y(world: &mut ApplicationWorld) {
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("Move returns to geometry selection without an error dialog")]
fn move_ready(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    world.expect_status("Move: Select geometry");
}

#[then("the ellipse appears above its original position")]
fn ellipse_moved(world: &mut ApplicationWorld) {
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    let center = before.ellipses()[0].center();
    let original = before.point(center).unwrap().position;
    let moved = world.edited_geometry().point(center).unwrap().position;
    assert!(
        (moved[0] - original[0] - 1.).abs() < 1e-8 && (moved[1] - original[1]).abs() < 1e-8,
        "expected a 10 mm X displacement, got {original:?} -> {moved:?}"
    );
    let old_pixel = world.sketch_pixel_of(original);
    let new_pixel = world.sketch_pixel_of(moved);
    assert!(
        new_pixel[1] < old_pixel[1],
        "ellipse center {old_pixel:?} -> {new_pixel:?}"
    );
    world.observed_view = Some(new_pixel);
}

#[then("Move closes and the displaced ellipse remains")]
fn finished(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_move.is_none());
    let geometry = world.edited_geometry();
    let center = geometry.ellipses()[0].center();
    assert_eq!(
        world.sketch_pixel_of(geometry.point(center).unwrap().position),
        world.observed_view.unwrap()
    );
}

#[given("Move Copy is checked and the original ellipse is selected")]
fn select_copy(world: &mut ApplicationWorld) {
    open_move(world);
    enable_copy_and_select(world);
}

#[when("I enable Copy and select the original ellipse")]
#[when("the user enables Copy and selects the original ellipse")]
fn enable_copy_and_select(world: &mut ApplicationWorld) {
    let check = world.run_in_window("Move", "Copy").center();
    world.click(diagnostics::pointer_pixels(check));
    assert!(world.state().sketch_move.unwrap().copy);
    world.sketch_click(EDGE);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
}

#[given("Move has reopened with Precise Input checked and Copy unchecked")]
fn reopened_move(world: &mut ApplicationWorld) {
    open_move(world);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    done(world);
    open_move(world);
    assert!(!world.state().sketch_move.unwrap().copy);
}

#[when("I choose its center as Base Point and click the displaced endpoint")]
#[when("the user chooses its center as Base Point and clicks the displaced endpoint")]
fn copy_from_center(world: &mut ApplicationWorld) {
    place_copy(world);
}

#[then("Move returns to geometry selection")]
fn copy_returns_to_selection(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
}

#[then("a displaced ellipse and the original ellipse are both visible")]
#[then("both ellipse outlines remain")]
fn copy_outlines(world: &mut ApplicationWorld) {
    both_ellipses(world);
}

#[when("the base and displaced endpoint are clicked")]
fn place_copy(world: &mut ApplicationWorld) {
    world.sketch_click(CENTER);
    let endpoint = [CENTER[0] + 110, CENTER[1] - 100];
    world.hover(endpoint);
    let preview = crate::HeadSketchWork::sketch_preview(&world.app().owner).unwrap();
    assert!(
        matches!(preview, oblikovati_viewport::ViewportSketchPreview::Resolved { members, .. } if members.len() == 1)
    );
    assert_eq!(world.edited_geometry().ellipses().len(), 1);
    world.expect_segment_drawn(EDGE, EDGE);
    let copy_edge = [EDGE[0] + 110, EDGE[1] - 100];
    world.expect_segment_drawn(copy_edge, copy_edge);
    world.sketch_click(endpoint);
}

#[then("a second ellipse appears while the original remains")]
#[then("both outlines remain visible")]
fn both_ellipses(world: &mut ApplicationWorld) {
    let original = &world.observed_sketch.as_ref().unwrap().geometry;
    let old = original.ellipses()[0];
    let old_center = original.point(old.center()).unwrap().position;
    let geometry = world.edited_geometry();
    assert_eq!(geometry.ellipses().len(), 2);
    assert_eq!(geometry.point(old.center()).unwrap().position, old_center);
    assert_ne!(
        geometry
            .point(geometry.ellipses()[1].center())
            .unwrap()
            .position,
        old_center
    );
    world.expect_segment_drawn(EDGE, EDGE);
    world.expect_segment_drawn(
        [EDGE[0] + 110, EDGE[1] - 100],
        [EDGE[0] + 110, EDGE[1] - 100],
    );
}

#[when("Done is clicked")]
fn done(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Parameters").is_some() {
        let done = world.run_in_window("Parameters", "Done").center();
        world.click(diagnostics::pointer_pixels(done));
        world.pump(Vec::new());
        assert!(diagnostics::window_geometry("Parameters").is_none());
        return;
    }
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    assert!(world.state().sketch_move.is_none());
}

#[cfg(target_os = "linux")]
#[test]
fn native_done_closes_move_without_changing_geometry() {
    let mut world = ApplicationWorld::new();
    editable_ellipse(&mut world);
    open_move(&mut world);
    let done = world.run_in_window("Move", "Done").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(done));
    world.pump(Vec::new());
    assert!(world.state().sketch_move.is_none());
    let original = world.observed_sketch.as_ref().unwrap().geometry.clone();
    assert!(world.edited_geometry().same_recipe(&original));
    world.close();
}

#[given("Move Copy has created the displaced ellipse and Move is closed")]
fn copied_ellipse(world: &mut ApplicationWorld) {
    select_copy(world);
    place_copy(world);
    both_ellipses(world);
    done(world);
}

#[then("only the original ellipse remains in the pair")]
#[then("the trial copy is removed again")]
fn original_only(world: &mut ApplicationWorld) {
    let current = world.edited_geometry();
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    assert!(current.same_recipe(before));
}

#[then("the displaced ellipse returns")]
fn copied_again(world: &mut ApplicationWorld) {
    both_ellipses(world);
}

#[test]
fn cancelling_copy_preview_keeps_the_source_and_history() {
    let mut world = ApplicationWorld::new();
    editable_ellipse(&mut world);
    let history = world.state().undo_label;
    select_copy(&mut world);
    world.sketch_click(CENTER);
    world.hover([CENTER[0] + 110, CENTER[1] - 100]);
    assert!(crate::HeadSketchWork::sketch_preview(&world.app().owner).is_some());
    world.escape();
    assert!(crate::HeadSketchWork::sketch_preview(&world.app().owner).is_none());
    original_only(&mut world);
    assert_eq!(world.state().undo_label, history);
    open_move(&mut world);
    assert!(world.state().sketch_move.unwrap().copy);
    world.close();
}

#[test]
fn relative_pointer_values_accept_the_displayed_endpoint() {
    let mut world = ApplicationWorld::new();
    editable_ellipse(&mut world);
    let before = world.edited_geometry();
    let center = before.ellipses()[0].center();
    select_ellipse(&mut world);
    choose_base(&mut world);
    let endpoint = [CENTER[0] + 110, CENTER[1] - 100];
    world.hover(endpoint);
    let target = world.state().precise_target.unwrap();
    let shown = target.relative_pointer.unwrap();
    assert_ne!(Some(shown.clone()), target.pointer);
    let text = world.window_text("Precise Input");
    assert!(
        text.contains(&shown[0]) && text.contains(&shown[1]),
        "expected {shown:?} in {text:?}"
    );
    world.click_precise_field("Y");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    let geometry = world.edited_geometry();
    let center = geometry.point(center).unwrap().position;
    let pixel = world.sketch_pixel_of(center);
    assert!(
        (pixel[0] - endpoint[0]).abs() <= 1 && (pixel[1] - endpoint[1]).abs() <= 1,
        "expected {endpoint:?}, observed {pixel:?}"
    );
    world.close();
}

/// Opens Move through the ribbon, expanding the Modify panel when needed.
/// For example, `open_move(world)` also works when Move is already visible.
pub(super) fn open_move(world: &mut ApplicationWorld) {
    if !world.frame.runs.iter().any(|run| run.text == "Move") {
        open_modify_panel(world);
    }
    world.click_text("Move");
    assert!(world.state().sketch_move.is_some(), "Move did not open");
}

#[test]
fn move_opens_from_the_expanded_modify_panel() {
    let mut world = ApplicationWorld::new();
    editable_ellipse(&mut world);
    if !world.frame.runs.iter().any(|run| run.text == "Move") {
        open_modify_panel(&mut world);
    }
    world.expect_text("Move");
    open_move(&mut world);
    world.close();
}

#[test]
fn move_opens_from_the_collapsed_modify_panel() {
    let mut world = ApplicationWorld::new();
    editable_ellipse(&mut world);
    assert!(
        !world.frame.runs.iter().any(|run| run.text == "Move"),
        "expected a collapsed Modify panel"
    );
    open_move(&mut world);
    world.close();
}

fn open_modify_panel(world: &mut ApplicationWorld) {
    world.click(collapsed_modify_button(world));
}

pub(super) fn collapsed_modify_button(world: &ApplicationWorld) -> [i32; 2] {
    let group = world.run("Modify");
    diagnostics::pointer_pixels(group.center())
}

#[given("the precise ellipse Move has been accepted and Move closed")]
fn accepted_precise_move(world: &mut ApplicationWorld) {
    select_ellipse(world);
    choose_base(world);
    type_x(world);
    type_y(world);
    move_ready(world);
    ellipse_moved(world);
    done(world);
}

#[when("the user undoes")]
fn undo_move(world: &mut ApplicationWorld) {
    world.undo();
}

#[when("the user redoes")]
fn redo_move(world: &mut ApplicationWorld) {
    world.redo();
}

#[then("the ellipse returns to its original screen position")]
#[then("the original fixture position is restored again")]
fn original_precise_position(world: &mut ApplicationWorld) {
    original_only(world);
    world.expect_segment_drawn(EDGE, EDGE);
}

#[then("the ellipse returns to its displaced screen position")]
fn displaced_precise_position(world: &mut ApplicationWorld) {
    ellipse_moved(world);
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    let original = before
        .point(before.ellipses()[0].center())
        .unwrap()
        .position;
    let old_pixel = world.sketch_pixel_of(original);
    let new_pixel = world.observed_view.unwrap();
    let edge = [
        EDGE[0] + new_pixel[0] - old_pixel[0],
        EDGE[1] + new_pixel[1] - old_pixel[1],
    ];
    world.expect_segment_drawn(edge, edge);
}
