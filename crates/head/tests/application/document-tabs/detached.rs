// SPDX-License-Identifier: GPL-2.0-only
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
    let from = world.expect_tab(&name).center().map(|value| value as i32);
    let target = [800, 500];
    #[cfg(target_os = "linux")]
    drag_tab_with_x11(world, from, target);
    #[cfg(not(target_os = "linux"))]
    world.drag(from, target);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
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
    let at = sheet_point(world).map(|coordinate| coordinate as usize);
    let sheet = world.pixel_at(at);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "pixel {at:?} {sheet:?}"
    );
}
#[then("the part remains open in the main workspace")]
fn part_in_main(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I select another open part tab in the main workspace")]
#[when("I select the part tab in the main workspace")]
#[when("the user selects another open part tab in the main workspace")]
#[when("the user selects the part tab in the main workspace")]
fn select_main_part_tab(world: &mut ApplicationWorld) {
    world.click_tab(part_of(world));
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
    let labels = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == name)
        .count();
    assert!(
        labels >= 2,
        "expected the window title and its tab to read {name:?}"
    );
    detached_tab_position(world);
}
fn detached_window_geometry(world: &mut ApplicationWorld) -> ([f32; 2], [f32; 2]) {
    let drawing = drawing_of(world);
    let name = world.document_name(drawing);
    window_geometry(&format!("{name}###detached-document-{drawing}"))
        .expect("expected detached drawing geometry")
}
fn detached_tab_position(world: &mut ApplicationWorld) -> [i32; 2] {
    let (position, size) = detached_window_geometry(world);
    let name = world.document_name(drawing_of(world));
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let [x, y] = run.center();
            run.text == name
                && x >= position[0]
                && x < position[0] + size[0]
                && y >= position[1] + size[1] * 0.8
                && y < position[1] + size[1]
        })
        .expect("expected the drawing tab below the sheet");
    super::super::diagnostics::pointer_pixels(run.center())
}
fn sheet_point(world: &mut ApplicationWorld) -> [i32; 2] {
    let (position, size) = detached_window_geometry(world);
    super::super::diagnostics::pointer_pixels(std::array::from_fn(|axis| {
        position[axis] + size[axis] / 2.
    }))
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
    let center = detached_tab_position(world);
    #[cfg(target_os = "linux")]
    super::super::x11_input::right_click(center);
    #[cfg(not(target_os = "linux"))]
    world.right_click(center);
    world.pump(Vec::new());
}
#[given("a part is open in the main workspace")]
#[given("the part context is active in the main workspace")]
fn part_context(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I click the detached drawing sheet")]
#[when("the user clicks the detached drawing sheet")]
fn click_detached_sheet(world: &mut ApplicationWorld) {
    let point = sheet_point(world);
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(point);
    #[cfg(not(target_os = "linux"))]
    world.click(point);
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
    world.save_capture("detached-drawing-with-main-strip-hidden");
    world.close();
}
