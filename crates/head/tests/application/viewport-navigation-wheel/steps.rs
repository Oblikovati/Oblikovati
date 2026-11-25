// SPDX-License-Identifier: GPL-2.0-only
//! `features/application/viewport-navigation-wheel`: the ring of view tools and its options.
use super::viewport_orientation::solid_extent;
use super::viewport_pan::NavigationControl;
use super::viewport_pan::{camera, press_bar};
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

const WHEEL: &str = "##navigation-wheel";
const PLACED_AT: [i32; 2] = [760, 480];
const OPTIONS_TITLE: &str = "Navigation Wheel Options";
const MINI_SIDE: f32 = 80.;
const FULL_WHEEL_SIDE: f32 = 160.;
#[path = "big_size.rs"]
mod big_size;
#[path = "fixture.rs"]
mod fixture;
pub(super) use fixture::ensure_block;
#[cfg(test)]
use fixture::open_block;
#[path = "preferences.rs"]
mod preferences;
#[path = "segment_pixels.rs"]
mod segment_pixels;
#[cfg(test)]
#[path = "size_regression.rs"]
mod size_regression;
#[cfg(target_os = "linux")]
#[path = "wheel_controls.rs"]
mod wheel_controls;
/// How much further away the "small" block's view stands.
const SMALL: f64 = 5.;
const LAYOUTS: [&str; 6] = [
    "Mini View Object Wheel",
    "Mini Tour Building Wheel",
    "Mini Full Navigation Wheel",
    "Full Navigation Wheel",
    "Basic View Object Wheel",
    "Basic Tour Building Wheel",
];
fn settle(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn open_wheel(world: &mut ApplicationWorld) {
    press_bar(world, NavigationControl::Wheel);
    world.click(PLACED_AT);
    settle(world);
}
fn open_wheel_on_block(world: &mut ApplicationWorld) {
    ensure_block(world);
    world.observed_view_camera = Some(camera(world));
    open_full_wheel(world);
}
fn wheel_geometry() -> ([f32; 2], [f32; 2]) {
    diagnostics::window_geometry(WHEEL).expect("expected the navigation wheel")
}
fn right_click_wheel(world: &mut ApplicationWorld) {
    let (position, size) = wheel_geometry();
    let at = [
        (position[0] + size[0] / 2.) as i32,
        (position[1] + size[1] / 2.) as i32,
    ];
    world.right_click(at);
    settle(world);
}
fn choose_row(world: &mut ApplicationWorld, row: &str) {
    world.click_text(row);
    settle(world);
}
/// Tools across the ring can share a line, and a rendered run then holds them both.
fn expect_rows(world: &ApplicationWorld, rows: &[&str]) {
    let all: String = world
        .frame
        .runs
        .iter()
        .map(|run| run.text.as_str())
        .collect();
    for row in rows {
        assert!(
            all.contains(&row.replace(' ', "")),
            "expected {row:?} in {all:?}"
        );
    }
}
fn expect_marked(world: &ApplicationWorld, layout: &str) {
    assert!(world.checked(layout), "expected {layout} marked");
}
fn wheel_side() -> f32 {
    wheel_geometry().1[0]
}
fn expect_view_kept(world: &mut ApplicationWorld) {
    let earlier = world
        .observed_view_camera
        .expect("expected the earlier view");
    assert_eq!(camera(world), earlier, "view");
    solid_extent(world);
}
fn open_options(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Options");
    world.expect_text(OPTIONS_TITLE);
}
fn open_list(world: &mut ApplicationWorld, label: &str) {
    let field = world.run(label).clone();
    world.click([field.minimum[0] as i32 - 40, field.center()[1] as i32]);
    settle(world);
}
fn open_full_wheel(world: &mut ApplicationWorld) {
    open_wheel(world);
    right_click_wheel(world);
    choose_row(world, "Full Navigation Wheel");
    right_click_wheel(world);
    expect_marked(world, "Full Navigation Wheel");
    choose_row(world, "Full Navigation Wheel");
}
#[given("the full navigation wheel is the selected wheel layout")]
fn full_selected(world: &mut ApplicationWorld) {
    open_full_wheel(world);
    right_click_wheel(world);
    choose_row(world, "Close Wheel");
    assert!(diagnostics::window_geometry(WHEEL).is_none());
}
#[when("I open the navigation wheel from the navigation bar")]
#[when("the user opens the navigation wheel from the navigation bar")]
fn open_from_bar(world: &mut ApplicationWorld) {
    world.observed_view_camera = Some(camera(world));
    press_bar(world, NavigationControl::Wheel);
}
#[when("I click within the viewport")]
#[when("the user clicks within the viewport")]
fn click_view(world: &mut ApplicationWorld) {
    world.click(PLACED_AT);
    settle(world);
}
#[then("a circular navigation wheel is visible over the model")]
fn wheel_visible(_: &mut ApplicationWorld) {
    wheel_geometry();
}
#[then("the wheel offers Zoom, Orbit, Pan, Rewind, Center, Walk, Look and Up Down")]
fn full_tools(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Zoom", "Orbit", "Pan", "Rewind", "Center", "Walk", "Look", "Up Down",
        ],
    );
}
#[when("I right-click the navigation wheel")]
#[when("the user right-clicks the navigation wheel")]
#[when("I right-click the wheel")]
#[when("the user right-clicks the wheel")]
fn right_click(world: &mut ApplicationWorld) {
    right_click_wheel(world);
}
#[then("its menu marks Full Navigation Wheel as selected")]
fn full_marked(world: &mut ApplicationWorld) {
    expect_marked(world, "Full Navigation Wheel");
}
#[then("the menu offers alternate mini and basic wheel layouts")]
fn layouts_offered(world: &mut ApplicationWorld) {
    expect_rows(world, &LAYOUTS);
}
#[then("the menu offers Go Home, Fit to Window, Restore Original Center, Options and Close Wheel")]
fn menu_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &[
            "Go Home",
            "Fit to Window",
            "Restore Original Center",
            "Options",
            "Close Wheel",
        ],
    );
}
#[when("I choose Close Wheel")]
#[when("the user chooses Close Wheel")]
fn close_wheel(world: &mut ApplicationWorld) {
    choose_row(world, "Close Wheel");
}
#[then("the wheel and its menu disappear")]
#[then("the wheel disappears")]
fn wheel_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Close Wheel");
    world.expect_no_text("Rewind");
}
#[then("the model remains visible in the same apparent view")]
fn same_view(world: &mut ApplicationWorld) {
    expect_view_kept(world);
}
// Basic layouts
#[given("the full navigation wheel is open over a visible model in the MCAD Software")]
#[given("the full navigation wheel is open over a model in the MCAD Software")]
fn wheel_open(world: &mut ApplicationWorld) {
    open_wheel_on_block(world);
}
#[when("I right-click the wheel and choose Basic View Object Wheel")]
#[when("the user right-clicks the wheel and chooses Basic View Object Wheel")]
fn choose_basic_view(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Basic View Object Wheel");
}
#[then("the wheel shows Center, Zoom, Rewind and Orbit controls")]
fn basic_view_tools(world: &mut ApplicationWorld) {
    expect_rows(world, &["Center", "Zoom", "Rewind", "Orbit"]);
}
#[then("Basic View Object Wheel is marked as the selected layout")]
fn basic_view_marked(world: &mut ApplicationWorld) {
    expect_marked(world, "Basic View Object Wheel");
}
#[when("I choose Basic Tour Building Wheel")]
#[when("the user chooses Basic Tour Building Wheel")]
fn choose_basic_tour(world: &mut ApplicationWorld) {
    choose_row(world, "Basic Tour Building Wheel");
}
#[then("the wheel shows Forward, Look, Rewind and Up Down controls")]
fn basic_tour_tools(world: &mut ApplicationWorld) {
    expect_rows(world, &["Forward", "Look", "Rewind", "Up Down"]);
}
#[then("Basic Tour Building Wheel is marked as the selected layout")]
fn basic_tour_marked(world: &mut ApplicationWorld) {
    expect_marked(world, "Basic Tour Building Wheel");
}
#[when("I choose Full Navigation Wheel")]
#[when("the user chooses Full Navigation Wheel")]
fn choose_full(world: &mut ApplicationWorld) {
    choose_row(world, "Full Navigation Wheel");
}
// Compact layouts
#[when("I right-click the wheel and choose Mini View Object Wheel")]
#[when("the user right-clicks the wheel and chooses Mini View Object Wheel")]
fn choose_mini_view(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Mini View Object Wheel");
}
#[then("a small segmented circular wheel appears")]
#[then("a small segmented circular wheel remains visible")]
fn small_wheel(world: &mut ApplicationWorld) {
    assert!(wheel_side() < MINI_SIDE, "side {}", wheel_side());
    world.hover([500, 500]);
    settle(world);
    let count = visible_mini_segments(world);
    assert!(
        count >= 2,
        "mini wheel segments {count}: expected a segmented wheel"
    );
    world.observed_counts = vec![count];
}
#[then("the hovered segment is identified as Rewind")]
fn hovered_rewind(world: &mut ApplicationWorld) {
    let (position, size) = wheel_geometry();
    // Rewind is the mini ring's right-hand segment.
    let at = [
        (position[0] + size[0] * 0.85) as i32,
        (position[1] + size[1] / 2.) as i32,
    ];
    world.hover(at);
    settle(world);
    world.expect_text("Rewind");
}
#[when("I right-click the wheel and choose Mini Tour Building Wheel")]
#[when("the user right-clicks the wheel and chooses Mini Tour Building Wheel")]
fn choose_mini_tour(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Mini Tour Building Wheel");
}
#[then("Mini Tour Building Wheel is marked as the selected layout")]
fn mini_tour_marked(world: &mut ApplicationWorld) {
    expect_marked(world, "Mini Tour Building Wheel");
}
#[when("I choose Mini Full Navigation Wheel")]
#[when("the user chooses Mini Full Navigation Wheel")]
fn choose_mini_full(world: &mut ApplicationWorld) {
    choose_row(world, "Mini Full Navigation Wheel");
}
fn visible_mini_segments(world: &mut ApplicationWorld) -> usize {
    let bounds = diagnostics::window_pixel_bounds(WHEEL).expect("expected mini wheel bounds");
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    segment_pixels::visible_separators(&capture, bounds)
}
#[then("a small circular wheel with more segments appears")]
fn more_segments(world: &mut ApplicationWorld) {
    world.hover([500, 500]);
    settle(world);
    let count = visible_mini_segments(world);
    assert!(wheel_side() < MINI_SIDE, "side {}", wheel_side());
    assert!(
        count > world.observed_counts[0],
        "mini full segments {count}: expected more than {}",
        world.observed_counts[0]
    );
}
#[then("Mini Full Navigation Wheel is marked as the selected layout")]
fn mini_full_marked(world: &mut ApplicationWorld) {
    expect_marked(world, "Mini Full Navigation Wheel");
}
// Options
#[given("the saved mini wheel size is Normal")]
fn mini_normal(world: &mut ApplicationWorld) {
    set_saved_size(world, "Mini Wheel Size", "Normal");
}
#[when("I open Options from the wheel context menu")]
#[when("the user opens Options from the wheel context menu")]
#[when("I reopen Options from the wheel context menu")]
#[when("the user reopens Options from the wheel context menu")]
fn options_from_menu(world: &mut ApplicationWorld) {
    open_options(world);
}
#[then("the dialog shows tool message and tool tip controls")]
fn tool_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["Show Tool Messages", "Show Tool Tips"]);
}
#[then("it shows mini wheel size, big wheel size and wheel opacity controls")]
fn size_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Mini Wheel Size", "Big Wheel Size", "Wheel Opacity"],
    );
}
#[then("it shows navigation options and a walk speed factor slider")]
fn navigation_rows(world: &mut ApplicationWorld) {
    expect_rows(
        world,
        &["Invert vertical axis for Look", "Walk Speed Factor"],
    );
}
#[when("I open the mini wheel size list")]
#[when("the user opens the mini wheel size list")]
fn open_mini_list(world: &mut ApplicationWorld) {
    open_list(world, "Mini Wheel Size");
}
#[then("it offers Small, Normal, Large and Extra Large")]
fn mini_sizes(world: &mut ApplicationWorld) {
    expect_rows(world, &["Small", "Normal", "Large", "Extra Large"]);
}
#[when("I choose Extra Large")]
#[when("the user chooses Extra Large")]
fn choose_extra_large(world: &mut ApplicationWorld) {
    choose_row(world, "Extra Large");
}
fn set_saved_size(world: &mut ApplicationWorld, field: &str, size: &str) {
    open_options(world);
    open_list(world, field);
    choose_row(world, size);
    expect_field_reads(world, field, size);
    choose_row(world, "OK");
    open_options(world);
    expect_field_reads(world, field, size);
    choose_row(world, "Cancel");
}
fn expect_field_reads(world: &ApplicationWorld, label: &str, value: &str) {
    let field = world.run(label).clone();
    let shown = world.frame.runs.iter().any(|run| {
        run.text.replace(' ', "") == value.replace(' ', "")
            && run.maximum[0] < field.minimum[0]
            && (run.center()[1] - field.center()[1]).abs() < 4.
    });
    assert!(shown, "expected {label} to read {value}");
}
#[then("the mini wheel size field displays Extra Large")]
fn field_extra_large(world: &mut ApplicationWorld) {
    expect_field_reads(world, "Mini Wheel Size", "Extra Large");
}
#[then("OK is enabled")]
fn ok_enabled(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}
#[then("the mini wheel size field displays Normal")]
fn field_normal(world: &mut ApplicationWorld) {
    expect_field_reads(world, "Mini Wheel Size", "Normal");
}
#[when("I click Cancel and press Escape")]
#[when("the user clicks Cancel and presses Escape")]
fn cancel_and_escape(world: &mut ApplicationWorld) {
    choose_row(world, "Cancel");
    world.escape();
    settle(world);
}
#[then("the options dialog and wheel are closed")]
fn all_closed(world: &mut ApplicationWorld) {
    world.expect_no_text(OPTIONS_TITLE);
    world.expect_no_text("Rewind");
}
#[given("the navigation wheel options dialog is open in the MCAD Software")]
fn options_open(world: &mut ApplicationWorld) {
    open_wheel_on_block(world);
    open_options(world);
}
#[when("I open the big wheel size list")]
#[when("the user opens the big wheel size list")]
fn open_big_list(world: &mut ApplicationWorld) {
    open_list(world, "Big Wheel Size");
}
#[then("it offers Small, Normal and Large")]
fn big_sizes(world: &mut ApplicationWorld) {
    expect_rows(world, &["Small", "Normal", "Large"]);
}
#[when("I dismiss that list and open the wheel opacity list")]
#[when("the user dismisses that list and opens the wheel opacity list")]
fn open_opacity_list(world: &mut ApplicationWorld) {
    world.click_text("Big Wheel Size");
    settle(world);
    world.expect_text(OPTIONS_TITLE);
    open_list(world, "Wheel Opacity");
}
#[then("it offers 25%, 50%, 75% and 90%")]
fn opacity_rows(world: &mut ApplicationWorld) {
    expect_rows(world, &["25%", "50%", "75%", "90%"]);
}
#[when("I dismiss the list and click Cancel")]
#[when("the user dismisses the list and clicks Cancel")]
fn dismiss_cancel(world: &mut ApplicationWorld) {
    if diagnostics::window_geometry("Extrude").is_some() {
        world.escape();
        let cancel = world.run_in_window("Extrude", "Cancel").center();
        world.click(diagnostics::pointer_pixels(cancel));
        world.pump(Vec::new());
        return;
    }
    world.click_text("Wheel Opacity");
    settle(world);
    world.expect_text(OPTIONS_TITLE);
    choose_row(world, "Cancel");
}
#[then("the options dialog closes and the navigation wheel remains visible")]
fn dialog_closed_wheel_up(world: &mut ApplicationWorld) {
    world.expect_no_text(OPTIONS_TITLE);
    wheel_geometry();
}
// Big wheel size
fn remember_side(world: &mut ApplicationWorld) {
    world.observed_counts = vec![wheel_side() as usize];
}
// Fit and Home from the wheel's menu
#[given("a single rectangular block appears small in an active part in the MCAD Software")]
fn small_block(world: &mut ApplicationWorld) {
    ensure_block(world);
    let view = camera(world);
    crate::HeadWindow::set_view_camera(
        &mut world.app().window,
        oblikovati_viewport::ViewportCamera {
            scale: view.scale * SMALL,
            ..view
        },
    )
    .unwrap();
    world.pump(Vec::new());
    world.observed_view_camera = Some(camera(world));
    world.observed_counts = vec![solid_extent(world)[0]];
}
#[given("the full navigation wheel is open")]
fn wheel_is_open(world: &mut ApplicationWorld) {
    open_full_wheel(world);
}
#[when("I right-click the wheel and choose Fit to Window")]
#[when("the user right-clicks the wheel and chooses Fit to Window")]
fn choose_fit(world: &mut ApplicationWorld) {
    right_click_wheel(world);
    choose_row(world, "Fit to Window");
}
#[then("the wheel remains open")]
#[then("the navigation wheel remains open")]
fn stays_open(_: &mut ApplicationWorld) {
    wheel_geometry();
}
#[then("the wheel closes and the status area displays Ready")]
fn closes_ready(world: &mut ApplicationWorld) {
    world.expect_no_text("Rewind");
    world.expect_ready();
}
#[path = "home.rs"]
mod home;
#[path = "opacity.rs"]
mod opacity;

#[cfg(test)]
#[path = "layout_fixture_regression.rs"]
mod layout_fixture_regression;
