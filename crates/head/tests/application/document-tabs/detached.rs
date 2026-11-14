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
fn drag_drawing_out(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let from = world.expect_tab(&name).center().map(|value| value as i32);
    world.drag(from, [800, 500]);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("the drawing is displayed in a separate window with its own document tab")]
fn drawing_detached(world: &mut ApplicationWorld) {
    expect_drawing_window(world);
}
#[then("the drawing sheet remains visible in that window")]
fn sheet_in_window(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let title = world.run(&name).center();
    let at = [title[0] as usize + 200, title[1] as usize + 200];
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
fn open_detached_menu(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let center = world.run(&name).center().map(|value| value as i32);
    world.right_click(center);
    world.pump(Vec::new());
}
#[given("a part is open in the main workspace")]
#[given("the part context is active in the main workspace")]
fn part_context(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
}
#[when("I click the detached drawing sheet")]
fn click_detached_sheet(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let title = world.run(&name).center();
    world.click([title[0] as i32 + 200, title[1] as i32 + 200]);
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
