// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the application-owned frame against the Wayland compositor.
use super::super::main_window_frame_native::{
    frame_control_center, press_maximize, pump_frame_controls, pump_until,
};
use super::super::{diagnostics, workspace_layout};
use super::{
    ApplicationWorld, controls_remain_in_bar, double_click_bar, maximize_tooltip, open_frame_part,
    restore_tooltip,
};

pub(in super::super) fn compositor_frame_bounds() -> Option<[i32; 4]> {
    let graph = compositor_scene_graph()?;
    Some(current_frame_bounds(&graph))
}

pub(super) fn expect_maximized_work_area() {
    let graph =
        compositor_scene_graph().expect("set OBK_BDD_WESTON_DEBUG for the Wayland work area");
    let work_area =
        scene_work_area(&graph).expect("expected one Weston output with a full-width top panel");
    assert_eq!(current_frame_bounds(&graph), work_area);
}

pub(super) fn minimize_and_switch_back(world: &mut ApplicationWorld) {
    use super::super::x11_input::window_targets::switch_fullscreen_wayland_task;

    require_nested_input();
    let visible = compositor_frame_layer().expect("expected a visible Weston layer");
    click_frame_control(world, 0);
    wait_for_frame_layer(world, |layer| layer != Some(visible.as_str()));
    switch_fullscreen_wayland_task();
    wait_for_frame_layer(world, |layer| layer == Some(visible.as_str()));
    world.pump(Vec::new());
}

pub(in super::super) fn click_frame_control(world: &mut ApplicationWorld, index: usize) {
    use super::super::x11_input::window_targets::{
        fullscreen_left_button, move_fullscreen_pointer,
    };

    require_nested_input();
    let [x, y, _, _] = compositor_frame_bounds().unwrap();
    let [control_x, control_y] = frame_control_center(index);
    let target = [x + control_x, y + control_y];
    move_fullscreen_pointer(target);
    world.pump(Vec::new());
    fullscreen_left_button(target, true);
    world.pump(Vec::new());
    fullscreen_left_button(target, false);
}

fn wait_for_frame_layer(world: &mut ApplicationWorld, ready: impl Fn(Option<&str>) -> bool) {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(5);
    loop {
        pump_frame_controls(world);
        let layer = compositor_frame_layer();
        if ready(layer.as_deref()) {
            break;
        }
        assert!(
            std::time::Instant::now() < deadline,
            "unexpected Weston frame layer {layer:?}"
        );
        std::thread::sleep(std::time::Duration::from_millis(30));
    }
}

fn compositor_frame_layer() -> Option<String> {
    let graph = compositor_scene_graph().expect("set OBK_BDD_WESTON_DEBUG for Wayland layers");
    let pid = format!(", PID {}, ", std::process::id());
    let mut layer = None;
    for line in graph.lines() {
        if line.starts_with("Layer ") {
            layer = line
                .split_once("(pos ")
                .and_then(|(_, tail)| tail.strip_suffix("):"));
        }
        if is_application_toplevel(line, &pid) {
            return layer.map(str::to_owned);
        }
    }
    None
}

pub(super) fn drag_application_bar(world: &mut ApplicationWorld) {
    require_nested_input();
    let graph = compositor_scene_graph().expect("set OBK_BDD_WESTON_DEBUG for Wayland bounds");
    let [x, y, width, _] = current_frame_bounds(&graph);
    let [area_x, _, area_width, _] = scene_work_area(&graph).expect("expected Weston work area");
    let direction = if area_x + area_width - x - width >= 72 {
        1
    } else {
        -1
    };
    let start = [x + width - 390, y + 18];
    let motion = [20, 72].map(|distance| [start[0] + direction * distance, start[1]]);
    drag_fullscreen_pointer(world, start, &motion);
}

fn require_nested_input() {
    assert_eq!(
        std::env::var("OBK_BDD_WESTON_NESTED_X11").as_deref(),
        Ok("1"),
        "set OBK_BDD_WESTON_NESTED_X11=1 for fullscreen Weston on a private X11 display"
    );
}

pub(in super::super) fn hover_resize_handle(point: [i32; 2]) {
    require_nested_input();
    super::super::x11_input::window_targets::move_fullscreen_pointer(point);
}

