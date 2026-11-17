// SPDX-License-Identifier: GPL-2.0-only
//! Drive a tool into the shared workspace dockspace with the native pointer.
use super::*;

pub(in super::super) fn viewport(world: &mut ApplicationWorld) -> [u32; 4] {
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
    resize_boundary(&mut world, after, -80);
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
    world.click_text("View");
    world.click_text("User Interface");
    world.click_text("Material Browser");
    world.pump(Vec::new());
    assert!(
        window_geometry("Material Browser").is_some(),
        "menu must reopen the panel"
    );
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser").unwrap(),
        0,
        "reopened panel must return to its dock"
    );
    world.save_capture("workspace-docking-menu-restored");
    world.close();
}

pub(in super::super) fn resize_boundary(world: &mut ApplicationWorld, area: [u32; 4], delta: i32) {
    let start = [area[2] as i32 + 2, ((area[1] + area[3]) / 2) as i32];
    let end = [start[0] + delta, start[1]];
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

#[test]
fn floating_tool_docks_to_each_remaining_workspace_edge() {
    for edge in ["left", "top", "bottom"] {
        check_workspace_edge(edge);
    }
}

fn check_workspace_edge(edge: &str) {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    let icon = super::super::quick_access_icons::standard_icon_center("Material Browser").unwrap();
    world.click(super::super::diagnostics::pointer_pixels(icon));
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let mid = [
        ((before[0] + before[2]) / 2) as i32,
        ((before[1] + before[3]) / 2) as i32,
    ];
    let target = match edge {
        "left" => [15, mid[1]],
        "top" => [(world.extent()[0] / 2) as i32, before[1] as i32 + 15],
        "bottom" => [
            (world.extent()[0] / 2) as i32,
            window_geometry("##status-bar").unwrap().0[1] as i32 - 15,
        ],
        _ => panic!("edge {edge:?}: expected left, top, or bottom"),
    };
    native_drag(&mut world, start, target);
    world.save_capture(&format!("workspace-dock-{edge}-attempt"));
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0, "{edge}: expected a docked panel");
    let after = viewport(&mut world);
    let changed = match edge {
        "left" => after[0] > before[0],
        "top" => after[1] > before[1],
        "bottom" => after[3] < before[3],
        _ => false,
    };
    assert!(changed, "{edge}: viewport {before:?} -> {after:?}");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_solid_fill(&capture, &windows),
        "{edge}: expected visible solid"
    );
    world.save_capture(&format!("workspace-dock-{edge}"));
    world.close();
}

pub(in super::super) fn native_drag(world: &mut ApplicationWorld, start: [i32; 2], end: [i32; 2]) {
    super::super::x11_input::move_pointer(start);
    world.pump(Vec::new());
    super::super::x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=8 {
        let at = std::array::from_fn(|axis| start[axis] + (end[axis] - start[axis]) * step / 8);
        super::super::x11_input::move_pointer(at);
        world.pump(Vec::new());
    }
    super::super::x11_input::left_button(end, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[test]
fn tool_windows_join_as_tabs_in_a_workspace_dock() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        &mut world,
        start,
        [before[2] as i32 - 15, ((before[1] + before[3]) / 2) as i32],
    );
    open_toolbar_tool(&mut world, "Appearance Browser");
    let (source, source_size) = window_geometry("Appearance Browser").unwrap();
    let (destination, destination_size) = window_geometry("Material Browser").unwrap();
    let start = [
        (source[0] + source_size[0] * 0.4) as i32,
        (source[1] + 10.) as i32,
    ];
    let target = [
        (destination[0] + destination_size[0] * 0.5) as i32,
        (destination[1] + 10.) as i32,
    ];
    native_drag(&mut world, start, target);
    let material = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    let appearance = super::super::diagnostics::window_dock_id("Appearance Browser").unwrap();
    assert!(
        material != 0 && material == appearance,
        "expected one dock node: {material}, {appearance}"
    );
    let (minimum, maximum) =
        super::super::diagnostics::window_dock_tab_bounds("Material Browser").unwrap();
    world.click(super::super::diagnostics::pointer_pixels([
        (minimum[0] + maximum[0]) * 0.5,
        (minimum[1] + maximum[1]) * 0.5,
    ]));
    world.pump(Vec::new());
    world.run_in_window("Material Browser", "Assigntopart");
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Material Browser"),
        Some(true)
    );
    let (minimum, maximum) =
        super::super::diagnostics::window_dock_tab_bounds("Appearance Browser").unwrap();
    world.click(super::super::diagnostics::pointer_pixels([
        (minimum[0] + maximum[0]) * 0.5,
        (minimum[1] + maximum[1]) * 0.5,
    ]));
    world.pump(Vec::new());
    world.run_in_window("Appearance Browser", "Assigntopart");
    assert_eq!(
        super::super::diagnostics::window_dock_tab_visible("Appearance Browser"),
        Some(true)
    );
    assert!(viewport(&mut world)[2] < before[2]);
    world.save_capture("workspace-dock-tab-group");
    world.close();
}

