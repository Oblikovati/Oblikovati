// SPDX-License-Identifier: GPL-2.0-only
use super::super::{diagnostics::pointer_pixels, x11_input};
use super::mixed_documents::expect_drawing_shown;
use super::*;
const NEIGHBOR_PARTS: usize = 3;
fn open_parts_then_drawing(world: &mut ApplicationWorld) {
    for _ in 0..NEIGHBOR_PARTS {
        world.add_part();
    }
    world.previous_document = Some(world.add_drawing());
}
pub(super) fn tab_x(world: &mut ApplicationWorld, id: u64) -> f32 {
    let name = world.document_name(id);
    world.expect_tab(&name).center()[0]
}
fn drag_tab_to_first_place(world: &mut ApplicationWorld, id: u64) {
    world.dragged_tab_x = Some(tab_x(world, id));
    let name = world.document_name(id);
    let from = world.expect_tab(&name).center();
    let first = tab_x(world, world.parts[0]);
    drag_reorder_in_strip(world, from, first - 30.);
}

fn drag_reorder_in_strip(world: &mut ApplicationWorld, from: [f32; 2], target_x: f32) {
    let start = pointer_pixels(from);
    x11_input::move_pointer(start);
    world.pump(Vec::new());
    x11_input::left_button(start, true);
    world.pump(Vec::new());
    for step in 1..=4 {
        let x = from[0] + (target_x - from[0]) * (step as f32 / 4.);
        x11_input::move_pointer(pointer_pixels([x, from[1]]));
        world.pump(Vec::new());
    }
    x11_input::left_button(pointer_pixels([target_x, from[1]]), false);
    world.pump(Vec::new());
}
fn expect_tab_moved_left(world: &mut ApplicationWorld, id: u64) {
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, id);
    assert!(
        now < before - 100.,
        "tab at {now}: expected well left of {before}"
    );
    let first_part = tab_x(world, world.parts[0]);
    assert!(
        now < first_part,
        "tab at {now}: expected before the first part at {first_part}"
    );
}
#[given("a drawing and several other documents are open in the MCAD Software")]
#[given("a drawing and several parts are open in the MCAD Software")]
fn parts_and_drawing(world: &mut ApplicationWorld) {
    // Reordering across visible neighbors needs a saved workspace with room
    // for all four tabs. The fresh profile also docks two asset tools (#3900).
    world.restore_layout_without_asset_tools();
    open_parts_then_drawing(world);
    for id in world.parts.clone() {
        tab_x(world, id);
    }
}
#[given("the drawing is active and its tab is visible")]
#[given("the drawing is active")]
fn drawing_tab_visible(world: &mut ApplicationWorld) {
    let drawing = drawing_of(world);
    assert_eq!(world.state().document, Some(drawing));
    tab_x(world, drawing);
}
#[when("I drag the drawing tab left across several neighboring tabs")]
#[when("the user drags the drawing tab left across several neighboring tabs")]
fn drag_drawing_tab_left(world: &mut ApplicationWorld) {
    drag_tab_to_first_place(world, drawing_of(world));
}
#[then("the drawing tab appears at its new position among the document tabs")]
fn drawing_tab_moved(world: &mut ApplicationWorld) {
    expect_tab_moved_left(world, drawing_of(world));
}
#[then("its sheet and browser remain displayed")]
fn sheet_and_browser(world: &mut ApplicationWorld) {
    expect_drawing_shown(world);
}
#[when("I drag the drawing tab back to its earlier position")]
#[when("the user drags the drawing tab back to its earlier position")]
fn drag_drawing_tab_back(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    let from = world.expect_tab(&name).center();
    let target = world.dragged_tab_x.expect("expected a starting position");
    drag_reorder_in_strip(world, from, target + 30.);
}
#[then("the drawing tab returns to its earlier position")]
fn drawing_tab_back(world: &mut ApplicationWorld) {
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, drawing_of(world));
    assert!((now - before).abs() < 6., "tab at {now}: expected {before}");
}
#[given("an inactive part tab is visible")]
fn inactive_part_visible(world: &mut ApplicationWorld) {
    let part = world.parts[NEIGHBOR_PARTS - 1];
    tab_x(world, part);
}
#[when("I drag the inactive part tab left across several neighboring tabs")]
#[when("the user drags the inactive part tab left across several neighboring tabs")]
fn drag_part_tab_left(world: &mut ApplicationWorld) {
    drag_tab_to_first_place(world, world.parts[NEIGHBOR_PARTS - 1]);
}
#[then("the part tab appears at its new position among the document tabs")]
fn part_tab_moved(world: &mut ApplicationWorld) {
    let part = world.parts[NEIGHBOR_PARTS - 1];
    let before = world.dragged_tab_x.expect("expected a starting position");
    let now = tab_x(world, part);
    assert!(
        now < before - 60.,
        "tab at {now}: expected well left of {before}"
    );
    let first_part = tab_x(world, world.parts[0]);
    assert!(
        now < first_part,
        "tab at {now}: expected before the first part at {first_part}"
    );
}
#[then("the part viewport and model browser are displayed")]
fn dragged_part_shown(world: &mut ApplicationWorld) {
    let part = world.parts[NEIGHBOR_PARTS - 1];
    world.expect_active(part);
    world.expect_browser_of(part);
    world.expect_blank_region(
        super::super::documents::VIEWPORT_X,
        super::super::documents::VIEWPORT_Y,
    );
}
#[then("the drawing remains open")]
fn drawing_open(world: &mut ApplicationWorld) {
    assert!(world.document_is_open(drawing_of(world)));
}
