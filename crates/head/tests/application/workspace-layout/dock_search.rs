// SPDX-License-Identifier: GPL-2.0-only
//! Click the actual dock search item and verify focus and filtering (#3897).
use super::super::diagnostics;
use super::{ApplicationWorld, drawing_browser_steps, native_part};
use cucumber::{given, then, when};
use imgui::sys;
use std::ffi::CStr;

#[test]
fn native_model_menu_and_search_live_in_the_dock_title() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let body = world.window_text("Model");
    assert!(!body.contains(" Menu "), "{body}");
    assert!(!body.contains(" Search "), "{body}");
    world.save_capture("model-dock-title-controls");

    click_model_menu(&mut world);
    world.expect_text("Autoscroll");
    world.expect_text("Collapse All");
    world.escape();
    world.pump(Vec::new());
    world.expect_no_text("Collapse All");

    click_model_search(&mut world);
    empty_filter_is_focused(&mut world);
    world.close();
}

#[given("Model and Log occupy one dock tab bar with Log active")]
fn log_is_active(world: &mut ApplicationWorld) {
    drawing_browser_steps::add_log(world);
    let dock = diagnostics::window_dock_id("Model").filter(|value| *value != 0);
    assert!(dock.is_some(), "expected Model to have a dock node");
    assert_eq!(diagnostics::window_dock_id("Log"), dock);
    assert_eq!(diagnostics::window_dock_tab_visible("Log"), Some(true));
    world.docking_node = dock;
}

#[given("the Model search field is closed")]
fn tree_search_closed(world: &mut ApplicationWorld) {
    assert!(!tree_filter_focused());
    world.expect_no_text("Type Keyword");
}

