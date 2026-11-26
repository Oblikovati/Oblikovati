// SPDX-License-Identifier: GPL-2.0-only
use super::super::main_window_frame::wayland;
use super::reordering::tab_x;
use super::*;
#[given("the drawing is active in the main document tab strip")]
fn drawing_in_main_strip(world: &mut ApplicationWorld) {
    let drawing = drawing_of(world);
    assert_eq!(world.state().document, Some(drawing));
    tab_x(world, drawing);
}
#[when("I drag the drawing tab from the strip into the workspace")]
#[when("the user drags the drawing tab from the strip into the workspace")]
fn drag_drawing_out(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let from = super::super::diagnostics::pointer_pixels(world.expect_tab(&name).center());
    let target = [800, 500];
    if world.app().window.native.view.uses_wayland() {
        let [x, y, _, _] = main_wayland_bounds(world);
        let start = [x + from[0], y + from[1]];
        let motion = [[start[0], y + target[1]], [x + target[0], y + target[1]]];
        wayland::drag_fullscreen_pointer(world, start, &motion);
    } else {
        drag_tab_with_x11(world, from, target);
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[given("the user drags its tab into the workspace to detach it into a separate window")]
#[given("the user drags the drawing tab into the workspace to detach it into a separate window")]
fn detach_saved_drawing(world: &mut ApplicationWorld) {
    let only_drawing = world.state().documents.len() == 1;
    world.activate_through_tab(drawing_of(world));
    drag_drawing_out(world);
    expect_drawing_window(world);
    if only_drawing {
        assert!(window_geometry("##home-workspace").is_some());
    }
}

fn drag_tab_with_x11(world: &mut ApplicationWorld, from: [i32; 2], target: [i32; 2]) {
    use super::super::x11_input::{left_button, move_pointer};
    move_pointer(from);
    world.pump(Vec::new());
    left_button(from, true);
    world.pump(Vec::new());
    move_pointer([from[0], target[1]]);
    world.pump(Vec::new());
    move_pointer(target);
    world.pump(Vec::new());
    left_button(target, false);
}
#[then("the drawing is displayed in a separate window with its own document tab")]
fn drawing_detached(world: &mut ApplicationWorld) {
    expect_drawing_window(world);
}
#[then("the drawing sheet remains visible in that window")]
fn sheet_in_window(world: &mut ApplicationWorld) {
    let capture = detached_capture(world);
    let [width, height] = capture.extent.map(|dimension| dimension as usize);
    let at = [width / 2, height / 2];
    let index = (at[1] * width + at[0]) * 4;
    let sheet = &capture.rgba[index..index + 4];
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "pixel {at:?} {sheet:?}"
    );
}
#[then("the part remains open in the main workspace")]
fn part_in_main(world: &mut ApplicationWorld) {
    let active = world.active_id();
    assert!(world.parts.contains(&active));
    world.expect_browser_of(active);
    let name = world.document_name(active);
    world.expect_tab(&name);
}
#[when("I select another open part tab in the main workspace")]
#[when("I select the part tab in the main workspace")]
#[when("the user selects another open part tab in the main workspace")]
#[when("the user selects the part tab in the main workspace")]
fn select_main_part_tab(world: &mut ApplicationWorld) {
    let name = world.document_name(part_of(world));
    let at = super::super::diagnostics::pointer_pixels(world.expect_tab(&name).center());
    if world.app().window.native.view.uses_wayland() {
        let [x, y, _, _] = main_wayland_bounds(world);
        wayland::click_compositor_point(world, [x + at[0], y + at[1]]);
    } else {
        super::super::x11_input::click(at);
    }
    world.pump(Vec::new());
}

fn main_wayland_bounds(world: &mut ApplicationWorld) -> [i32; 4] {
    let title = world.app().window.native.view.native_title().unwrap();
    wayland::compositor_toplevel_bounds(&title)
}
#[then("that part becomes active in the main workspace")]
fn that_part_active(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
fn expect_drawing_window(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    assert!(
        world.tab_run(&name).is_none(),
        "tab {name:?}: expected it out of the strip"
    );
    assert_native_drawing(world, &name);
    detached_tab_position(world);
}

fn assert_native_drawing(world: &mut ApplicationWorld, name: &str) {
    let id = drawing_of(world);
    let title = world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(id, |view| {
            view.native_title()
                .expect("expected an operating-system drawing title")
        })
        .expect("expected a drawing viewport");
    assert_eq!(title, name);
    if world.app().window.native.view.uses_wayland() {
        let main_title = world.app().window.native.view.native_title().unwrap();
        assert_ne!(name, main_title);
        wayland::compositor_toplevel_bounds(&main_title);
        wayland::compositor_toplevel_bounds(name);
    } else {
        assert!(super::super::x11_input::window_is_separate(name));
    }
}

fn detached_frame(world: &mut ApplicationWorld) -> super::super::diagnostics::RenderedFrame {
    let id = drawing_of(world);
    world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(id, |_| super::super::diagnostics::rendered_frame())
        .expect("expected a completed drawing GUI")
}

fn detached_capture(world: &mut ApplicationWorld) -> oblikovati_viewport::ViewportCapture {
    let id = drawing_of(world);
    world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(id, |view| {
            view.capture()
                .expect("expected a presented drawing surface")
        })
        .expect("expected a detached drawing viewport")
}

fn detached_tab_position(world: &mut ApplicationWorld) -> [i32; 2] {
    let frame = detached_frame(world);
    let height = detached_extent(world)[1] as f32;
    let name = world.document_name(drawing_of(world));
    let label = super::super::world::squeezed(&name);
    let run = frame
        .runs
        .iter()
        .find(|run| run.text == label && run.center()[1] > height * 0.8)
        .unwrap_or_else(|| panic!("expected drawing tab {name:?} below the sheet"));
    super::super::diagnostics::pointer_pixels(run.center())
}

fn detached_extent(world: &mut ApplicationWorld) -> [u32; 2] {
    let id = drawing_of(world);
    world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(id, |view| {
            view.capture().expect("expected drawing geometry").extent
        })
        .expect("expected a detached drawing viewport")
}

fn drawing_point_in_main_coordinates(world: &mut ApplicationWorld, point: [i32; 2]) -> [i32; 2] {
    let id = drawing_of(world);
    let drawing = world
        .app()
        .window
        .native
        .drawing_windows
        .inspect(id, |view| {
            view.window_screen_bounds()
                .expect("expected drawing screen bounds")
        })
        .expect("expected a detached drawing viewport");
    let main = super::super::x11_input::application_bounds();
    std::array::from_fn(|axis| point[axis] + drawing[axis] - main[axis])
}

fn drawing_point_on_weston(world: &mut ApplicationWorld, point: [i32; 2]) -> [i32; 2] {
    let name = world.document_name(drawing_of(world));
    let [x, y, _, _] = wayland::compositor_toplevel_bounds(&name);
    [x + point[0], y + point[1]]
}

#[then("the drawing remains open in its separate window")]
#[then("the detached drawing remains open")]
#[given("the drawing has a document tab in that window")]
fn drawing_window_remains(world: &mut ApplicationWorld) {
    expect_drawing_window(world);
}
#[given("a drawing is open in a separate window in the MCAD Software")]
fn drawing_in_window(world: &mut ApplicationWorld) {
    world.other_document = Some(world.add_part());
    world.previous_document = Some(world.add_drawing());
    drag_drawing_out(world);
}
#[when("I open the detached drawing tab context menu")]
#[when("the user opens the detached drawing tab context menu")]
fn open_detached_menu(world: &mut ApplicationWorld) {
    let point = detached_tab_position(world);
    if world.app().window.native.view.uses_wayland() {
        let at = drawing_point_on_weston(world, point);
        wayland::right_click_compositor_point(world, at);
    } else {
        super::super::x11_input::right_click(drawing_point_in_main_coordinates(world, point));
    }
    world.pump(Vec::new());
    world.frame = detached_frame(world);
    world.detached_drawing_menu = true;
}

pub(super) fn dismiss_detached_menu(world: &mut ApplicationWorld) {
    if world.app().window.native.view.uses_wayland() {
        super::super::x11_input::window_targets::press_weston_key(0xff1b);
    } else {
        let name = world.document_name(drawing_of(world));
        super::super::x11_input::press_window_key(&name, 0xff1b);
    }
    world.pump(Vec::new());
    let auxiliary_frame = detached_frame(world);
    let main_frame = std::mem::replace(&mut world.frame, auxiliary_frame);
    world.expect_no_text("Close Tabs");
    world.frame = main_frame;
}

#[given("a part is open in the main workspace")]
#[given("the part context is active in the main workspace")]
fn part_context(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I click the detached drawing sheet")]
#[when("the user clicks the detached drawing sheet")]
fn click_detached_sheet(world: &mut ApplicationWorld) {
    let point = detached_extent(world).map(|dimension| dimension as i32 / 2);
    if world.app().window.native.view.uses_wayland() {
        let at = drawing_point_on_weston(world, point);
        wayland::click_compositor_point(world, at);
    } else {
        super::super::x11_input::click(drawing_point_in_main_coordinates(world, point));
    }
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the main application browser displays the drawing tree")]
fn main_drawing_tree(world: &mut ApplicationWorld) {
    world.expect_text("Drawing Resources");
    world.expect_no_text("Origin");
}
#[then("the main application ribbon displays drawing-context tabs")]
fn main_drawing_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Place Views");
}
#[then("the main application browser displays the part tree")]
fn main_part_tree(world: &mut ApplicationWorld) {
    world.expect_text("Origin");
    world.expect_no_text("Drawing Resources");
}
#[then("the main application ribbon displays part-context tabs")]
fn main_part_tabs(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
}

#[test]
fn detached_drawing_keeps_its_bottom_tab_when_the_main_strip_is_hidden() {
    let mut world = ApplicationWorld::new();
    drawing_in_window(&mut world);
    expect_drawing_window(&mut world);
    toggle_tabs_option(&mut world);
    assert!(window_geometry(TAB_STRIP).is_none());
    expect_drawing_window(&mut world);
    sheet_in_window(&mut world);
    let path = super::super::world::reports_folder().join("detached-drawing-native-sheet.png");
    if let Some(folder) = path.parent() {
        std::fs::create_dir_all(folder).unwrap();
    }
    super::super::world::write_png(&path, &detached_capture(&mut world)).unwrap();
    world.close();
}

#[test]
fn closing_the_native_drawing_window_restores_the_same_open_tab() {
    let mut world = ApplicationWorld::new();
    drawing_in_window(&mut world);
    expect_drawing_window(&mut world);
    let id = drawing_of(&world);
    let name = world.document_name(id);
    super::super::x11_input::request_named_window_close(&name);
    for _ in 0..4 {
        world.pump(Vec::new());
    }
    assert!(
        world
            .app()
            .window
            .native
            .drawing_windows
            .inspect(id, |_| ())
            .is_none()
    );
    world.expect_tab(&name);
    world.expect_active(part_of(&world));
    world.close();
}
