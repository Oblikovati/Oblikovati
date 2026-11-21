// SPDX-License-Identifier: GPL-2.0-only
//! Verify that a native divider drag changes both browser and viewport.
use super::super::super::workspace_layout::dockspace_native::{native_drag, viewport};
use super::*;

#[test]
fn native_browser_resize_changes_the_viewport() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    expect_left_docked(&mut world);
    let before = viewport(&mut world);
    let original_width = browser_geometry().1[0];
    let delta = (world.extent()[0] / 12) as i32;
    drag_boundary(&mut world, before, delta);
    let widened = viewport(&mut world);
    expect_browser_widened(&mut world, before, widened, original_width);
    drag_boundary(&mut world, widened, -delta);
    let restored = viewport(&mut world);
    expect_browser_restored(before, restored, original_width);
    world.close();
}

fn expect_browser_restored(before: [u32; 4], restored: [u32; 4], original_width: f32) {
    assert!(
        restored[0].abs_diff(before[0]) <= 2,
        "viewport {before:?} -> {restored:?}"
    );
    assert!((browser_geometry().1[0] - original_width).abs() <= 2.);
}

fn expect_browser_widened(
    world: &mut ApplicationWorld,
    before: [u32; 4],
    widened: [u32; 4],
    original_width: f32,
) {
    let widened_width = browser_geometry().1[0];
    assert!(
        widened_width > original_width,
        "browser width {original_width} -> {widened_width}"
    );
    assert!(widened[0] > before[0], "viewport {before:?} -> {widened:?}");
    assert_eq!(
        widened[2], before[2],
        "viewport right edge must remain fixed"
    );
    expect_block_kept(world);
}

fn drag_boundary(world: &mut ApplicationWorld, area: [u32; 4], delta: i32) {
    let start = [area[0] as i32 - 2, ((area[1] + area[3]) / 2) as i32];
    native_drag(world, start, [start[0] + delta, start[1]]);
}

#[test]
fn native_browser_redocks_on_both_workspace_edges() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    expect_left_docked(&mut world);
    let area = viewport(&mut world);
    drag_tab(
        &mut world,
        [
            ((area[0] + area[2]) / 2) as i32,
            ((area[1] + area[3]) / 2) as i32,
        ],
    );
    assert_eq!(diagnostics::window_dock_id("Model"), Some(0));
    redock_both_sides(&mut world);
    world.close();
}

fn redock_both_sides(world: &mut ApplicationWorld) {
    let left = edge_target(world, false);
    drag_tab(world, left);
    expect_left_docked(world);
    let right = edge_target(world, true);
    drag_tab(world, right);
    docks_right(world);
    tree_and_block(world);
    let left = edge_target(world, false);
    drag_tab(world, left);
    expect_left_docked(world);
}

fn drag_tab(world: &mut ApplicationWorld, target: [i32; 2]) {
    let start = diagnostics::pointer_pixels(tab_bar_of(world, "Model"));
    native_drag(world, start, target);
}