pub(in super::super) fn drag_resize_handle(
    world: &mut ApplicationWorld,
    start: [i32; 2],
    end: [i32; 2],
) {
    require_nested_input();
    drag_fullscreen_pointer(world, start, &[end]);
}

pub(in super::super) fn expect_resize_cursor([x, y]: [i32; 2]) {
    assert_eq!(
        std::env::var("XCURSOR_THEME").as_deref(),
        Ok("oblikovati-frame"),
        "expected the private resize test cursor theme"
    );
    let graph = compositor_scene_graph().expect("set OBK_BDD_WESTON_DEBUG for the cursor");
    let pid = format!(", PID {}, ", std::process::id());
    let mut lines = graph.lines();
    lines
        .find(|line| line.contains("role wl_pointer-cursor") && line.contains(&pid))
        .expect("expected a compositor cursor surface from Oblikovati");
    let bounds = scene_bounds(lines.next().unwrap()).expect("expected Weston cursor bounds");
    // The private frame theme assigns a distinct buffer width to each resize icon.
    assert_eq!(
        bounds[2],
        24 + (x + 1) * 3 + y + 1,
        "cursor for axes [{x}, {y}]"
    );
}

fn drag_fullscreen_pointer(world: &mut ApplicationWorld, start: [i32; 2], motion: &[[i32; 2]]) {
    use super::super::x11_input::window_targets::{
        fullscreen_left_button, move_fullscreen_pointer,
    };

    move_fullscreen_pointer(start);
    world.pump(Vec::new());
    fullscreen_left_button(start, true);
    world.pump(Vec::new());
    for &point in motion {
        move_fullscreen_pointer(point);
        world.pump(Vec::new());
    }
    fullscreen_left_button(*motion.last().unwrap(), false);
    world.pump(Vec::new());
}

fn current_frame_bounds(graph: &str) -> [i32; 4] {
    let pid = std::process::id();
    scene_frame_bounds(graph, pid).unwrap_or_else(|| {
        panic!("Weston scene graph: expected bounds for Oblikovati process {pid}")
    })
}

fn compositor_scene_graph() -> Option<String> {
    let program = std::env::var_os("OBK_BDD_WESTON_DEBUG")?;
    let output = std::process::Command::new(&program)
        .arg("scene-graph")
        .output()
        .unwrap_or_else(|error| panic!("{program:?}: expected a Weston scene graph: {error}"));
    assert!(
        output.status.success(),
        "Weston scene graph: {}",
        String::from_utf8_lossy(&output.stderr)
    );
    Some(String::from_utf8(output.stdout).expect("expected UTF-8 Weston scene graph"))
}

fn scene_frame_bounds(graph: &str, pid: u32) -> Option<[i32; 4]> {
    let pid = format!(", PID {pid}, ");
    let mut lines = graph.lines();
    lines.find(|line| is_application_toplevel(line, &pid))?;
    scene_bounds(lines.next()?)
}

fn is_application_toplevel(line: &str, pid: &str) -> bool {
    line.contains("(role xdg_toplevel")
        && line.contains(pid)
        && line.contains(" of org.oblikovati.Oblikovati,")
}

fn scene_work_area(graph: &str) -> Option<[i32; 4]> {
    let mut lines = graph.lines();
    let header = lines.find(|line| line.starts_with("Output 0 ("))?;
    let output = header.strip_prefix("Output 0 (")?.strip_suffix("):")?;
    let [x, y, width, height] = scene_bounds(lines.next()?)?;
    let mut panels = graph.lines();
    let panel_description = format!(", panel for output {output}, ");
    panels.find(|line| line.contains(&panel_description))?;
    let [panel_x, panel_y, panel_width, panel_height] = scene_bounds(panels.next()?)?;
    (panel_x == x && panel_y == y && panel_width == width && panel_height < height).then_some([
        x,
        y + panel_height,
        width,
        height - panel_height,
    ])
}

fn scene_bounds(line: &str) -> Option<[i32; 4]> {
    let bounds = line.trim().strip_prefix("position: ")?;
    let (start, end) = bounds.split_once(" -> ")?;
    let [x, y] = scene_point(start)?;
    let [right, bottom] = scene_point(end)?;
    let (width, height) = (right.checked_sub(x)?, bottom.checked_sub(y)?);
    (width > 0 && height > 0).then_some([x, y, width, height])
}

