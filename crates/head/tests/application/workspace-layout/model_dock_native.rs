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
    let mut world = ApplicationWorld::new();
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
        let mut world = ApplicationWorld::new();
        super::browser_feature::open_visible_extruded_part(&mut world);
        let before = viewport(&mut world);
        let (position, _) = window_geometry("Model").unwrap();
        let from = [(position[0] + 40.) as i32, (position[1] + 9.) as i32];
        let y = if edge == "top" {
            before[1] + 15
        } else {
            window_geometry("##status-bar").unwrap().0[1] as u32 - 15
        };
        let to = [(world.extent()[0] / 2) as i32, y as i32];
        native_drag(&mut world, from, to);
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
