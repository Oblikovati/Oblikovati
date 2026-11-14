// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[given("the part has an active sketch editing environment")]
fn part_in_sketch(world: &mut ApplicationWorld) {
    // A mixed fixture names its part; on its own the scenario's part is the active document.
    let part = match world.other_document {
        Some(part) => part,
        None => world.active_id(),
    };
    world.activate_through_tab(part);
    world.start_sketch();
}
#[given("the drawing is active with its drawing resources branch collapsed")]
fn drawing_active_collapsed(world: &mut ApplicationWorld) {
    world.activate_through_tab(drawing_of(world));
    world.press_disclosure("Drawing Resources");
    world.expect_no_text("Sheet Formats");
}
#[when("I select the part document tab")]
#[when("I select the part document tab again")]
fn select_part_tab(world: &mut ApplicationWorld) {
    world.click_tab(part_of(world));
}
#[then("the part sketch viewport and model browser are displayed")]
fn part_sketch_shown(world: &mut ApplicationWorld) {
    world.expect_active(part_of(world));
    let centre = world.pixel_at([700, 450]);
    assert!(
        centre[..3].iter().all(|channel| *channel < 150),
        "viewport {centre:?}: expected no sheet"
    );
}
#[then("the sketch ribbon is active")]
fn sketch_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Finish Sketch");
}
#[when("I select the drawing document tab")]
fn select_drawing_tab(world: &mut ApplicationWorld) {
    world.click_tab(drawing_of(world));
}
pub(super) fn expect_drawing_shown(world: &mut ApplicationWorld) {
    let sheet = world.pixel_at([1000, 500]);
    assert!(
        sheet[..3].iter().all(|channel| *channel > 200),
        "sheet {sheet:?}"
    );
    world.expect_text("Drawing Resources");
}
#[then("the drawing sheet and drawing browser are displayed")]
#[then("its sheet and drawing browser are displayed")]
fn drawing_shown(world: &mut ApplicationWorld) {
    expect_drawing_shown(world);
}
#[then("the drawing ribbon is active")]
fn drawing_ribbon_active(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.expect_no_text("Finish Sketch");
}
#[then("the part remains available as an open tab")]
fn part_tab_remains(world: &mut ApplicationWorld) {
    let name = world.document_name(part_of(world));
    world.expect_tab(&name);
}
#[given("the part is active")]
fn part_active(world: &mut ApplicationWorld) {
    world.activate_through_tab(part_of(world));
}
#[when("I open the document list at the end of the tab strip")]
fn open_document_list(world: &mut ApplicationWorld) {
    world.click_text("Documents");
    world.pump(Vec::new());
}
#[then("the list contains the open part and drawing")]
fn list_contains(world: &mut ApplicationWorld) {
    for id in [part_of(world), drawing_of(world)] {
        let name = world.document_name(id);
        world.expect_text(&name);
    }
}
#[then("the active part is marked in the list")]
fn active_marked(world: &mut ApplicationWorld) {
    let name = world.document_name(part_of(world));
    assert!(world.checked(&name), "list row {name:?}: expected a mark");
}
#[when("I select the drawing from the list")]
fn select_drawing_row(world: &mut ApplicationWorld) {
    let name = world.document_name(drawing_of(world));
    world.click_text(&name);
}
#[then("the drawing becomes active")]
fn drawing_is_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, Some(drawing_of(world)));
}
