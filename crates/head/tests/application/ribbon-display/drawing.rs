// SPDX-License-Identifier: GPL-2.0-only
//! Ribbon display scenarios for drawing documents.
use super::*;

const DRAWING_GROUPS: [&str; 4] = ["Create", "Modify", "Sketch", "Sheets"];
#[given("the Place Views tab shows its full command groups")]
fn place_views_full(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.ribbon_height = band_height();
}
fn double_click_place_views(world: &mut ApplicationWorld) {
    let center = world.run("Place Views").center().map(|value| value as i32);
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        center, 2,
    ));
    world.pump(Vec::new());
}
#[when("I double-click the active Place Views tab")]
#[when("the user double-clicks the active Place Views tab")]
#[when("I double-click the active Place Views tab again")]
#[when("the user double-clicks the active Place Views tab again")]
fn double_click(world: &mut ApplicationWorld) {
    double_click_place_views(world);
}
fn expect_compact_drawing(world: &mut ApplicationWorld) {
    for group in DRAWING_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text("Base View");
}
#[then("the ribbon shows compact Create, Modify, Sketch and Sheets group buttons")]
#[then("its ribbon displays compact Create, Modify, Sketch and Sheets group buttons")]
fn compact_drawing(world: &mut ApplicationWorld) {
    expect_compact_drawing(world);
}
#[then("the ribbon shows the group titles in a narrow strip")]
fn titles_strip(world: &mut ApplicationWorld) {
    for group in DRAWING_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text("Base View");
    assert!(
        band_height() < world.ribbon_height,
        "expected a narrower ribbon"
    );
}
#[then("the ribbon shows tabs without the group strip")]
fn tabs_without_strip(world: &mut ApplicationWorld) {
    world.expect_text("Place Views");
    for group in ["Modify", "Sheets"] {
        world.expect_no_text(group);
    }
}
#[then("the full Place Views command groups are visible")]
fn full_groups(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.expect_text("New Sheet");
}
#[then("the drawing document remains active")]
pub(super) fn drawing_remains(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.document, world.previous_document);
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_DRAWING)
    );
}
#[given("a drawing and a part are open in the MCAD Software")]
fn drawing_and_part(world: &mut ApplicationWorld) {
    world.other_document = Some(world.add_part());
    world.previous_document = Some(world.add_drawing());
}
#[given("the drawing is active with its ribbon in compact group-button mode")]
#[given("the Place Views ribbon shows compact group buttons")]
fn compact_mode(world: &mut ApplicationWorld) {
    click_display_menu(world);
    world.click_text("Minimize to Panel Buttons");
    expect_compact_drawing(world);
}
#[when("I activate the part document")]
#[when("the user activates the part document")]
fn activate_part(world: &mut ApplicationWorld) {
    let part = world.other_document.expect("expected a part");
    world.click_tab(part);
}
#[then("the part ribbon displays its command groups as compact buttons")]
fn part_compact(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    world.expect_no_text("Extrude");
}
#[then("those buttons include Sketch, Create and Modify")]
fn part_groups(world: &mut ApplicationWorld) {
    for group in ["Sketch", "Create", "Modify"] {
        world.expect_text(group);
    }
}
#[when("I return to the drawing document")]
#[when("the user returns to the drawing document")]
fn back_to_drawing(world: &mut ApplicationWorld) {
    let drawing = world.previous_document.expect("expected a drawing");
    world.click_tab(drawing);
}
#[when("I select the Create group button")]
#[when("the user selects the Create group button")]
fn select_create(world: &mut ApplicationWorld) {
    world.click_text("Create");
    world.pump(Vec::new());
}
#[then("a temporary Create panel displays drawing-view commands")]
fn create_panel(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
}
#[when("I select the Modify group button")]
fn select_modify(world: &mut ApplicationWorld) {
    world.click_text("Modify");
    world.pump(Vec::new());
}
#[then("the Create panel closes")]
fn create_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Base View");
    world.expect_no_text("Extrude");
}
#[then("a temporary Modify panel displays modification commands")]
fn modify_panel(world: &mut ApplicationWorld) {
    world.expect_text("Break");
}
#[when("I click an empty area of the drawing outside the panel")]
#[when("the user clicks an empty area of the drawing outside the panel")]
fn click_drawing(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
}
#[then("the temporary Modify panel closes")]
fn modify_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Break");
    if world.state().document_type == Some(oblikovati_api::types::DOCUMENT_PART) {
        assert!(world.run("Hole").minimum[1] < band_height());
    }
}
#[then("the compact group buttons remain visible")]
fn compact_remains(world: &mut ApplicationWorld) {
    expect_compact_drawing(world);
}
#[given("a drawing is active in the MCAD Software")]
fn a_drawing_active(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_drawing());
}
#[given("the temporary Create group panel is open")]
fn create_group_open(world: &mut ApplicationWorld) {
    select_create(world);
    world.expect_text("Base View");
}
