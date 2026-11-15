// SPDX-License-Identifier: GPL-2.0-only
//! Orientation scenarios on what the options change in use: north, fit, upright, corners, lock.
use super::appearance::{
    choose_mode, compass_shown, expect_no_compass, open_menu, part_top_view, toggle_compass,
};
use super::*;
// North angle
fn enter_north_angle(world: &mut ApplicationWorld, degrees: &str) {
    let field = world.run("Angle of North").clone();
    world.click([field.minimum[0] as i32 - 60, field.center()[1] as i32]);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(degrees);
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}
fn compass_letter(world: &ApplicationWorld, letter: &str) -> [f32; 2] {
    compass_observation::compass_label(world, letter)
        .unwrap_or_else(|| panic!("expected the compass letter {letter}"))
        .center()
}
fn control_center() -> [f32; 2] {
    let (position, parts) = layout();
    [position[0] + parts.center[0], position[1] + parts.center[1]]
}
#[given("a rectangular block is active in the MCAD Software in Top view")]
fn block_in_top(world: &mut ApplicationWorld) {
    part_top_view(world);
}
#[given("the compass is disabled and its north angle is 0 degrees")]
fn compass_at_zero(world: &mut ApplicationWorld) {
    expect_no_compass(world);
}
#[when("I enable the compass and enter 90 degrees as the north angle")]
#[when("the user enables the compass and enters 90 degrees as the north angle")]
fn enable_with_ninety(world: &mut ApplicationWorld) {
    click_row(world, "Show the compass");
    enter_north_angle(world, "90");
}
#[when("I confirm the options")]
#[when("the user confirms the options")]
fn confirm_options(world: &mut ApplicationWorld) {
    confirm(world);
}
#[then("the compass is visible around the orientation control")]
fn compass_visible(world: &mut ApplicationWorld) {
    compass_shown(world);
}
#[then("its north indicator appears to the right of the Top face")]
fn north_right(world: &mut ApplicationWorld) {
    let [x, y] = compass_letter(world, "N");
    let [middle, level] = control_center();
    assert!(
        x > middle && (y - level).abs() < 6.,
        "N at {x},{y} from {middle},{level}"
    );
}
#[then("the block remains in Top view")]
fn block_stays_top(world: &mut ApplicationWorld) {
    expect_control_shows(world, "Top");
}
#[then("the compass is enabled and the north angle is 90 degrees")]
fn ninety_shown(world: &mut ApplicationWorld) {
    assert!(world.checked("Show the compass"), "expected it checked");
    let shown = world
        .frame
        .runs
        .iter()
        .any(|run| run.text.starts_with("90"));
    assert!(shown, "expected 90 in the angle field");
}
#[when("I change the north angle to 0 degrees and confirm")]
#[when("the user changes the north angle to 0 degrees and confirms")]
fn change_to_zero(world: &mut ApplicationWorld) {
    enter_north_angle(world, "0");
    confirm(world);
}
#[then("the compass north indicator appears above the Top face")]
fn north_above(world: &mut ApplicationWorld) {
    let [x, y] = compass_letter(world, "N");
    let [middle, level] = control_center();
    assert!(
        y < level && (x - middle).abs() < 6.,
        "N at {x},{y} from {middle},{level}"
    );
}
#[when("I disable the compass in the orientation control options and confirm")]
#[when("the user disables the compass in the orientation control options and confirms")]
fn disable_compass(world: &mut ApplicationWorld) {
    open_options(world);
    toggle_compass(world);
}
#[then("the compass is hidden and the block remains visible")]
fn compass_hidden(world: &mut ApplicationWorld) {
    expect_no_compass(world);
    solid_extent(world);
}
// Fit on change
const MAGNIFIED: f64 = 10.;
const WINDOW_WIDTH: usize = 1440;
fn set_fit_option(world: &mut ApplicationWorld) {
    open_options(world);
    click_row(world, "Fit to view on change");
    confirm(world);
}
fn magnify(world: &mut ApplicationWorld) {
    let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    crate::HeadWindow::set_view_camera(
        &mut world.app().window,
        oblikovati_viewport::ViewportCamera {
            scale: camera.scale / MAGNIFIED,
            ..camera
        },
    )
    .unwrap();
    world.pump(Vec::new());
}
/// A magnified block runs off the window's right edge; a fitted one stops short of it.
pub(in super::super) fn whole_block_visible(world: &mut ApplicationWorld) -> bool {
    let (_, high) = solid_bounds(world);
    high[0] + 1 < WINDOW_WIDTH
}
fn press_top_rim(world: &mut ApplicationWorld) {
    press_top_in_iso(world);
}
#[given("fit to view on orientation change is enabled")]
fn fit_enabled(world: &mut ApplicationWorld) {
    set_fit_option(world);
}
#[given("the block is magnified until its face extends beyond the viewport")]
#[when("I magnify the Front view until the block extends beyond the viewport")]
#[when("the user magnifies the Front view until the block extends beyond the viewport")]
fn magnified(world: &mut ApplicationWorld) {
    magnify(world);
    assert!(!whole_block_visible(world), "expected the block magnified");
}
/// The rim at the screen's bottom picks the face below: from Top, that is Front.
#[when("I use the orientation control to change to Front view")]
#[when("the user uses the orientation control to change to Front view")]
fn change_to_front(world: &mut ApplicationWorld) {
    let (position, parts) = layout();
    press_at(
        world,
        position,
        [
            parts.center[0],
            parts.center[1] + parts.half * IN_THE_CORNER,
        ],
    );
    world.pump(Vec::new());
    expect_control_shows(world, "Front");
}
#[when("I use the orientation control to change to Top view")]
#[when("the user uses the orientation control to change to Top view")]
fn change_to_top(world: &mut ApplicationWorld) {
    press_top_rim(world);
    expect_control_shows(world, "Top");
}
#[then("the whole block is visible within the viewport")]
fn whole_visible(world: &mut ApplicationWorld) {
    assert!(
        whole_block_visible(world),
        "expected the whole block in view"
    );
}
#[when("I disable fit to view on orientation change in the orientation control options")]
#[when("the user disables fit to view on orientation change in the orientation control options")]
#[when("I re-enable fit to view on orientation change")]
#[when("the user re-enables fit to view on orientation change")]
fn toggle_fit(world: &mut ApplicationWorld) {
    set_fit_option(world);
}
#[then("the block remains magnified beyond the viewport")]
fn stays_magnified(world: &mut ApplicationWorld) {
    assert!(
        !whole_block_visible(world),
        "expected the block still magnified"
    );
}
// Keep model upright
const KEEP_UPRIGHT: &str = "Keep model upright";
#[given("a rectangular block is active in the MCAD Software in upright Front view")]
fn block_upright_front(world: &mut ApplicationWorld) {
    open_block(world);
    front_displayed(world);
}
#[given("the orientation control option to keep the model upright is disabled")]
fn upright_off(world: &mut ApplicationWorld) {
    open_options(world);
    assert!(!world.checked(KEEP_UPRIGHT), "expected it unchecked");
    click_row(world, "Cancel");
}
#[when("I enable keep model upright and confirm the options")]
#[when("the user enables keep model upright and confirms the options")]
fn enable_upright(world: &mut ApplicationWorld) {
    open_options(world);
    click_row(world, KEEP_UPRIGHT);
    confirm(world);
}
#[then("the Front view and its label rotate by a quarter turn in the screen plane")]
fn turned_with_upright(world: &mut ApplicationWorld) {
    label_turned(world);
    taller(world);
}
#[then("the rectangular block remains visible")]
#[then("the active part remains visible")]
fn block_visible(world: &mut ApplicationWorld) {
    solid_extent(world);
}
#[then("keep model upright is checked")]
fn upright_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(KEEP_UPRIGHT), "expected it checked");
}
#[when("I clear keep model upright and confirm the options")]
#[when("the user clears keep model upright and confirms the options")]
fn clear_upright(world: &mut ApplicationWorld) {
    click_row(world, KEEP_UPRIGHT);
    confirm(world);
}
// The lower corners
fn set_position(world: &mut ApplicationWorld, corner: &str) {
    open_options(world);
    choose_list_row(world, "On-screen Position", corner);
    confirm(world);
}
fn expect_lower(side_right: bool) {
    let (control, _) = control_geometry();
    let (bar, _) = diagnostics::window_geometry(NAVIGATION).expect("expected the navigation bar");
    assert!(control[1] > 500., "control {control:?}");
    assert!(bar[1] < control[1], "bar {bar:?} above control {control:?}");
    assert_eq!(control[0] > 700., side_right, "control {control:?}");
    assert_eq!(bar[0] > 700., side_right, "bar {bar:?}");
}
#[given("the orientation control is in the Top Right position")]
fn in_top_right(world: &mut ApplicationWorld) {
    expect_on_right(world);
}
#[when("I set the orientation control position to Bottom Right and confirm")]
#[when("the user sets the orientation control position to Bottom Right and confirms")]
fn to_bottom_right(world: &mut ApplicationWorld) {
    set_position(world, "Bottom Right");
}
#[then("the orientation control appears near the lower-right viewport corner")]
#[then("the navigation bar appears above it on the right")]
fn lower_right(_: &mut ApplicationWorld) {
    expect_lower(true);
}
#[then("the part view remains visible")]
fn view_remains(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_blank_region([400, 600], [250, 300]);
}
#[then("the position is Bottom Right")]
fn position_shown(world: &mut ApplicationWorld) {
    world.expect_text("Bottom Right");
}
#[when("I set the position to Bottom Left and confirm")]
#[when("the user sets the position to Bottom Left and confirms")]
fn to_bottom_left(world: &mut ApplicationWorld) {
    choose_list_row(world, "On-screen Position", "Bottom Left");
    confirm(world);
}
#[then("the orientation control appears near the lower-left viewport corner")]
#[then("the navigation bar appears above it on the left")]
fn lower_left(_: &mut ApplicationWorld) {
    expect_lower(false);
}
#[when("I restore the position to Top Right and confirm")]
#[when("the user restores the position to Top Right and confirms")]
fn restore_top_right(world: &mut ApplicationWorld) {
    set_position(world, "Top Right");
}
// Lock to the current selection
const LOCK_ROW: &str = "Lock to Current Selection";
fn close_menu(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
fn select_mode(world: &mut ApplicationWorld, label: &str) {
    world.click_text("Select");
    world.click_text(label);
    world.pump(Vec::new());
}
fn selection_mode_label(mode: oblikovati_gui::GuiSelectionMode) -> &'static str {
    use oblikovati_gui::GuiSelectionMode;
    match mode {
        GuiSelectionMode::All => "All Types",
        GuiSelectionMode::Faces => "Faces",
        GuiSelectionMode::Components => "Components",
        GuiSelectionMode::Custom => panic!("custom selection mode: expected a restorable preset"),
    }
}
fn selection_order(
    state: &oblikovati_gui::GuiCommandState,
) -> Vec<oblikovati_gui::GuiSelectionKind> {
    state
        .selection_filter
        .as_ref()
        .expect("expected selection filter")
        .entries
        .iter()
        .map(|entry| entry.kind)
        .collect()
}
fn click_visible_face(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point = super::super::solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected a visible face patch outside GUI windows");
    world.click(point);
    world.pump(Vec::new());
    assert!(world.state().model_face_chosen, "expected a selected face");
}
fn click_block_face(world: &mut ApplicationWorld) {
    let before = world.state();
    let restore = selection_mode_label(before.selection_mode);
    select_mode(world, "Faces");
    click_visible_face(world);
    select_mode(world, restore);
    let after = world.state();
    assert_eq!(after.selection_mode, before.selection_mode);
    assert_eq!(selection_order(&after), selection_order(&before));
    assert!(after.model_face_chosen, "expected face selection retained");
}
#[given("no model geometry is selected")]
fn nothing_selected(world: &mut ApplicationWorld) {
    world.click(AWAY);
    world.pump(Vec::new());
    let state = world.state();
    let browser = state.browser.expect("expected a model browser");
    assert!(!state.model_face_chosen, "expected no face selection");
    assert!(browser.selected_body.is_none() && browser.selected_occurrence.is_none());
    assert!(browser.selected_sketches.is_empty() && browser.selected_feature.is_none());
}
#[then("Lock to Current Selection is unavailable")]
fn lock_off(world: &mut ApplicationWorld) {
    assert!(
        !world.run_is_enabled(LOCK_ROW),
        "expected {LOCK_ROW} unavailable"
    );
}
#[when("I close the menu and select a visible face of the block")]
#[when("the user closes the menu and selects a visible face of the block")]
fn select_a_face(world: &mut ApplicationWorld) {
    close_menu(world);
    click_block_face(world);
}
#[then("Lock to Current Selection is available")]
fn lock_on(world: &mut ApplicationWorld) {
    assert!(
        world.run_is_enabled(LOCK_ROW),
        "expected {LOCK_ROW} available"
    );
}
#[when("I choose Lock to Current Selection")]
#[when("the user chooses Lock to Current Selection")]
#[when("I choose Lock to Current Selection again")]
#[when("the user chooses Lock to Current Selection again")]
fn choose_lock(world: &mut ApplicationWorld) {
    choose_mode(world, LOCK_ROW);
}
#[then("a lock indicator appears beside the orientation control")]
fn lock_shown(world: &mut ApplicationWorld) {
    world.expect_text("Locked");
}
#[then("Lock to Current Selection is checked")]
fn lock_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(LOCK_ROW), "expected {LOCK_ROW} checked");
}
#[then("the lock indicator disappears")]
fn lock_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Locked");
}
#[when("I clear the face selection and reopen the orientation control context menu")]
#[when("the user clears the face selection and reopens the orientation control context menu")]
fn clear_face(world: &mut ApplicationWorld) {
    nothing_selected(world);
    open_menu(world);
}
// Set Current View as
#[when("I expand Set Current View as")]
#[when("the user expands Set Current View as")]
fn expand_set_view(world: &mut ApplicationWorld) {
    choose_mode(world, "Set Current View as");
}
#[then("the submenu offers Top and Front")]
fn top_and_front(world: &mut ApplicationWorld) {
    for row in ["Top", "Front"] {
        world.expect_text(row);
    }
}
#[when("I expand Set Current View as Home")]
#[when("the user expands Set Current View as Home")]
fn expand_set_home(world: &mut ApplicationWorld) {
    choose_mode(world, "Set Current View as Home");
}
#[then("the submenu offers Fixed Distance and Fit to View")]
fn home_rows(world: &mut ApplicationWorld) {
    for row in ["Fixed Distance", "Fit to View"] {
        world.expect_text(row);
    }
}
#[when("I press Escape without choosing a submenu action")]
#[when("the user presses Escape without choosing a submenu action")]
fn escape_menus(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the menus close and the same part view remains visible")]
fn menus_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Orthographic");
    world.expect_text("Origin");
    expect_same_part(world);
    world.expect_blank_region([400, 600], [250, 300]);
}
// Tiny
#[given("the orientation control size is Automatic")]
fn size_automatic(world: &mut ApplicationWorld) {
    automatic_size(world);
}
#[when("I set the orientation control size to Tiny and confirm")]
#[when("the user sets the orientation control size to Tiny and confirms")]
fn to_tiny(world: &mut ApplicationWorld) {
    open_options(world);
    choose_list_row(world, "Size", "Tiny");
    confirm(world);
}
#[then("the orientation control is visibly smaller in the same viewport")]
fn is_smaller(world: &mut ApplicationWorld) {
    let (_, size) = control_geometry();
    assert!(
        (size[0] as usize) < world.observed_counts[0],
        "side {}",
        size[0]
    );
}
#[then("the size is Tiny")]
fn size_is_tiny(world: &mut ApplicationWorld) {
    world.expect_text("Tiny");
}
#[when("I set the size to Automatic and confirm")]
#[when("the user sets the size to Automatic and confirms")]
fn to_automatic(world: &mut ApplicationWorld) {
    choose_list_row(world, "Size", "Automatic");
    confirm(world);
}
#[then("the orientation control returns to its earlier apparent size")]
fn apparent_size(world: &mut ApplicationWorld) {
    earlier_size(world);
}
#[then("the same part view remains visible")]
fn same_view(world: &mut ApplicationWorld) {
    expect_same_part(world);
    world.expect_blank_region([400, 600], [250, 300]);
}

#[test]
fn empty_selection_fixture_clears_a_real_face_and_can_reselect_it() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    click_block_face(&mut world);
    assert!(world.state().model_face_chosen);
    nothing_selected(&mut world);
    click_block_face(&mut world);
    assert!(world.state().model_face_chosen);
    world.close();
}
