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

#[test]
fn origin_auto_resize_is_a_per_datum_native_menu_setting() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    native_menu(&mut world, "X Axis");
    assert!(world.run_is_enabled("Auto-Resize"));
    assert!(world.checked("Auto-Resize"));
    native_click(&mut world, "Auto-Resize");
    native_menu(&mut world, "X Axis");
    assert!(!world.checked("Auto-Resize"));
    x11_input::press_escape();
    settle(&mut world);
    native_menu(&mut world, "YZ Plane");
    assert!(world.checked("Auto-Resize"));
    native_click(&mut world, "Auto-Resize");
    native_menu(&mut world, "YZ Plane");
    assert!(!world.checked("Auto-Resize"));
    world.close();
}

#[test]
fn origin_axis_auto_resize_changes_visible_extent_on_a_large_part() {
    let mut world = ApplicationWorld::new();
    open_large_origin_fixture(&mut world);
    native_menu(&mut world, "X Axis");
    assert!(world.checked("Auto-Resize"));
    let automatic = blue_axis_pixels(&mut world);
    native_click(&mut world, "Auto-Resize");
    let fixed = blue_axis_pixels(&mut world);
    assert!(
        automatic > fixed,
        "automatic axis pixels {automatic}: expected more than fixed {fixed}"
    );
    native_menu(&mut world, "X Axis");
    assert!(!world.checked("Auto-Resize"));
    native_click(&mut world, "Auto-Resize");
    assert!(blue_axis_pixels(&mut world) > fixed);
    world.close();
}

fn open_large_origin_fixture(world: &mut ApplicationWorld) {
    let bytes = super::super::super::workspace_layout::browser_feature::extruded_part_bytes(
        "Extrusion",
        "Profile",
        20.,
    );
    let path = world.scratch_folder().join("large-origin.opd");
    std::fs::write(&path, bytes).unwrap();
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    settle(world);
}

fn blue_axis_pixels(world: &mut ApplicationWorld) -> usize {
    super::super::super::workspace_layout::browser_feature::colored_pixels(world, [64, 153, 255])
}

#[test]
fn presentation_bounds_are_reused_between_frames_and_refreshed_for_new_scene() {
    let mut world = ApplicationWorld::new();
    open_large_origin_fixture(&mut world);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    let before = world
        .app()
        .window
        .native
        .presentation_bounds_refresh_count();
    let bounds = crate::HeadWindow::presentation_bounds(&world.app().window);
    assert!(
        bounds.is_some(),
        "large part must publish presentation bounds"
    );
    for _ in 0..5 {
        world.pump(Vec::new());
    }
    assert_eq!(
        world
            .app()
            .window
            .native
            .presentation_bounds_refresh_count(),
        before
    );
    world.add_part();
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert!(
        world
            .app()
            .window
            .native
            .presentation_bounds_refresh_count()
            > before
    );
    world.close();
}

#[test]
fn presentation_bounds_refresh_for_sketch_geometry_and_remain_cached() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.start_sketch();
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    let before = world
        .app()
        .window
        .native
        .presentation_bounds_refresh_count();
    world.draw_rectangle();
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    let after = world
        .app()
        .window
        .native
        .presentation_bounds_refresh_count();
    assert!(after > before, "sketch geometry change must refresh bounds");
    for _ in 0..5 {
        world.pump(Vec::new());
    }
    assert_eq!(
        world
            .app()
            .window
            .native
            .presentation_bounds_refresh_count(),
        after
    );
    world.close();
}
