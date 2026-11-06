// SPDX-License-Identifier: GPL-2.0-only
//! A second window on one part: numbered tabs, separate views, closing without a prompt.
use super::super::ApplicationWorld;
use crate::HeadWindow;
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportPhysicalKey};

const FIRST: &str = "Part1:1";
const SECOND: &str = "Part1:2";

/// Where one sketch unit along X lands on screen: it reads a view's scale and turn.
fn view_probe(world: &mut ApplicationWorld) -> [i32; 2] {
    let window = &world.app().window;
    let origin = HeadWindow::sketch_pixel(window, SketchPlane::xy(), [0., 0.]).unwrap();
    let unit = HeadWindow::sketch_pixel(window, SketchPlane::xy(), [1., 0.]).unwrap();
    [unit[0] - origin[0], unit[1] - origin[1]]
}
fn click_window_tab(world: &mut ApplicationWorld, label: &str) {
    let at = world.expect_tab(label).center().map(|value| value as i32);
    world.click(at);
    world.pump(Vec::new());
}
fn through_home(world: &mut ApplicationWorld, label: &str) {
    click_window_tab(world, "Home");
    click_window_tab(world, label);
}
fn open_second_window(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.click_text("New");
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn two_windows(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
    world.start_sketch();
    world.pump(Vec::new());
    open_second_window(world);
}
#[when("I choose New in the View tab's Windows group")]
fn choose_new(world: &mut ApplicationWorld) {
    open_second_window(world);
}
#[then("a second window for the same part becomes active")]
fn second_active(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_active_by_id(part);
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part,
        "expected the new window outside the sketch"
    );
}
#[then("the new window has a numbered document tab")]
fn numbered_tab(world: &mut ApplicationWorld) {
    world.expect_tab(SECOND);
}
#[then("the part's model browser is displayed")]
fn browser_shown(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_browser_of(part);
}
#[then("the original part window is listed with a different window number")]
fn original_listed(world: &mut ApplicationWorld) {
    world.expect_text(FIRST);
    world.expect_text(SECOND);
}
#[when("I select the original part window")]
fn select_original(world: &mut ApplicationWorld) {
    world.click_text(FIRST);
    world.pump(Vec::new());
}
#[then("the original sketch viewport is displayed")]
fn original_sketch(world: &mut ApplicationWorld) {
    world.expect_text("Finish Sketch");
}
#[then("the original numbered document tab is active")]
fn original_tab_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let active = state.active_window.expect("expected an active window");
    let window = state
        .document_windows
        .iter()
        .find(|window| window.id == active)
        .expect("expected the active window listed");
    assert_eq!(window.number, 1, "expected the first window in front");
}
#[then("the second window remains open")]
fn second_open(world: &mut ApplicationWorld) {
    world.expect_tab(SECOND);
}
// Keyboard
#[given("Home and several document windows are open in the MCAD Software")]
fn home_and_windows(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
    open_second_window(world);
}
#[given("Home is the first tab and a part's second window is the last tab")]
fn order_of_tabs(world: &mut ApplicationWorld) {
    let home = world.expect_tab("Home").center()[0];
    let last = world.expect_tab(SECOND).center()[0];
    assert!(
        home < last,
        "Home at {home}: expected left of the window at {last}"
    );
}
#[given("Home is active")]
fn home_is_active(world: &mut ApplicationWorld) {
    click_window_tab(world, "Home");
}
#[then("the part's second window becomes active")]
fn second_window_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let active = state.active_window.expect("expected an active window");
    let window = state
        .document_windows
        .iter()
        .find(|window| window.id == active)
        .expect("expected the active window listed");
    assert_eq!(window.number, 2, "expected the second window in front");
}
// Separate views
#[given("two numbered windows of the same part are open in the MCAD Software")]
#[given("a part has two numbered windows in the MCAD Software")]
#[given("two numbered windows of one part are open in the MCAD Software")]
fn given_two_windows(world: &mut ApplicationWorld) {
    two_windows(world);
}
#[given("the first window displays the sketch in an orthogonal view")]
fn first_orthogonal(world: &mut ApplicationWorld) {
    click_window_tab(world, FIRST);
    let probe = view_probe(world);
    assert_eq!(
        probe[1], 0,
        "probe {probe:?}: expected a view facing the sketch"
    );
}
#[given("the second window displays the part in an isometric view")]
fn second_isometric(world: &mut ApplicationWorld) {
    click_window_tab(world, SECOND);
    let probe = view_probe(world);
    assert_ne!(probe[1], 0, "probe {probe:?}: expected an oblique view");
}
#[given("the first window is active")]
fn first_window_active(world: &mut ApplicationWorld) {
    click_window_tab(world, FIRST);
}
#[when("I switch to the second window through Home")]
fn second_through_home(world: &mut ApplicationWorld) {
    through_home(world, SECOND);
}
#[then("the second window displays its isometric view")]
fn shows_isometric(world: &mut ApplicationWorld) {
    let probe = view_probe(world);
    assert_ne!(probe[1], 0, "probe {probe:?}: expected an oblique view");
}
#[when("I switch back to the first window through Home")]
fn first_through_home(world: &mut ApplicationWorld) {
    through_home(world, FIRST);
}
#[then("the first window displays its orthogonal sketch view")]
fn shows_orthogonal(world: &mut ApplicationWorld) {
    let probe = view_probe(world);
    assert_eq!(
        probe[1], 0,
        "probe {probe:?}: expected a view facing the sketch"
    );
}
// Closing one window
#[given("the first window has an active sketch editing environment")]
fn first_has_edit(world: &mut ApplicationWorld) {
    click_window_tab(world, FIRST);
    world.expect_text("Finish Sketch");
}
#[given("the second window is active after switching from Home")]
fn second_from_home(world: &mut ApplicationWorld) {
    through_home(world, SECOND);
}
#[when("I press Control+F4")]
fn control_f4(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::F4, true);
    world.pump(Vec::new());
}
#[then("the second window closes without a save prompt")]
fn closed_without_prompt(world: &mut ApplicationWorld) {
    assert!(
        world.tab_run(SECOND).is_none(),
        "expected the second window gone"
    );
    world.expect_no_text("Save changes");
}
#[then("the remaining part tab no longer has a window number")]
fn no_number(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    let name = world.document_name(part);
    let run = world.expect_tab(&name);
    assert!(
        !run.text.contains(':'),
        "tab {:?}: expected no window number",
        run.text
    );
}
// Separate zoom
#[given("the first window displays an active sketch")]
fn first_active_sketch(world: &mut ApplicationWorld) {
    click_window_tab(world, FIRST);
    world.observed_view = Some(view_probe(world));
    world.expect_text("Finish Sketch");
}
#[given("the second window is active")]
fn second_is_active_window(world: &mut ApplicationWorld) {
    click_window_tab(world, SECOND);
}
#[when("I zoom in within the second window")]
fn zoom_second(world: &mut ApplicationWorld) {
    let before = view_probe(world);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Motion {
        position: [900, 500],
        modifiers: Default::default(),
    })]);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches: 3,
        position: [900, 500],
    })]);
    let after = view_probe(world);
    assert_ne!(before, after, "expected the wheel to change the view");
    world.observed_second_view = Some(after);
}
#[when("I select the first window from the View tab's Switch menu")]
fn first_from_switch(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.click_text("Switch");
    world.pump(Vec::new());
    world.click_text(FIRST);
    world.pump(Vec::new());
}
#[then("the first window retains its previous sketch view and scale")]
fn first_retained(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(view_probe(world)),
        world.observed_view,
        "first window's view"
    );
}
#[when("I return to the second window")]
fn return_second(world: &mut ApplicationWorld) {
    click_window_tab(world, SECOND);
}
#[then("the second window retains its zoomed-in view")]
fn second_retained(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(view_probe(world)),
        world.observed_second_view,
        "second window's view"
    );
}
