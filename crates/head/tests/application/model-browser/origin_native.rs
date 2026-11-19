// SPDX-License-Identifier: GPL-2.0-only
//! Exercise origin context menus through native X11 pointer input (#4115).
use super::super::super::x11_input;
use super::*;

fn native_click(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(world.run(label).center());
    x11_input::click(at);
    settle(world);
}

fn native_menu(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(world.run(label).center());
    x11_input::right_click(at);
    settle(world);
}

fn clear_selection(world: &mut ApplicationWorld) {
    x11_input::click([1300, 800]);
    settle(world);
}

#[test]
fn origin_plane_visibility_round_trips_through_native_menu() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    yz_hidden_oblique(&mut world);
    native_menu(&mut world, YZ);
    plane_menu(&mut world);
    assert!(!world.checked("Visibility"));
    native_click(&mut world, "Visibility");
    clear_selection(&mut world);
    plane_stays(&mut world);
    world.save_capture("origin-plane-visible");
    native_menu(&mut world, YZ);
    assert!(world.checked("Visibility"));
    native_click(&mut world, "Visibility");
    clear_selection(&mut world);
    assert!(!plane_shows_at(&mut world, YZ));
    expect_solid_visible(&mut world);
    world.close();
}

#[test]
fn origin_axis_visibility_round_trips_through_native_menu() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    x_axis_hidden(&mut world);
    native_menu(&mut world, "X Axis");
    axis_lit(&mut world);
    axis_menu_marks(&mut world);
    native_click(&mut world, "Visibility");
    clear_selection(&mut world);
    axis_stays(&mut world);
    native_menu(&mut world, "X Axis");
    assert!(world.checked("Visibility"));
    native_click(&mut world, "Visibility");
    clear_selection(&mut world);
    axis_hidden(&mut world);
    world.close();
}

#[test]
fn origin_center_point_native_menu_identifies_row_and_commands() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    native_menu(&mut world, "Center Point");
    assert!(row_accent(&mut world, "Center Point"));
    point_rows(&mut world);
    relationships_row(&mut world);
    world.save_capture("origin-point-menu");
    x11_input::press_escape();
    settle(&mut world);
    expect_block_kept(&mut world);
    world.close();
}

#[test]
fn origin_folder_native_collapse_preserves_direct_rows() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    native_menu(&mut world, "Origin");
    expect_rows(&world, &["Expand All Children", "Collapse All Children"]);
    native_click(&mut world, "Collapse All Children");
    expect_rows(&world, &["YZ Plane", "X Axis", "Center Point"]);
    let origin = world.run("Origin");
    let arrow = [origin.minimum[0] - 10., origin.center()[1]];
    x11_input::click(diagnostics::pointer_pixels(arrow));
    settle(&mut world);
    world.expect_no_text("YZ Plane");
    x11_input::click(diagnostics::pointer_pixels(arrow));
    settle(&mut world);
    expect_rows(&world, &["YZ Plane", "X Axis", "Center Point"]);
    expect_block_kept(&mut world);
    world.close();
}
