// SPDX-License-Identifier: GPL-2.0-only
//! Native pointer checks for the model browser's shared dock.
use super::dockspace_native::{native_drag, viewport};
use super::*;

#[test]
fn model_browser_uses_a_dock_node_and_floats_from_its_header() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let (position, size) = window_geometry("Model").expect("visible model browser");
    let node = super::super::diagnostics::window_dock_id("Model").unwrap();
    assert_ne!(node, 0, "model browser must be in a shared dock node");
    let initial = viewport(&mut world);
    assert!((size[0] - world.extent()[0] as f32 * 0.25).abs() < 12.);
    assert!((initial[0] as f32 - position[0] - size[0]).abs() < 12.);
    let from = [(position[0] + 40.) as i32, (position[1] + 9.) as i32];
    native_drag(&mut world, from, [800, 400]);
    assert_eq!(super::super::diagnostics::window_dock_id("Model"), Some(0));
    let floating = viewport(&mut world);
    assert!(
        floating[0] < initial[0],
        "viewport {initial:?} -> {floating:?}"
    );
    world.save_capture("model-browser-undocked");
    let (position, _) = window_geometry("Model").unwrap();
    let from = [(position[0] + 40.) as i32, (position[1] + 9.) as i32];
    native_drag(
        &mut world,
        from,
        [15, ((initial[1] + initial[3]) / 2) as i32],
    );
    assert_ne!(
        super::super::diagnostics::window_dock_id("Model").unwrap(),
        0
    );
    let restored = viewport(&mut world);
    assert!(
        restored[0] > floating[0],
        "viewport {floating:?} -> {restored:?}"
    );
    let (position, size) = window_geometry("Model").unwrap();
    let (status, _) = window_geometry("##status-bar").unwrap();
    assert!(
        (position[1] + size[1] - status[1]).abs() <= 2.,
        "browser must extend to status area"
    );
    world.save_capture("model-browser-redocked");
    world.reopen_application();
    world.add_part();
    world.pump(Vec::new());
    assert_ne!(
        super::super::diagnostics::window_dock_id("Model").unwrap(),
        0
    );
    assert_eq!(
        viewport(&mut world),
        restored,
        "browser dock must survive restart"
    );
    world.close();
}

#[test]
fn tool_and_model_browser_can_join_one_dock_region() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    super::dockspace_native::open_toolbar_tool(&mut world, "Material Browser");
    let (source, source_size) = window_geometry("Material Browser").unwrap();
    let (destination, destination_size) = window_geometry("Model").unwrap();
    let from = [
        (source[0] + source_size[0] * 0.4) as i32,
        (source[1] + 10.) as i32,
    ];
    let to = [
        (destination[0] + destination_size[0] * 0.5) as i32,
        (destination[1] + 10.) as i32,
    ];
    native_drag(&mut world, from, to);
    let model = super::super::diagnostics::window_dock_id("Model").unwrap();
    assert_ne!(model, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(model)
    );
    world.save_capture("model-browser-shared-tabs");
    let (minimum, maximum) = super::super::diagnostics::window_dock_tab_bounds("Model").unwrap();
    let tab = [
        (minimum[0] + maximum[0]) * 0.5,
        (minimum[1] + maximum[1]) * 0.5,
    ];
    let from = super::super::diagnostics::pointer_pixels(tab);
    world.click(from);
    world.pump(Vec::new());
    native_drag(&mut world, from, [800, 400]);
    assert_eq!(super::super::diagnostics::window_dock_id("Model"), Some(0));
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser").unwrap(),
        0
    );
    world.close();
}

#[test]
fn model_browser_can_move_to_the_right_outer_edge() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    let (position, _) = window_geometry("Model").unwrap();
    let from = [(position[0] + 40.) as i32, (position[1] + 9.) as i32];
    let to = [
        world.extent()[0] as i32 - 15,
        ((before[1] + before[3]) / 2) as i32,
    ];
    native_drag(&mut world, from, to);
    let (position, size) = window_geometry("Model").unwrap();
    let (status, _) = window_geometry("##status-bar").unwrap();
    let after = viewport(&mut world);
    assert!((position[0] + size[0] - world.extent()[0] as f32).abs() <= 2.);
    assert!((position[1] + size[1] - status[1]).abs() <= 2.);
    assert!(
        after[0] < before[0] && after[2] < before[2],
        "viewport {before:?} -> {after:?}"
    );
    world.save_capture("model-browser-right-dock");
    world.close();
}

