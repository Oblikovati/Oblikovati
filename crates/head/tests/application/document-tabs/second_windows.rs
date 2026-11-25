// SPDX-License-Identifier: GPL-2.0-only
//! A second window on one part: numbered tabs, separate views, closing without a prompt.
use super::super::ApplicationWorld;
use crate::HeadWindow;
use crate::work::HeadWork;
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportPhysicalKey};

fn numbered_tab_label(world: &mut ApplicationWorld, number: u32) -> String {
    let part = world
        .previous_document
        .expect("expected the Part with two windows");
    format!("{}:{number}", world.document_name(part))
}

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
fn click_numbered_tab(world: &mut ApplicationWorld, number: u32) {
    let label = numbered_tab_label(world, number);
    click_window_tab(world, &label);
}
fn through_home(world: &mut ApplicationWorld, number: u32) {
    click_window_tab(world, "Home");
    click_numbered_tab(world, number);
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
#[when("the user chooses New in the View tab's Windows group")]
fn choose_new(world: &mut ApplicationWorld) {
    open_second_window(world);
}
#[given("the user creates a second window for the profile part through View New")]
#[given("the user creates a second window for the part through View New")]
fn create_second_saved_window(world: &mut ApplicationWorld) {
    open_second_window(world);
    second_window_active(world);
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
    let label = numbered_tab_label(world, 2);
    world.expect_tab(&label);
}
#[then("the part's model browser is displayed")]
fn browser_shown(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected a remembered part");
    world.expect_browser_of(part);
}
#[then("the original part window is listed with a different window number")]
fn original_listed(world: &mut ApplicationWorld) {
    for number in [1, 2] {
        let label = numbered_tab_label(world, number);
        world.expect_text(&label);
    }
}
#[when("I select the original part window")]
#[when("the user selects the original part window")]
fn select_original(world: &mut ApplicationWorld) {
    let label = numbered_tab_label(world, 1);
    world.click_text(&label);
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
    let label = numbered_tab_label(world, 2);
    world.expect_tab(&label);
}
// Keyboard
#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd`, `fixtures/cad/quick-access-toolbar/PartTwo.opd` and `fixtures/cad/revolve/revolve-profile.opd` are open"
)]
fn open_saved_window_parts(world: &mut ApplicationWorld) {
    use super::saved_tabs::{open_disposable_part, open_disposable_tab};
    world.restore_layout_without_asset_tools();
    open_disposable_part(world, "PartOne.opd");
    open_disposable_part(world, "PartTwo.opd");
    let profile = open_disposable_tab(
        world,
        "fixtures/cad/revolve/revolve-profile.opd",
        oblikovati_api::types::DOCUMENT_PART,
    );
    world.parts.push(profile);
    world.previous_document = Some(profile);
}
#[given("Home and several document windows are open in the MCAD Software")]
fn home_and_windows(world: &mut ApplicationWorld) {
    let part = world.add_part();
    world.previous_document = Some(part);
    open_second_window(world);
}
#[given("Home is the first tab and a part's second window is the last tab")]
fn order_of_tabs(world: &mut ApplicationWorld) {
    let home = world.expect_tab("Home").center()[0];
    let label = numbered_tab_label(world, 2);
    let last = world.expect_tab(&label).center()[0];
    for part in world.parts.clone() {
        let name = world.document_name(part);
        let x = world.expect_tab(&name).center()[0];
        assert!(
            home < x && x < last,
            "tab {name:?} at {x}: expected between Home and {label}"
        );
    }
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
    click_numbered_tab(world, 1);
    let probe = view_probe(world);
    assert_eq!(
        probe[1], 0,
        "probe {probe:?}: expected a view facing the sketch"
    );
}
#[given("the second window displays the part in an isometric view")]
fn second_isometric(world: &mut ApplicationWorld) {
    click_numbered_tab(world, 2);
    let probe = view_probe(world);
    assert_ne!(probe[1], 0, "probe {probe:?}: expected an oblique view");
}
#[given("the first window is active")]
fn first_window_active(world: &mut ApplicationWorld) {
    click_numbered_tab(world, 1);
}
#[when("I switch to the second window through Home")]
#[when("the user switches to the second window through Home")]
fn second_through_home(world: &mut ApplicationWorld) {
    through_home(world, 2);
}
#[then("the second window displays its isometric view")]
fn shows_isometric(world: &mut ApplicationWorld) {
    let probe = view_probe(world);
    assert_ne!(probe[1], 0, "probe {probe:?}: expected an oblique view");
}
#[when("I switch back to the first window through Home")]
#[when("the user switches back to the first window through Home")]
fn first_through_home(world: &mut ApplicationWorld) {
    through_home(world, 1);
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
    click_numbered_tab(world, 1);
    world.expect_text("Finish Sketch");
}
#[given("the second window is active after switching from Home")]
fn second_from_home(world: &mut ApplicationWorld) {
    through_home(world, 2);
}
#[when("I press Control+F4")]
#[when("the user presses Control+F4")]
fn control_f4(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::F4, true);
    world.pump(Vec::new());
}
#[then("the second window closes without a save prompt")]
fn closed_without_prompt(world: &mut ApplicationWorld) {
    let label = numbered_tab_label(world, 2);
    assert!(
        world.tab_run(&label).is_none(),
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

#[test]
fn hidden_tabs_survive_closing_a_window_and_show_the_survivor_preview() {
    let mut world = ApplicationWorld::new();
    two_windows(&mut world);
    let part = world.previous_document.unwrap();
    super::toggle_tabs_option(&mut world);
    assert!(super::super::diagnostics::window_geometry("##open-document-tabs").is_none());
    control_f4(&mut world);
    assert!(world.state().document_windows.is_empty());
    assert!(world.document_is_open(part));
    super::toggle_tabs_option(&mut world);
    world.pump(Vec::new());
    let name = world.document_name(part);
    let at = world.expect_tab(&name).center().map(|value| value as i32);
    world.hover(at);
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("##tab-preview").is_some());
    world.save_capture("hidden-tab-surviving-window-preview");
    world.close();
}

#[test]
fn scaled_preview_avoids_the_tab_when_ribbon_is_minimized() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    let part = world.add_part();
    super::super::ribbon_display::click_display_menu(&mut world);
    world.click_text("Minimize to Tabs");
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(2., 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    let name = world.document_name(part);
    let tab = world.expect_tab(&name);
    world.hover(super::super::diagnostics::pointer_pixels(tab.center()));
    world.pump(Vec::new());
    let (position, size) = super::super::diagnostics::window_geometry("##tab-preview")
        .expect("expected preview with enlarged text");
    let (strip, strip_size) = super::super::diagnostics::window_geometry("##open-document-tabs")
        .expect("expected document tabs");
    assert!(
        position[1] + size[1] <= strip[1],
        "preview {position:?} / {size:?} overlaps strip {strip:?} / {strip_size:?}"
    );
    let [width, height] = world.extent();
    assert!(position[0] >= 0. && position[1] >= 0.);
    assert!(position[0] + size[0] <= width as f32 / 2.);
    assert!(position[1] + size[1] <= height as f32 / 2.);
    world.save_capture("scaled-minimized-tab-preview");
    world.hover([width as i32 - 80, height as i32 - 80]);
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("##tab-preview").is_none());
    world.expect_active(part);
    world.close();
}