fn scene_point(point: &str) -> Option<[i32; 2]> {
    let (x, y) = point
        .strip_prefix('(')?
        .strip_suffix(')')?
        .split_once(',')?;
    Some([x.trim().parse().ok()?, y.trim().parse().ok()?])
}

#[test]
fn scene_graph_bounds_select_the_application_toplevel_by_pid() {
    let graph = "\
View 0 (role (null), PID 42, surface ID 4, panel for output headless, 0x0):
    position: (0, 0) -> (1920, 32)
View 0 (role xdg_toplevel, PID 42, surface ID 8, top-level window 'Part' of org.oblikovati.Oblikovati, 0x1):
    position: (-3, 32) -> (1437, 932)
View 0 (role xdg_toplevel, PID 43, surface ID 7, top-level window 'Home' of org.oblikovati.Oblikovati, 0x2):
    position: (7, 12) -> (1447, 912)";
    assert_eq!(scene_frame_bounds(graph, 42), Some([-3, 32, 1440, 900]));
    assert_eq!(scene_frame_bounds(graph, 41), None);
}

#[test]
fn scene_work_area_excludes_the_panel_in_logical_coordinates() {
    let graph = "\
Output 0 (headless):
    position: (0, 0) -> (1920, 1080)
    mode: 3840x2160@60.000Hz
    scale: 2
View 0 (role (null), PID 42, surface ID 4, panel for output headless, 0x0):
    position: (0, 0) -> (1920, 32)";
    assert_eq!(scene_work_area(graph), Some([0, 32, 1920, 1048]));
    assert_eq!(scene_work_area(&graph.replace("1920, 32", "960, 32")), None);
}

#[test]
#[ignore = "requires a Wayland desktop compositor with room to maximize"]
fn wayland_frame_maximizes_and_restores_the_saved_part() {
    let mut world = ApplicationWorld::new();
    assert!(world.app().window.native.view.uses_wayland());
    let document = open_frame_part(&mut world);
    let restored = world.extent();
    maximize_tooltip(&mut world);
    press_maximize(&mut world);
    let maximized = expect_maximized(&mut world, restored);
    expect_frame_layout(&mut world, maximized, document);
    press_maximize(&mut world);
    expect_restored(&mut world, restored);
    expect_frame_layout(&mut world, restored, document);
    world.close();
}

#[test]
#[ignore = "requires a Wayland desktop compositor with room to maximize"]
fn wayland_double_click_maximizes_and_restores_the_application_bar() {
    let mut world = ApplicationWorld::new();
    assert!(world.app().window.native.view.uses_wayland());
    let restored = world.extent();
    double_click_bar(&mut world);
    expect_maximized(&mut world, restored);
    double_click_bar(&mut world);
    expect_restored(&mut world, restored);
    world.close();
}

fn expect_maximized(world: &mut ApplicationWorld, restored: [u32; 2]) -> [u32; 2] {
    pump_until(world, true);
    let maximized = world.extent();
    assert!(
        (0..2).all(|axis| maximized[axis] > restored[axis]),
        "expected growth from {restored:?} to {maximized:?}"
    );
    restore_tooltip(world);
    maximized
}

fn expect_restored(world: &mut ApplicationWorld, restored: [u32; 2]) {
    pump_until(world, false);
    assert_eq!(world.extent(), restored);
    maximize_tooltip(world);
}

pub(in super::super) fn expect_frame_layout(
    world: &mut ApplicationWorld,
    extent: [u32; 2],
    document: u64,
) {
    let [left, top, right, bottom] = world.viewport_bounds();
    assert!(left < right && right <= extent[0] && top < bottom && bottom <= extent[1]);
    let [left, top, right, bottom] =
        diagnostics::window_pixel_bounds("##application-ribbon").unwrap();
    assert!(0. <= left && right <= extent[0] as f32 && 0. <= top && bottom <= extent[1] as f32);
    controls_remain_in_bar(world);
    world.expect_active(document);
    workspace_layout::browser_feature::expect_solid_visible(world);
}
