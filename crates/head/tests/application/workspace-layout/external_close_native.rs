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
    super::dock_search::click_model_search(&mut world);
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
    let mut world = ApplicationWorld::with_asset_tools_closed();
    world.add_part();
    let node = dock_material_browser(&mut world);
    open_template_projects(&mut world);
    restore_template_project_dock(&mut world, node);
    select_dock_tab(&mut world, TEMPLATE_PROJECTS);
    close_template_chooser(&mut world);
    world.save_capture("chooser-close-reveals-material");
    world.close();
}

fn open_template_projects(world: &mut ApplicationWorld) {
    world.key(oblikovati_viewport::ViewportPhysicalKey::N, true);
    super::super::project_dialog::choose_projects(world);
    assert!(window_geometry(TEMPLATE_PROJECTS).is_some());
}

fn restore_template_project_dock(world: &mut ApplicationWorld, node: u32) {
    let title = std::ffi::CString::new(TEMPLATE_PROJECTS).unwrap();
    // Seed the saved child placement without requiring a drag out of a modal.
    unsafe {
        imgui::sys::igDockBuilderDockWindow(title.as_ptr(), node);
        imgui::sys::igDockBuilderFinish(node);
    }
    world.pump(Vec::new());
    assert_eq!(
        super::super::diagnostics::window_dock_id(TEMPLATE_PROJECTS),
        Some(node)
    );
}

fn close_template_chooser(world: &mut ApplicationWorld) {
    click_before_dock_frame(world, "New document", "Cancel");
    assert_closed_with_visible_tab(world, TEMPLATE_PROJECTS, MATERIAL_BROWSER);
    assert!(window_geometry("New document").is_none());
}

fn click_before_dock_frame(world: &mut ApplicationWorld, dialog: &str, label: &str) {
    let center = world.run_in_window(dialog, label).center();
    let point = super::super::diagnostics::pointer_pixels(center);
    if world.app().window.native.view.uses_wayland() {
        world.app().window.scripted = super::super::pointer_click_events(point, 1);
    } else {
        super::super::x11_input::click(point);
    }
}

#[test]
fn docked_projects_reveals_the_other_tab_and_restores_its_dock() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    let part = world.add_part();
    let path = world.scratch_folder().join("dock-project.opj");
    std::fs::write(
        &path,
        "schemaVersion: 1\nname: Dock Project\nworkspace: work\n",
    )
    .unwrap();
    world.app().owner.load_desktop_project(&path).unwrap();
    world.pump(Vec::new());

    let node = dock_standalone_projects(&mut world);
    for close in ["Done", "Escape"] {
        close_and_reopen_projects(&mut world, node, close);
    }
    world.expect_active(part);
    world.save_capture("projects-restored-to-shared-dock");
    select_dock_tab(&mut world, PROJECTS);
    click_before_dock_frame(&mut world, PROJECTS, "Done");
    assert_closed_with_visible_tab(&mut world, PROJECTS, MATERIAL_BROWSER);
    check_projects_after_restart(&mut world, &path, node);
    world.close();
}

fn check_projects_after_restart(world: &mut ApplicationWorld, path: &std::path::Path, node: u32) {
    world.reopen_application();
    let part = world.add_part();
    world.app().owner.load_desktop_project(path).unwrap();
    world.pump(Vec::new());
    assert_eq!(material_node(), node);
    open_toolbar_tool(world, "Projects");
    assert_eq!(
        super::super::diagnostics::window_dock_id(PROJECTS),
        Some(node)
    );
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible(PROJECTS),
        Some(true),
        "opening standalone Projects should reveal its restored dock tab"
    );
    world.run_in_window(PROJECTS, "ProjectSettings");
    click_before_dock_frame(world, PROJECTS, "Done");
    assert_closed_with_visible_tab(world, PROJECTS, MATERIAL_BROWSER);
    world.expect_active(part);
}

const PROJECTS: &str = "Projects###file-projects";
const TEMPLATE_PROJECTS: &str = "Projects###template-projects";
const MATERIAL_BROWSER: &str = "Material Browser";

fn dock_standalone_projects(world: &mut ApplicationWorld) -> u32 {
    let node = dock_material_browser(world);
    open_toolbar_tool(world, "Projects");
    let (position, size) = window_geometry(MATERIAL_BROWSER).unwrap();
    drag_title(
        world,
        PROJECTS,
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    assert_eq!(
        super::super::diagnostics::window_dock_id(PROJECTS),
        Some(node)
    );
    node
}

fn dock_material_browser(world: &mut ApplicationWorld) -> u32 {
    let area = viewport(world);
    open_toolbar_tool(world, MATERIAL_BROWSER);
    drag_title(
        world,
        MATERIAL_BROWSER,
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    material_node()
}

fn material_node() -> u32 {
    let node = super::super::diagnostics::window_dock_id(MATERIAL_BROWSER).unwrap();
    assert_ne!(node, 0);
    node
}

fn close_and_reopen_projects(world: &mut ApplicationWorld, node: u32, close: &str) {
    select_dock_tab(world, PROJECTS);
    world.run_in_window(PROJECTS, "ProjectSettings");
    if close == "Done" {
        click_before_dock_frame(world, PROJECTS, "Done");
    } else {
        escape_before_dock_frame(world);
    }
    assert_closed_with_visible_tab(world, PROJECTS, MATERIAL_BROWSER);
    open_toolbar_tool(world, "Projects");
    assert_eq!(
        super::super::diagnostics::window_dock_id(PROJECTS),
        Some(node)
    );
}

fn escape_before_dock_frame(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey};
    if !world.app().window.native.view.uses_wayland() {
        super::super::x11_input::press_escape();
        return;
    }
    world.app().window.scripted = [true, false]
        .into_iter()
        .map(|pressed| {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Escape,
                layout_key: ViewportPhysicalKey::Escape,
                pressed,
                command: Some(ViewportKey::Cancel),
                modifiers: Default::default(),
            })
        })
        .collect();
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