#[test]
fn full_ribbon_keeps_the_preview_above_the_tab() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let name = world.document_name(part);
    let tab = world.expect_tab(&name);
    world.hover(super::super::diagnostics::pointer_pixels(tab.center()));
    world.pump(Vec::new());
    let (position, size) = super::super::diagnostics::window_geometry("##tab-preview")
        .expect("expected preview on full ribbon");
    let (strip, _) = super::super::diagnostics::window_geometry("##open-document-tabs")
        .expect("expected document tabs");
    assert!(position[1] + size[1] <= strip[1]);
    world.close();
}
// Separate zoom
#[given("the first window displays an active sketch")]
fn first_active_sketch(world: &mut ApplicationWorld) {
    click_numbered_tab(world, 1);
    world.expect_text("Finish Sketch");
    let part = world.active_id();
    if super::saved_sketch_parts::is_saved_sketch_part(world, part) {
        expect_active_profile(world);
    } else {
        world.draw_rectangle();
        world.pump(Vec::new());
        world.expect_drawn_region();
    }
    world.observed_view = Some(view_probe(world));
}

fn expect_active_profile(world: &mut ApplicationWorld) {
    assert_eq!(world.edited_geometry().lines().len(), 4);
    // The rectangle has an edge on the red axis. Sample the opposite edge away from the green axis.
    let from = world.line_point(2, 0.6);
    let to = world.line_point(2, 0.85);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        super::super::rectangle_visibility::edge_visible(&capture, from, to, [0, -1]),
        "profile edge {from:?} to {to:?}: expected pixels away from the sketch axes"
    );
}
#[given("the second window is active")]
fn second_is_active_window(world: &mut ApplicationWorld) {
    click_numbered_tab(world, 2);
}
#[when("I zoom in within the second window")]
#[when("I scroll to zoom in within the second window")]
#[when("the user scrolls to zoom in within the second window")]
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
    let length_squared = |v: [i32; 2]| i64::from(v[0]).pow(2) + i64::from(v[1]).pow(2);
    assert!(
        length_squared(after) > length_squared(before),
        "wheel zoom {before:?} -> {after:?}: expected an increase"
    );
    world.observed_second_view = Some(after);
}
#[then("its displayed geometry appears enlarged")]
fn zoomed_geometry_visible(world: &mut ApplicationWorld) {
    world.expect_drawn_region();
    assert_eq!(Some(view_probe(world)), world.observed_second_view);
}
#[when("I select the first window from the View tab's Switch menu")]
#[when("the user selects the first window from the View tab's Switch menu")]
fn first_from_switch(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.pump(Vec::new());
    world.click_text("Switch");
    world.pump(Vec::new());
    let label = numbered_tab_label(world, 1);
    world.click_text(&label);
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
#[when("the user returns to the second window")]
fn return_second(world: &mut ApplicationWorld) {
    click_numbered_tab(world, 2);
}
#[then("the second window retains its zoomed-in view")]
fn second_retained(world: &mut ApplicationWorld) {
    assert_eq!(
        Some(view_probe(world)),
        world.observed_second_view,
        "second window's view"
    );
}