pub(in super::super) fn open_toolbar_tool(world: &mut ApplicationWorld, label: &str) {
    let icon = super::super::quick_access_icons::standard_icon_center(label).unwrap();
    world.click(super::super::diagnostics::pointer_pixels(icon));
    world.pump(Vec::new());
}

#[test]
fn docked_tool_can_float_and_return_to_its_workspace_edge() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let before = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let target = [before[2] as i32 - 15, ((before[1] + before[3]) / 2) as i32];
    native_drag(&mut world, start, target);
    let (minimum, maximum) =
        super::super::diagnostics::window_dock_tab_bounds("Material Browser").unwrap();
    let tab = [
        ((minimum[0] + maximum[0]) * 0.5) as i32,
        ((minimum[1] + maximum[1]) * 0.5) as i32,
    ];
    let floating = [
        ((before[0] + before[2]) / 2) as i32,
        ((before[1] + before[3]) / 2) as i32,
    ];
    native_drag(&mut world, tab, floating);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(0)
    );
    assert_eq!(
        viewport(&mut world),
        before,
        "floating the panel must free the viewport"
    );
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(&mut world, start, target);
    assert_ne!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(0)
    );
    assert!(viewport(&mut world)[2] < before[2]);
    world.save_capture("workspace-dock-redocked");
    world.close();
}

#[test]
fn docked_tool_retains_its_node_when_documents_change() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let first = world.active_id();
    let initial = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    let (position, size) = window_geometry("Material Browser").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        &mut world,
        start,
        [
            initial[2] as i32 - 15,
            ((initial[1] + initial[3]) / 2) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    let docked_area = viewport(&mut world);
    let second = world.add_part();
    assert_ne!(first, second);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(node)
    );
    assert_eq!(viewport(&mut world), docked_area);
    let name = world.document_name(first);
    let point = super::super::diagnostics::pointer_pixels(world.expect_tab(&name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(first));
    assert_eq!(
        super::super::diagnostics::window_dock_id("Material Browser"),
        Some(node)
    );
    assert_eq!(viewport(&mut world), docked_area);
    world.close();
}

#[test]
fn docked_parameters_stays_open_when_switching_parts() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let first = world.active_id();
    let initial = viewport(&mut world);
    open_toolbar_tool(&mut world, "Parameters");
    let (position, size) = window_geometry("Parameters").unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        &mut world,
        start,
        [
            initial[2] as i32 - 15,
            ((initial[1] + initial[3]) / 2) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Parameters").unwrap();
    assert_ne!(node, 0);
    let docked_area = viewport(&mut world);
    let second = world.add_part();
    assert_ne!(first, second);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Parameters"),
        Some(node)
    );
    assert_eq!(viewport(&mut world), docked_area);
    world.run_in_window("Parameters", "UserParameters");
    let name = world.document_name(first);
    let point = super::super::diagnostics::pointer_pixels(world.expect_tab(&name).center());
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(world.state().document, Some(first));
    assert_eq!(
        super::super::diagnostics::window_dock_id("Parameters"),
        Some(node)
    );
    assert_eq!(viewport(&mut world), docked_area);
    world.close();
}
