// SPDX-License-Identifier: GPL-2.0-only
//! Check tab-row alignment for outer and inner bottom docking targets.
use super::dockspace_native::{native_drag, open_toolbar_tool, viewport};
use super::*;

fn dock_bottom(world: &mut ApplicationWorld, target: [i32; 2]) {
    open_toolbar_tool(world, "Material Browser");
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(world, start, target);
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser").unwrap(),
        0
    );
}

#[test]
fn outer_bottom_panel_has_a_full_width_document_row() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    let status = window_geometry("##status-bar").unwrap().0;
    let target = [(world.extent()[0] / 2) as i32, status[1] as i32 - 15];
    dock_bottom(&mut world, target);
    let (panel, panel_size) = window_geometry("Material Browser").unwrap();
    let (tabs, _) = window_geometry("##open-document-tabs").unwrap();
    let (list, list_size) = window_geometry("##document-list-control").unwrap();
    world.save_capture("outer-bottom-panel");
    assert!(viewport(&mut world)[3] < before[3]);
    assert_rows_touch(panel, panel_size, tabs);
    assert!(
        (tabs[0] - panel[0]).abs() <= 1.,
        "tabs {tabs:?}, panel {panel:?}"
    );
    assert!((list[0] + list_size[0] - panel[0] - panel_size[0]).abs() <= 1.);
    world.close();
}

#[test]
fn partial_outer_bottom_panel_has_a_matching_document_row() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let area = viewport(&mut world);
    let target = [
        ((area[0] + area[2]) / 2) as i32,
        ((area[1] + area[3]) / 2) as i32 + 46,
    ];
    dock_bottom(&mut world, target);
    let (panel, panel_size) = window_geometry("Material Browser").unwrap();
    let (tabs, _) = window_geometry("##open-document-tabs").unwrap();
    let (list, list_size) = window_geometry("##document-list-control").unwrap();
    world.save_capture("partial-bottom-panel");
    assert!(viewport(&mut world)[3] < area[3]);
    assert_rows_touch(panel, panel_size, tabs);
    assert!(
        (tabs[0] - panel[0]).abs() <= 1.,
        "tabs {tabs:?}, panel {panel:?}"
    );
    assert!((list[0] + list_size[0] - panel[0] - panel_size[0]).abs() <= 1.);
    world.close();
}

fn assert_rows_touch(panel: [f32; 2], panel_size: [f32; 2], tabs: [f32; 2]) {
    let (_, tabs_size) = window_geometry("##open-document-tabs").unwrap();
    let status = window_geometry("##status-bar").unwrap().0;
    assert!((panel[1] + panel_size[1] - tabs[1]).abs() <= 2.);
    assert!((tabs[1] + tabs_size[1] - status[1]).abs() <= 2.);
}
