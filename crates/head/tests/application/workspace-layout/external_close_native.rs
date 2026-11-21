// SPDX-License-Identifier: GPL-2.0-only
//! Verify a host-owned panel can close without a blank shared dock frame.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::select_dock_tab;
use super::orientation_close_native::drag_title;
use super::*;

#[test]
fn measure_done_selects_the_other_tab_in_the_first_frame() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let area = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    drag_title(
        &mut world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    world
        .app()
        .owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::StartMeasure)
        .unwrap();
    world.pump(Vec::new());
    let (position, size) = window_geometry("Material Browser").unwrap();
    drag_title(
        &mut world,
        "Measure",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Measure"),
        Some(node)
    );
    select_dock_tab(&mut world, "Measure");
    let close = world.run_in_window("Measure", "Done").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(close));
    assert_closed_with_visible_tab(&mut world, "Measure", "Material Browser");
    world.close();
}

pub(super) fn assert_closed_with_visible_tab(
    world: &mut ApplicationWorld,
    closed: &str,
    visible: &str,
) {
    for _ in 0..8 {
        let app = world.app();
        crate::frames::run_frames(
            &mut app.window,
            crate::HeadOptions { frames: Some(1) },
            &mut app.owner,
        )
        .unwrap();
        if window_geometry(closed).is_none() {
            assert_eq!(
                super::super::diagnostics::window_dock_tab_visible(visible),
                Some(true),
                "{visible}: expected the remaining tab in the first frame after {closed} closed"
            );
            world.frame = super::super::diagnostics::rendered_frame();
            let content = match visible {
                "Material Browser" => "Assigntopart",
                "Parameters" => "UserParameters",
                _ => panic!("{visible}: expected a known dock-tab content witness"),
            };
            world.run_in_window(visible, content);
            return;
        }
    }
    panic!("{closed}: expected close within eight frames");
}

#[test]
fn browser_feature_properties_survives_search_filter() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    super::browser_feature::right_click_row(&mut world, super::browser_feature::FEATURE);
    super::browser_feature::choose_menu_row(&mut world, "Properties");
    assert!(window_geometry("Feature Properties").is_some());
    world.click_text("Search");
    world.pump(Vec::new());
    let field = world.run("Type Keyword").center();
    world.click(super::super::diagnostics::pointer_pixels(field));
    world.type_text("Extrude");
    world.run("Extrude");
    assert!(
        window_geometry("Feature Properties").is_some(),
        "Feature Properties: expected the independent window to remain visible while filtering the browser"
    );
    world.run_in_window("Feature Properties", "Name");
    world.close();
}

#[test]
fn restored_project_dock_survives_parent_closure() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let area = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    drag_title(
        &mut world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    world.key(oblikovati_viewport::ViewportPhysicalKey::N, true);
    world.click_text("Projects");
    world.pump(Vec::new());
    let title = "Projects###template-projects";
    assert!(window_geometry(title).is_some());
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    let title_c = std::ffi::CString::new(title).unwrap();
    unsafe {
        imgui::sys::igDockBuilderDockWindow(title_c.as_ptr(), node);
        imgui::sys::igDockBuilderFinish(node);
    }
    world.pump(Vec::new());
    assert_eq!(super::super::diagnostics::window_dock_id(title), Some(node));
    select_dock_tab(&mut world, title);
    let cancel = world.run_in_window("New document", "Cancel").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(cancel));
    assert_closed_with_visible_tab(&mut world, title, "Material Browser");
    world.close();
}

#[test]
fn native_browser_search_hint_filters_and_restores_the_tree() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    super::dockspace_restart::native_click_label(&mut world, "Search");
    world.expect_text("Type Keyword");
    world.expect_text("Origin");
    super::dockspace_restart::native_click_label(&mut world, "Type Keyword");
    for character in "zzq".chars() {
        super::super::x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
    world.expect_text("No matching entries");
    super::dockspace_restart::native_click_label(&mut world, "x");
    world.expect_text("Type Keyword");
    world.expect_text("Origin");
    super::dockspace_restart::native_click_label(&mut world, "x");
    world.expect_no_text("Type Keyword");
    world.expect_text("Origin");
    world.close();
}

#[test]
fn native_browser_search_expands_a_matching_parent_and_restores_tree() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    world.expect_text("X Axis");
    native_search_query(&mut world, "origin");
    world.expect_text("Origin");
    world.expect_no_text("X Axis");
    assert!(
        world.text_is_selected("Origin"),
        "matching parent text must be highlighted"
    );
    let row = world.run("Origin");
    let point = [row.minimum[0] - 10., row.center()[1]];
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    for label in ["XY Plane", "X Axis", "Center Point"] {
        world.expect_text(label);
    }
    assert!(
        world.run("X Axis").alpha < world.run("Origin").alpha,
        "unmatched children must use dimmed text"
    );
    super::dockspace_restart::native_click_label(&mut world, "x");
    super::dockspace_restart::native_click_label(&mut world, "x");
    world.expect_text("X Axis");
    world.close();
}

#[test]
fn native_browser_search_origin_axis_keeps_selection_after_clear() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    native_search_query(&mut world, "axis");
    super::dockspace_restart::native_click_label(&mut world, "X Axis");
    super::browser_feature::expect_outline(&mut world, true);
    super::dockspace_restart::native_click_label(&mut world, "x");
    world.expect_text("XY Plane");
    super::browser_feature::expect_outline(&mut world, true);
    world.close();
}

fn native_search_query(world: &mut ApplicationWorld, query: &str) {
    super::dockspace_restart::native_click_label(world, "Search");
    super::dockspace_restart::native_click_label(world, "Type Keyword");
    for character in query.chars() {
        super::super::x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
}
