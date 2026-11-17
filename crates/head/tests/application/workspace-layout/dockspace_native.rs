// SPDX-License-Identifier: GPL-2.0-only
//! Drive a tool into the shared workspace dockspace with the native pointer.
use super::*;

fn viewport(world: &mut ApplicationWorld) -> [u32; 4] {
    let extent = world.extent();
    world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .unwrap()
        .document_area(extent)
        .unwrap()
}

#[test]
fn material_browser_docks_at_workspace_right_and_reduces_viewport() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    let icon = super::super::quick_access_icons::standard_icon_center("Material Browser").unwrap();
    world.click(super::super::diagnostics::pointer_pixels(icon));
    let (position, size) = window_geometry("Material Browser").expect("floating material browser");
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let end = [before[2] as i32 - 15, ((before[1] + before[3]) / 2) as i32];
    super::super::x11_input::move_pointer(start);
    world.pump(Vec::new());
    super::super::x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        let at = std::array::from_fn(|axis| start[axis] + (end[axis] - start[axis]) * step / 8);
        super::super::x11_input::move_pointer(at);
        world.pump(Vec::new());
    }
    world.save_capture("workspace-docking-preview");
    let preview = super::super::capture_framebuffer(&mut world.app().window.native);
    let index = ((before[1] + 40) * preview.extent[0] + before[2] - 60) as usize * 4;
    let color = &preview.rgba[index..index + 3];
    assert!(
        color[2] > color[0] + 20,
        "expected a blue docking preview, got {color:?}"
    );
    super::super::x11_input::left_button(end, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.save_capture("workspace-docking-result");
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0, "expected the browser in a dock node");
    let after = viewport(&mut world);
    assert!(after[2] < before[2], "viewport {before:?} -> {after:?}");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
    let (list, list_size) = window_geometry("##document-list-control").unwrap();
    assert!((list[0] + list_size[0] - after[2] as f32).abs() <= 1.);
    resize_boundary(&mut world, after);
    let resized = viewport(&mut world);
    assert!(
        resized[2] < after[2],
        "expected the panel to grow: {after:?} -> {resized:?}"
    );
    let (minimum, maximum) =
        super::super::diagnostics::window_dock_tab_bounds("Material Browser").unwrap();
    let close = [
        maximum[0] - (maximum[1] - minimum[1]) * 0.5,
        (minimum[1] + maximum[1]) * 0.5,
    ];
    world.click(super::super::diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(window_geometry("Material Browser").is_none());
    assert_eq!(
        viewport(&mut world),
        before,
        "closing the panel must restore the viewport"
    );
    world.close();
}

fn resize_boundary(world: &mut ApplicationWorld, area: [u32; 4]) {
    let start = [area[2] as i32 + 2, ((area[1] + area[3]) / 2) as i32];
    let end = [start[0] - 80, start[1]];
    super::super::x11_input::move_pointer(start);
    world.pump(Vec::new());
    super::super::x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        super::super::x11_input::move_pointer([
            start[0] + (end[0] - start[0]) * step / 8,
            start[1],
        ]);
        world.pump(Vec::new());
    }
    super::super::x11_input::left_button(end, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
