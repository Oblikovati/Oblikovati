// SPDX-License-Identifier: GPL-2.0-only
//! Drawing browser workspace scenarios.
use super::*;

pub(super) fn expect_sheet_at(world: &mut ApplicationWorld, at: [usize; 2]) {
    let sheet = world.pixel_at(at);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "pixel {at:?} {sheet:?}: expected the light drawing sheet"
    );
}
#[given("its browser is docked to the left of the drawing")]
fn drawing_browser_docked(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[when("I drag the browser header into the drawing area")]
#[when("the user drags the browser header into the drawing area")]
pub(super) fn drag_over_drawing(world: &mut ApplicationWorld) {
    drag_header(world, [800, 300]);
}
#[then("the browser floats over the drawing sheet")]
#[then("the browser remains floating over the sheet")]
pub(super) fn floats_over_sheet(_world: &mut ApplicationWorld) {
    let (position, _) = browser_geometry();
    assert!(
        position[0] > 400.,
        "browser {position:?}: expected it over the sheet"
    );
}
#[then("the browser displays drawing resources and the sheet tree")]
fn drawing_tree_shown(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
}
#[then("the drawing remains visible behind the browser")]
#[then("the drawing sheet remains visible behind the browser")]
fn sheet_behind(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [500, 500]);
}
#[then("the browser docks at the left edge")]
fn drawing_browser_docks(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[then("the drawing remains visible in the remaining workspace")]
fn sheet_remains(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
#[given("its model browser is docked on the left")]
#[given("the drawing browser is docked on the left")]
#[given("its Model browser is docked on the left")]
fn docked_on_left(world: &mut ApplicationWorld) {
    expect_docked_left(world);
    world.expect_text("Drawing Resources");
}
#[when("I select the browser tab's close control")]
#[when("the user selects the browser tab's close control")]
fn close_browser_tab(world: &mut ApplicationWorld) {
    world.click_run_close_mark("Model");
}
#[then("the browser tree is hidden")]
fn tree_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Drawing Resources");
    assert!(
        window_geometry(BROWSER).is_none(),
        "expected no browser window"
    );
}
#[then("the drawing occupies the space previously used by the browser")]
fn drawing_takes_space(world: &mut ApplicationWorld) {
    // This column was inside the docked browser, and the wider sheet now reaches it.
    expect_sheet_at(world, [300, 500]);
}
#[then("a plus control remains at the upper left of the workspace")]
#[then("a panel-add control remains at the upper left of the workspace")]
fn plus_remains(world: &mut ApplicationWorld) {
    let plus = world.run("+").center();
    assert!(
        plus[0] < 60. && plus[1] < 260.,
        "plus control {plus:?}: expected the upper left"
    );
}
#[when("I open the plus control")]
#[when("the user opens the plus control")]
#[when("I open the panel-add control")]
#[when("the user opens the panel-add control")]
fn open_plus(world: &mut ApplicationWorld) {
    if world.shows("+") {
        click_panel_action(world, world.run("+").center());
        return;
    }
    let dock = super::super::diagnostics::window_dock_id("Model")
        .filter(|id| *id != 0)
        .expect("expected the Model dock or the workspace launcher");
    super::dock_search::click_dock_tab_action(world, dock, c"Model", c"   ##dock-panel-menu");
}

pub(super) fn click_panel_action(world: &mut ApplicationWorld, center: [f32; 2]) {
    let point = super::super::diagnostics::pointer_pixels(center);
    #[cfg(target_os = "linux")]
    {
        super::super::x11_input::focus_window();
        super::super::x11_input::click(point);
    }
    #[cfg(not(target_os = "linux"))]
    world.click(point);
    world.pump(Vec::new());
}
#[then("the panel list offers Model Browser")]
fn offers_model(world: &mut ApplicationWorld) {
    world.expect_text("Model Browser");
}
#[when("I select Model Browser")]
#[when("the user selects Model Browser")]
fn select_model_browser(world: &mut ApplicationWorld) {
    let center = world.run("Model Browser").center();
    click_panel_action(world, center);
}
#[then("the browser returns to the left")]
#[then("the model browser returns to the left")]
fn browser_back(world: &mut ApplicationWorld) {
    expect_docked_left(world);
}
#[then("it displays the drawing resources and sheet tree")]
#[then("the drawing resources and sheet tree are displayed")]
pub(super) fn drawing_tree(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_text("Sheet:1");
}
#[given("an additional log panel is available in the browser's plus menu")]
fn log_available(world: &mut ApplicationWorld) {
    open_plus(world);
    world.expect_text("Log Panel");
    world.escape();
}
#[when("I add the log panel from the plus menu")]
#[when("the user adds the log panel from the plus menu")]
pub(in super::super) fn add_log(world: &mut ApplicationWorld) {
    open_plus(world);
    let center = world.run("Log Panel").center();
    click_panel_action(world, center);
}
fn expect_log_area(world: &mut ApplicationWorld) {
    world.expect_text("Application messages");
    world.expect_no_text("Drawing Resources");
}
#[then("the log panel becomes active in the same dock")]
#[then("the log area is displayed in the dock")]
fn log_active(world: &mut ApplicationWorld) {
    expect_log_area(world);
    expect_docked_left(world);
}
#[then("the Model browser remains available as a tab")]
fn model_tab_remains(_: &mut ApplicationWorld) {
    use super::super::diagnostics::{window_dock_id, window_dock_tab_visible};
    let dock = window_dock_id(BROWSER).filter(|id| *id != 0);
    assert!(dock.is_some(), "expected Model to remain docked");
    assert_eq!(window_dock_id("Log"), dock);
    assert_eq!(window_dock_tab_visible("Log"), Some(true));
}
#[when("I select the Model browser tab")]
#[when("the user selects the Model browser tab")]
fn select_model_tab(world: &mut ApplicationWorld) {
    select_dock_tab(world, BROWSER);
}
#[when("I select the log panel tab")]
#[when("the user selects the log panel tab")]
pub(super) fn select_log_tab(world: &mut ApplicationWorld) {
    select_dock_tab(world, "Log");
}
fn select_dock_tab(world: &mut ApplicationWorld, name: &str) {
    let (minimum, maximum) = super::super::diagnostics::window_dock_tab_bounds(name)
        .unwrap_or_else(|| panic!("{name:?}: expected a dock tab"));
    let center = std::array::from_fn(|axis| (minimum[axis] + maximum[axis]) * 0.5);
    click_panel_action(world, center);
}
#[when("I close the log panel tab")]
#[when("the user closes the log panel tab")]
fn close_log_tab(world: &mut ApplicationWorld) {
    world.click_run_close_mark("Log");
    world.pump(Vec::new());
}
#[then("the Model browser is displayed")]
fn model_displayed_again(world: &mut ApplicationWorld) {
    drawing_tree(world);
    world.expect_no_text("Application messages");
}
#[then("the drawing remains visible")]
fn drawing_visible(world: &mut ApplicationWorld) {
    expect_sheet_at(world, [1000, 500]);
}
