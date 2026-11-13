// SPDX-License-Identifier: GPL-2.0-only
//! Drive shared dockable windows with the native pointer.
use super::*;

fn close_title(world: &mut ApplicationWorld, title: &str) {
    let (position, size) = window_geometry(title).expect("expected a visible tool window");
    let center = [position[0] + size[0] - 13., position[1] + 10.];
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[test]
fn dockable_material_browser_closes_and_reopens_from_the_title_bar() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    world.click(row_center(&world, "MaterialBrowser"));
    assert!(window_geometry("Material Browser").is_some());
    close_title(&mut world, "Material Browser");
    assert!(window_geometry("Material Browser").is_none());
    part_stays(&mut world);
    world.click(row_center(&world, "MaterialBrowser"));
    assert!(window_geometry("Material Browser").is_some());
    close_title(&mut world, "Material Browser");
    world.close();
}

#[test]
fn dockable_tool_windows_can_be_docked_together() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    world.click_text("Tools");
    world.click_text("Application Options");
    world.click(row_center(&world, "MaterialBrowser"));
    let (source, source_size) = window_geometry("Material Browser").unwrap();
    let (target, target_size) = window_geometry("Application Options").unwrap();
    world.save_capture("dockable-before-drag");
    let from = [source[0] + source_size[0] * 0.4, source[1] + 10.];
    let to = [target[0] + target_size[0] * 0.5, target[1] + 10.];
    world.drag(
        super::super::diagnostics::pointer_pixels(from),
        super::super::diagnostics::pointer_pixels(to),
    );
    world.pump(Vec::new());
    world.save_capture("dockable-after-drag");
    let first = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    let second = super::super::diagnostics::window_dock_id("Application Options").unwrap();
    let mouse_down = unsafe { imgui::sys::igGetIO().as_ref().unwrap().MouseDown[0] };
    assert!(
        first != 0 && first == second,
        "expected one shared dock node, got {first} and {second}; mouse down {mouse_down}; source {source:?} {source_size:?}, target {target:?} {target_size:?}, after {:?}",
        window_geometry("Material Browser")
    );
    world.app().window.native.view.resize([1000, 700]).unwrap();
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(first)
    );
    assert_eq!(
        super::super::diagnostics::window_dock_id("Application Options"),
        Some(first)
    );
    let (minimum, maximum) = super::super::diagnostics::window_dock_tab_bounds("Material Browser")
        .expect("expected the browser's dock tab");
    let height = maximum[1] - minimum[1];
    let close = [maximum[0] - height * 0.5, (minimum[1] + maximum[1]) * 0.5];
    world.click(super::super::diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.save_capture("dockable-after-close");
    assert!(window_geometry("Material Browser").is_none());
    assert!(window_geometry("Application Options").is_some());
    world.close();
}