#[when("the user clicks the search icon in that dock tab bar")]
fn open_dock_search(world: &mut ApplicationWorld) {
    let dock = world.docking_node.expect("expected the saved dock node");
    click_dock_tab_action(world, dock, c"Log", c"   ##dock-search");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(in super::super) fn click_model_menu(world: &mut ApplicationWorld) {
    click_model_control(world, c"   ##dock-model-menu", "Menu");
}

pub(in super::super) fn click_model_search(world: &mut ApplicationWorld) {
    click_model_control(world, c"   ##dock-search", "Search");
}

fn click_model_control(world: &mut ApplicationWorld, item: &CStr, floating_label: &str) {
    if let Some(dock) = diagnostics::window_dock_id("Model").filter(|id| *id != 0) {
        click_dock_tab_action(world, dock, c"Model", item);
    } else {
        world.click_text(floating_label);
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(in super::super) fn check_inactive_hover(
    world: &mut ApplicationWorld,
    panels: [&str; 2],
    active: &str,
) {
    let bounds = panels.map(|tab| diagnostics::window_dock_tab_bounds(tab).unwrap());
    let (min, max) = bounds[usize::from(active == panels[0])];
    world.hover(diagnostics::pointer_pixels([
        max[0] - 8.,
        (min[1] + max[1]) / 2.,
    ]));
    assert_eq!(
        panels.map(|tab| diagnostics::window_dock_tab_bounds(tab).unwrap()),
        bounds,
        "revealing an inactive tab's close button must preserve both tab bounds"
    );
}

/// Click a trailing dock item relative to a live window tab in the same bar.
/// Example: `click_dock_tab_action(world, dock, c"Log", c"   ##dock-search")`.
pub(in super::super) fn click_dock_tab_action(
    world: &mut ApplicationWorld,
    dock: u32,
    anchor: &CStr,
    label: &CStr,
) {
    // SAFETY: the live label and dock ID use the tab-bar scope's hash.
    let id = unsafe { sys::igImHashStr(label.as_ptr(), 0, dock) };
    let [x, y, width] = dock_action_item(dock, id, anchor)
        .unwrap_or_else(|| panic!("expected dock item {label:?}"));
    // The native title bar can offset its buttons after tab measurement. Confirm
    // the real pointer target before clicking so a nearby button cannot pass.
    let center = [x, x + width * 0.5, x - width * 0.5]
        .into_iter()
        .map(|x| [x, y])
        .find(|point| hover_dock_action(world, *point, id))
        .unwrap_or_else(|| {
            panic!("dock item {label:?} near {x}, {y}: expected its pointer target")
        });
    drawing_browser_steps::click_panel_action(world, center);
}

fn hover_dock_action(world: &mut ApplicationWorld, center: [f32; 2], id: u32) -> bool {
    let point = diagnostics::pointer_pixels(center);
    #[cfg(target_os = "linux")]
    {
        super::super::x11_input::move_pointer(point);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.hover(point);
    // SAFETY: the focused test owns this frame's live ImGui context.
    unsafe { (*sys::igGetCurrentContext()).HoveredId == id }
}

fn dock_action_item(dock: u32, id: u32, anchor: &CStr) -> Option<[f32; 3]> {
    // SAFETY: test diagnostics only borrow this frame's live ImGui context and tab bar.
    unsafe {
        let node = sys::igDockBuilderGetNode(dock).as_ref()?;
        let bar = node.TabBar.as_ref()?;
        let tab = sys::igTabBarFindTabByID(node.TabBar, id).as_ref()?;
        let window = sys::igFindWindowByName(anchor.as_ptr()).as_ref()?;
        let anchor = sys::igTabBarFindTabByID(node.TabBar, window.TabId).as_ref()?;
        // The dock node can amend the bar after laying out window tabs. Anchor
        // the trailing button to the actual bounds of a normal dock tab.
        let origin = window.DockTabItemRect.Min.x - (anchor.Offset - bar.ScrollingAnim).floor();
        (tab.Width > 0.).then_some([
            origin + tab.Offset + tab.Width * 0.5,
            (bar.BarRect.Min.y + bar.BarRect.Max.y) * 0.5,
            tab.Width,
        ])
    }
}

fn tree_filter_focused() -> bool {
    // SAFETY: both pointers are read on the owner thread and do not survive this call.
    unsafe {
        let Some(model) = sys::igFindWindowByName(c"Model".as_ptr()).as_ref() else {
            return false;
        };
        let Some(context) = sys::igGetCurrentContext().as_ref() else {
            return false;
        };
        context.ActiveId == sys::igImHashStr(c"##browser-search".as_ptr(), 0, model.ID)
    }
}

#[then("Model becomes active and its empty search field has keyboard focus")]
fn empty_filter_is_focused(world: &mut ApplicationWorld) {
    assert_model_filter_has_focus();
    world.expect_text("Type Keyword");
}

fn assert_model_filter_has_focus() {
    assert_eq!(diagnostics::window_dock_tab_visible("Model"), Some(true));
    assert!(
        tree_filter_focused(),
        "expected keyboard focus in the Model search input"
    );
}

#[then("the Find criteria dialog remains closed")]
fn criteria_dialog_closed(_: &mut ApplicationWorld) {
    // SAFETY: the current context is live and this test retains no window pointer.
    unsafe {
        let hash = sys::igImHashStr(c"###find-dialog".as_ptr(), 0, 0);
        let window = sys::igFindWindowByID(hash).as_ref();
        assert!(!window.is_some_and(|window| window.Active));
    }
}

#[when(expr = "the user types {string} without clicking another control")]
fn type_using_current_focus(world: &mut ApplicationWorld, text: String) {
    #[cfg(target_os = "linux")]
    for character in text.chars() {
        super::super::x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.type_text(&text);
}

#[then("the Model tree shows InspectionPointA, InspectionPointB and InspectionPointC")]
fn model_tree_shows_points(world: &mut ApplicationWorld) {
    native_part::filtered_point_rows(world);
}

#[then("the query is highlighted in the matching tree rows")]
fn matching_point_prefixes_are_highlighted(world: &mut ApplicationWorld) {
    for name in native_part::INSPECTION_POINTS {
        assert!(
            world.text_is_selected(name),
            "{name}: expected highlight behind the match"
        );
    }
}

#[when("the user switches to Log and clicks the dock tab bar search icon again")]
fn return_to_model_search(world: &mut ApplicationWorld) {
    drawing_browser_steps::select_log_tab(world);
    assert_eq!(diagnostics::window_dock_tab_visible("Log"), Some(true));
    open_dock_search(world);
}

#[then(expr = "Model becomes active with {string} still in the focused search field")]
fn existing_filter_is_focused(world: &mut ApplicationWorld, query: String) {
    assert_model_filter_has_focus();
    world.expect_text(&query);
    native_part::filtered_point_rows(world);
}
