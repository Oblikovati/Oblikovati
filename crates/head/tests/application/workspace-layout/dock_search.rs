// SPDX-License-Identifier: GPL-2.0-only
//! Click the actual dock search item and verify focus and filtering (#3897).
use super::super::diagnostics;
use super::{ApplicationWorld, drawing_browser_steps, native_part};
use cucumber::{given, then, when};
use imgui::sys;

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
    let [x, y, width] = dock_search_item(dock).expect("expected a laid-out dock search button");
    // The native title bar can offset its buttons after tab measurement. Confirm
    // the real pointer target before clicking so a nearby button cannot pass.
    let center = [x, x + width * 0.5, x - width * 0.5]
        .into_iter()
        .map(|x| [x, y])
        .find(|point| hover_search(world, *point, dock))
        .unwrap_or_else(|| panic!("dock search near {x}, {y}: expected its pointer target"));
    drawing_browser_steps::click_panel_action(world, center);
}

fn hover_search(world: &mut ApplicationWorld, center: [f32; 2], dock: u32) -> bool {
    let point = diagnostics::pointer_pixels(center);
    #[cfg(target_os = "linux")]
    {
        super::super::x11_input::move_pointer(point);
        world.pump(Vec::new());
    }
    #[cfg(not(target_os = "linux"))]
    world.hover(point);
    // SAFETY: the focused test owns this frame's live ImGui context.
    unsafe { (*sys::igGetCurrentContext()).HoveredId == search_button_id(dock) }
}

fn search_button_id(dock: u32) -> u32 {
    // SAFETY: the C string is static. The dock ID seeds the same tab-bar scope.
    unsafe { sys::igImHashStr(c"   ##dock-search".as_ptr(), 0, dock) }
}

fn dock_search_item(dock: u32) -> Option<[f32; 3]> {
    // SAFETY: test diagnostics only borrow this frame's live ImGui context and tab bar.
    unsafe {
        let node = sys::igDockBuilderGetNode(dock).as_ref()?;
        let bar = node.TabBar.as_ref()?;
        // DockNodeBeginAmendTabBar pushes the dock node ID for trailing buttons.
        let id = search_button_id(dock);
        let tab = sys::igTabBarFindTabByID(node.TabBar, id).as_ref()?;
        let log = sys::igFindWindowByName(c"Log".as_ptr()).as_ref()?;
        let anchor = sys::igTabBarFindTabByID(node.TabBar, log.TabId).as_ref()?;
        // The dock node can amend the bar after laying out window tabs. Anchor
        // the trailing button to the actual bounds of a normal dock tab.
        let origin = log.DockTabItemRect.Min.x - (anchor.Offset - bar.ScrollingAnim).floor();
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