#[test]
fn hidden_model_browser_stays_hidden_after_restart() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Model Browser");
    world.pump(Vec::new());
    assert!(window_geometry("Model").is_none());
    world.reopen_application();
    world.add_part();
    world.pump(Vec::new());
    assert!(window_geometry("Model").is_none());
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Model Browser");
    world.pump(Vec::new());
    assert_ne!(
        super::super::diagnostics::window_dock_id("Model").unwrap(),
        0
    );
    world.close();
}
#[test]
fn model_browser_can_dock_above_and_below_the_viewport() {
    for edge in ["top", "bottom"] {
        let mut world = ApplicationWorld::with_asset_tools_closed();
        super::browser_feature::open_visible_extruded_part(&mut world);
        let before = viewport(&mut world);
        let (position, _) = window_geometry("Model").unwrap();
        let from = [(position[0] + 40.) as i32, (position[1] + 9.) as i32];
        let y = if edge == "top" {
            before[1] + 14
        } else {
            window_geometry("##status-bar").unwrap().0[1] as u32 - 15
        };
        let to = [(world.extent()[0] / 2) as i32, y as i32];
        if edge == "top" {
            let mid = [to[0], (before[1] + 140) as i32];
            super::dockspace_native::press_drag(&mut world, from);
            super::dockspace_native::move_drag(&mut world, from, mid);
            super::dockspace_native::move_drag(&mut world, mid, to);
            super::dockspace_native::release_drag(&mut world, to);
        } else {
            native_drag(&mut world, from, to);
        }
        let after = viewport(&mut world);
        world.save_capture(&format!("model-browser-{edge}-result"));
        assert_ne!(
            super::super::diagnostics::window_dock_id("Model").unwrap(),
            0,
            "{edge}: expected a dock node"
        );
        if edge == "top" {
            assert!(after[1] > before[1], "top: {before:?} -> {after:?}");
        } else {
            assert!(after[3] < before[3], "bottom: {before:?} -> {after:?}");
            let (panel, size) = window_geometry("Model").unwrap();
            let (tabs, _) = window_geometry("##open-document-tabs").unwrap();
            assert!(
                size[1] > 80.,
                "bottom panel must show its contents: {size:?}"
            );
            assert!(
                panel[1] + size[1] <= tabs[1] + 2.,
                "bottom panel must stay above document tabs"
            );
        }
        world.save_capture(&format!("model-browser-{edge}-dock"));
        world.close();
    }
}

#[test]
fn dragging_the_inner_model_tab_floats_the_browser() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let initial = viewport(&mut world);
    let (minimum, maximum) = super::super::diagnostics::window_dock_tab_bounds("Model").unwrap();
    let start = super::super::diagnostics::pointer_pixels([
        (minimum[0] + maximum[0]) * 0.5,
        (minimum[1] + maximum[1]) * 0.5,
    ]);
    native_drag(&mut world, start, [800, 400]);
    assert_eq!(super::super::diagnostics::window_dock_id("Model"), Some(0));
    let (floating, _) = window_geometry("Model").unwrap();
    assert!(
        floating[0] > initial[0] as f32,
        "browser must follow the pointer: {floating:?}"
    );
    let area = viewport(&mut world);
    assert!(area[0] < initial[0]);
    let start = [(floating[0] + 50.) as i32, (floating[1] + 10.) as i32];
    let end = [15, ((area[1] + area[3]) / 2) as i32];
    super::super::x11_input::move_pointer(start);
    world.pump(Vec::new());
    super::super::x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        let at = std::array::from_fn(|axis| start[axis] + (end[axis] - start[axis]) * step / 8);
        super::super::x11_input::move_pointer(at);
        world.pump(Vec::new());
    }
    world.save_capture("inner-model-tab-preview");
    super::super::x11_input::left_button(end, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.save_capture("inner-model-tab-redock");
    assert_ne!(
        super::super::diagnostics::window_dock_id("Model").unwrap(),
        0
    );
    assert!(viewport(&mut world)[0] > area[0]);
    world.close();
}
