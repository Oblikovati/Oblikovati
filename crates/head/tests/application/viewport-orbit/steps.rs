// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-orbit`: free and constrained rotation, and turns by an angle.
use super::viewport_orientation::{open_block, solid_extent};
use super::viewport_pan::NavigationControl;
use super::viewport_pan::{camera, press_bar};
use super::{ApplicationWorld, capture_framebuffer, diagnostics};
use cucumber::{given, then, when};
#[path = "fit_region.rs"]
mod fit_region;
#[path = "saved_block.rs"]
mod saved_block;

/// The guide's colour, as it blends over the view; the control's faces are darker.
const GUIDE_COLOR: [i32; 3] = [200, 200, 212];
const GUIDE_REACH: i32 = 12;
const GUIDE_RADIUS_SHARE: f32 = 0.35;
const GUIDE_SAMPLE_REACH: usize = 3;
const DRAG: [[i32; 2]; 2] = [[500, 400], [560, 440]];
const INCREMENT_FIELD_VALUE: &str = "10";
const SAME_VIEW: f64 = 1e-6;

fn ring_quadrants(world: &mut ApplicationWorld) -> [bool; 4] {
    let [left, top, right, bottom] = world.viewport_bounds().map(|edge| edge as f32);
    let center = [(left + right) / 2., (top + bottom) / 2.];
    let diagonal =
        (right - left).min(bottom - top) * GUIDE_RADIUS_SHARE * std::f32::consts::FRAC_1_SQRT_2;
    let capture = capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    // Sample the ring well away from the solid's projected edges near the center.
    [[-1., -1.], [1., -1.], [1., 1.], [-1., 1.]].map(|[across, down]| {
        let x = (center[0] + across * diagonal).round() as usize;
        let y = (center[1] + down * diagonal).round() as usize;
        (y - GUIDE_SAMPLE_REACH..=y + GUIDE_SAMPLE_REACH).any(|row| {
            (x - GUIDE_SAMPLE_REACH..=x + GUIDE_SAMPLE_REACH).any(|column| {
                let at = (row * width + column) * 4;
                (0..3).all(|channel| {
                    (i32::from(capture.rgba[at + channel]) - GUIDE_COLOR[channel]).abs()
                        < GUIDE_REACH
                })
            })
        })
    })
}
fn expect_guide(world: &mut ApplicationWorld, shown: bool) {
    assert_eq!(ring_quadrants(world), [shown; 4], "orbit guide ring");
}
/// The bar's second dropdown belongs to Orbit.
fn open_orbit_menu(world: &mut ApplicationWorld) {
    let arrow = super::viewport_pan::bar_dropdown_center(NavigationControl::Orbit);
    world.click(super::diagnostics::pointer_pixels(arrow));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn choose(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[when("I select the orbit control on the navigation bar")]
#[when("the user selects the orbit control on the navigation bar")]
fn select_orbit(world: &mut ApplicationWorld) {
    press_bar(world, NavigationControl::Orbit);
}
#[then("the status area indicates free rotation")]
fn free_rotation(world: &mut ApplicationWorld) {
    world.expect_status("Free Rotation");
}
#[when("I move through the viewport using a drag gesture")]
#[when("the user moves through the viewport using a drag gesture")]
fn drag_view(world: &mut ApplicationWorld) {
    let before = camera(world);
    world.drag(DRAG[0], DRAG[1]);
    world.pump(Vec::new());
    assert_ne!(camera(world).eye, before.eye, "orbit must rotate the view");
}
#[then("a circular guide with a central cross and radial marks is visible over the viewport")]
#[then("a circular rotation guide appears over the viewport")]
fn guide_visible(world: &mut ApplicationWorld) {
    expect_guide(world, true);
    world.save_capture("orbit-guide-visible");
}
#[when("I open the context menu in the viewport")]
#[when("the user opens the context menu in the viewport")]
fn open_view_menu(world: &mut ApplicationWorld) {
    world.right_click(super::viewport_pan::EMPTY_VIEW);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("it offers Done, OK, Home View, Pan, Zoom and Previous View")]
fn orbit_rows(world: &mut ApplicationWorld) {
    for row in ["Done", "OK", "Home View", "Pan", "Zoom", "Previous View"] {
        world.expect_text(row);
    }
}
#[then("the circular guide disappears")]
#[then("the guide disappears")]
fn guide_gone(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    expect_guide(world, false);
}
// Constrained orbit
#[when("I open the orbit menu on the navigation bar")]
#[when("the user opens the orbit menu on the navigation bar")]
#[when("I reopen the orbit menu")]
#[when("the user reopens the orbit menu")]
fn orbit_menu(world: &mut ApplicationWorld) {
    open_orbit_menu(world);
}
#[then("the menu offers Orbit, Constrained Orbit and Rotate at Angle")]
fn orbit_menu_rows(world: &mut ApplicationWorld) {
    for row in ["Orbit", "Constrained Orbit", "Rotate at Angle"] {
        world.expect_text(row);
    }
}
#[when("I choose Constrained Orbit")]
#[when("the user chooses Constrained Orbit")]
fn choose_constrained(world: &mut ApplicationWorld) {
    choose(world, "Constrained Orbit");
}
#[then("Constrained Orbit is marked as the selected action")]
fn constrained_marked(world: &mut ApplicationWorld) {
    assert!(world.checked("Constrained Orbit"), "expected it marked");
}
// Rotate at Angle
#[given("a rectangular block is visible face-on in an active part in the MCAD Software")]
fn face_on_block(world: &mut ApplicationWorld) {
    open_block(world);
    world.observed_view_camera = Some(camera(world));
}
#[when("I choose Rotate at Angle from the orbit menu")]
#[when("the user chooses Rotate at Angle from the orbit menu")]
fn choose_rotate(world: &mut ApplicationWorld) {
    open_orbit_menu(world);
    choose(world, "Rotate at Angle");
}
#[then("an incremental view rotation dialog opens")]
fn rotate_dialog(world: &mut ApplicationWorld) {
    world.expect_text("Incremental View Rotate");
}
#[then("it contains an increment field, six direction buttons, Reset, OK and Cancel")]
fn rotate_controls(world: &mut ApplicationWorld) {
    for row in [
        "Increment",
        "X+",
        "X-",
        "Y+",
        "Y-",
        "Z+",
        "Z-",
        "Reset",
        "OK",
        "Cancel",
    ] {
        world.expect_text(row);
    }
}
#[then("the increment field displays 10")]
#[when("the increment field displays 10")]
fn increment_shown(world: &mut ApplicationWorld) {
    let shown = world
        .frame
        .runs
        .iter()
        .any(|run| run.text.starts_with(INCREMENT_FIELD_VALUE));
    assert!(shown, "expected {INCREMENT_FIELD_VALUE} in the field");
}
#[when("I click the first direction button")]
#[when("the user clicks the first direction button")]
#[when("I click the first direction button again")]
#[when("the user clicks the first direction button again")]
fn first_direction(world: &mut ApplicationWorld) {
    choose(world, "X+");
}

#[test]
fn rotate_title_close_restores_the_marked_view() {
    let mut world = ApplicationWorld::new();
    face_on_block(&mut world);
    choose_rotate(&mut world);
    first_direction(&mut world);
    view_changes(&mut world);
    let (position, size) = diagnostics::window_geometry("Incremental View Rotate").unwrap();
    let close = diagnostics::pointer_pixels([position[0] + size[0] - 13., position[1] + 10.]);
    world.click(close);
    world.pump(Vec::new());
    let before = world.observed_view_camera.unwrap();
    assert_eq!(camera(&mut world).eye, before.eye);
    assert!(diagnostics::window_geometry("Incremental View Rotate").is_none());
    world.close();
}
#[then("the visible view changes while the dialog remains open")]
fn view_changes(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_ne!(camera(world).eye, earlier.eye, "view");
    world.expect_text("Incremental View Rotate");
}
#[when("I click Reset")]
#[when("the user clicks Reset")]
fn click_reset(world: &mut ApplicationWorld) {
    choose(world, "Reset");
}
#[then("the original face-on view is restored")]
fn view_restored(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    let (before, after) = (screen_up(&earlier), screen_up(&now));
    let close = (0..3).all(|axis| {
        (now.eye[axis] - earlier.eye[axis]).abs() < SAME_VIEW
            && (after[axis] - before[axis]).abs() < SAME_VIEW
    });
    assert!(
        close,
        "eye {:?} up {after:?} from eye {:?} up {before:?}",
        now.eye, earlier.eye
    );
}
/// A camera's up as the screen shows it: square to the view, however it was given.
fn screen_up(camera: &oblikovati_viewport::ViewportCamera) -> [f64; 3] {
    let delta: [f64; 3] = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    let square = |value: [f64; 3]| value.iter().map(|part| part * part).sum::<f64>();
    let along: f64 = (0..3)
        .map(|axis| camera.up[axis] * delta[axis])
        .sum::<f64>()
        / square(delta);
    let up: [f64; 3] = std::array::from_fn(|axis| camera.up[axis] - delta[axis] * along);
    let length = square(up).sqrt();
    up.map(|part| part / length)
}
#[when("I click Cancel")]
fn click_cancel(world: &mut ApplicationWorld) {
    choose(world, "Cancel");
}
#[when("I click OK")]
#[when("the user clicks OK")]
#[when("the user chooses OK")]
fn click_ok(world: &mut ApplicationWorld) {
    #[cfg(target_os = "linux")]
    if world.state().sketch_text.is_some() {
        let button = world.run_in_window("Format Text", "OK").center();
        super::x11_input::click(diagnostics::pointer_pixels(button));
        world.pump(Vec::new());
        return;
    }
    if diagnostics::window_geometry("Derived Part").is_some() {
        let button = world.run_in_window("Derived Part", "OK").center();
        world.click(diagnostics::pointer_pixels(button));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    if diagnostics::window_geometry("Decal").is_some() {
        let button = world.run_in_window("Decal", "OK").center();
        world.click(diagnostics::pointer_pixels(button));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    if diagnostics::window_geometry("Import").is_some() {
        let button = world.run_in_window("Import", "OK").center();
        world.click(diagnostics::pointer_pixels(button));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    if diagnostics::window_geometry("Unwrap").is_some() {
        let button = world.run_in_window("Unwrap", "OK").center();
        world.click(diagnostics::pointer_pixels(button));
        world.pump(Vec::new());
        world.pump(Vec::new());
        return;
    }
    choose(world, "OK");
}
#[then("the changed view remains visible")]
fn change_stays(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_ne!(camera(world).eye, earlier.eye, "view");
    solid_extent(world);
}
// Reversing a turn
const ROTATE_BUTTONS: [&str; 6] = ["X+", "X-", "Y+", "Y-", "Z+", "Z-"];
fn press_direction(world: &mut ApplicationWorld, ordinal: usize) {
    choose(world, ROTATE_BUTTONS[ordinal - 1]);
}
#[given("a rectangular block is visible at an inclined orientation in the MCAD Software")]
fn inclined_block(world: &mut ApplicationWorld) {
    open_block(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.1 },
    )
    .unwrap();
    world.pump(Vec::new());
    fit_region::assert_fixture_in_document_area(world);
    world.observed_view_camera = Some(camera(world));
}
#[given("the incremental view rotation dialog is open with increment 10")]
fn rotate_open(world: &mut ApplicationWorld) {
    choose_rotate(world);
    increment_shown(world);
}
#[when("I click the second direction button")]
#[when("the user clicks the second direction button")]
fn second_direction(world: &mut ApplicationWorld) {
    press_direction(world, 2);
}
#[then("the block view changes")]
fn block_changes(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    let now = camera(world);
    assert!(now.eye != earlier.eye || now.up != earlier.up, "view");
}
#[when("I click the first direction button once")]
#[when("the user clicks the first direction button once")]
fn first_once(world: &mut ApplicationWorld) {
    press_direction(world, 1);
}
#[then("the block returns to its initial apparent orientation")]
#[then("the block remains visible in the same apparent orientation")]
fn orientation_returned(world: &mut ApplicationWorld) {
    view_restored(world);
    solid_extent(world);
}
/// Scenarios of several dialogs share these words: whichever is up must still be.
#[then("the dialog remains open")]
fn dialog_open(world: &mut ApplicationWorld) {
    assert!(
        world.shows("Incremental View Rotate") || world.shows("Look in"),
        "expected a dialog on screen in {:?}",
        world.text()
    );
}
#[then("the dialog closes and the status area displays Ready")]
fn dialog_closed_ready(world: &mut ApplicationWorld) {
    world.expect_no_text("Incremental View Rotate");
    world.expect_ready();
}
#[when(expr = "I click direction button {int} once")]
#[when(expr = "the user clicks direction button {int} once")]
fn numbered_direction(world: &mut ApplicationWorld, ordinal: usize) {
    press_direction(world, ordinal);
}
// Ways out of free orbit
#[when("I select Orbit on the navigation bar")]
#[when("the user selects Orbit on the navigation bar")]
fn select_orbit_named(world: &mut ApplicationWorld) {
    press_bar(world, NavigationControl::Orbit);
}
#[given("Orbit is active with its circular guide visible")]
fn orbit_active(world: &mut ApplicationWorld) {
    world.observed_view_camera = Some(camera(world));
    press_bar(world, NavigationControl::Orbit);
    expect_guide(world, true);
}
#[when("I click Pan on the navigation bar")]
#[when("the user clicks Pan on the navigation bar")]
fn click_pan(world: &mut ApplicationWorld) {
    press_bar(world, NavigationControl::Pan);
}
#[then("the circular orbit guide disappears")]
fn orbit_guide_gone(world: &mut ApplicationWorld) {
    expect_guide(world, false);
}
#[when("I open the orbit context menu")]
#[when("the user opens the orbit context menu")]
fn open_orbit_context(world: &mut ApplicationWorld) {
    open_view_menu(world);
}
