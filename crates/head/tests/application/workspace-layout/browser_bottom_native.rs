// SPDX-License-Identifier: GPL-2.0-only
//! Check tab-row alignment for outer and inner bottom docking targets.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::*;

fn dock_bottom(world: &mut ApplicationWorld, target: [i32; 2], capture: &str) {
    open_toolbar_tool(world, "Material Browser");
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    super::super::x11_input::move_pointer(start);
    world.pump(Vec::new());
    super::super::x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        let at = std::array::from_fn(|axis| start[axis] + (target[axis] - start[axis]) * step / 8);
        super::super::x11_input::move_pointer(at);
        world.pump(Vec::new());
    }
    world.save_capture(capture);
    super::super::x11_input::left_button(target, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser").unwrap(),
        0
    );
}

#[test]
fn outer_bottom_panel_has_a_full_width_document_row() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    let status = window_geometry("##status-bar").unwrap().0;
    let target = [(world.extent()[0] / 2) as i32, status[1] as i32 - 15];
    dock_bottom(&mut world, target, "outer-target-preview");
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
fn inner_bottom_panel_preserves_browser_and_document_row() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let area = viewport(&mut world);
    let target = [((area[0] + area[2]) / 2) as i32, area[3] as i32 - 15];
    dock_bottom(&mut world, target, "inner-target-preview");
    let (panel, panel_size) = window_geometry("Material Browser").unwrap();
    let (tabs, _) = window_geometry("##open-document-tabs").unwrap();
    let (list, list_size) = window_geometry("##document-list-control").unwrap();
    let (browser, browser_size) = window_geometry("Model").unwrap();
    let status = window_geometry("##status-bar").unwrap().0;
    world.save_capture("partial-bottom-panel");
    assert!(
        (browser[1] + browser_size[1] - status[1]).abs() <= 2.,
        "browser {browser:?} size {browser_size:?}, status {status:?}"
    );
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

#[test]
fn scaled_outer_top_target_keeps_its_dock_behavior() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(1.5, 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    let (panel, size) = window_geometry("Material Browser").unwrap();
    let (outer, bounds) = window_geometry("##workspace-outer-host").unwrap();
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.unwrap();
    let half =
        (context.FontSize * 1.5).min((context.FontSize * 0.5).max(bounds[0].min(bounds[1]) / 8.));
    let narrow = (half * 0.8).floor();
    let from =
        super::super::diagnostics::pointer_pixels([panel[0] + size[0] * 0.4, panel[1] + 10.]);
    let target =
        super::super::diagnostics::pointer_pixels([outer[0] + bounds[0] * 0.5, outer[1] + narrow]);
    super::dockspace_native::native_drag(&mut world, from, target);
    let (panel, _) = window_geometry("Material Browser").unwrap();
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser").unwrap(),
        0
    );
    assert!(
        (panel[1] - outer[1]).abs() <= 1.,
        "panel {panel:?}, outer {outer:?}"
    );
    world.close();
}
