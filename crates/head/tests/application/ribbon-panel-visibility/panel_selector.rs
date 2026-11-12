// SPDX-License-Identifier: GPL-2.0-only
//! Read the direct selector and ribbon captions from completed native draw packets.
use super::super::diagnostics::window_geometry;
use super::*;
fn ribbon_bottom() -> f32 {
    let (position, size) = window_geometry("##application-ribbon").unwrap();
    position[1] + size[1]
}
fn selector_is_open(world: &ApplicationWorld) -> bool {
    let row = world.run(PARAMETERS).center();
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    row[0] > size[0] * 0.5 && row[1] > ribbon_bottom()
}
fn caption_is_visible(world: &ApplicationWorld) -> bool {
    world
        .frame
        .runs
        .iter()
        .any(|row| row.text == PARAMETERS && row.center()[1] < ribbon_bottom())
}
#[when("the user clicks the panel selector at the right end of the ribbon")]
#[when("the user clicks the ribbon panel selector")]
#[when("the user clicks the ribbon panel selector again")]
fn open_selector(world: &mut ApplicationWorld) {
    let (_, size) = window_geometry("##application-ribbon").unwrap();
    world.click([size[0] as i32 - 22, 45]);
    world.pump(Vec::new());
}
#[given("the full 3D Model ribbon shows the Modify panel")]
#[then("the Modify panel remains visible")]
fn modify_panel_visible(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
}
#[then("the panel list opens with Modify checked")]
fn list_modify_checked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(world.checked("Modify"), "expected Modify checked");
}
#[then("the panel list opens with Parameters unchecked")]
fn list_unchecked(world: &mut ApplicationWorld) {
    assert!(selector_is_open(world), "expected the direct panel list");
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
}
#[when("the user selects Parameters in the list")]
#[when("the user selects Parameters again")]
fn toggle_selector_parameter(world: &mut ApplicationWorld) {
    world.click_text(PARAMETERS);
    world.pump(Vec::new());
}
#[then("Parameters is checked and its panel appears in the ribbon")]
fn checked_and_visible(world: &mut ApplicationWorld) {
    assert!(world.checked(PARAMETERS), "expected Parameters checked");
    assert!(
        caption_is_visible(world),
        "expected the actual ribbon panel caption"
    );
}
#[then("Parameters is unchecked and its panel disappears")]
fn unchecked_and_hidden(world: &mut ApplicationWorld) {
    assert!(!world.checked(PARAMETERS), "expected Parameters unchecked");
    assert!(
        !caption_is_visible(world),
        "expected no ribbon panel caption"
    );
}
#[then("the panel list remains open")]
fn list_stays_open(world: &mut ApplicationWorld) {
    assert!(
        selector_is_open(world),
        "expected the panel list to stay open"
    );
}
#[when("the user presses Escape")]
fn escape_selector(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
}
#[then("the panel list closes")]
fn list_closes(world: &mut ApplicationWorld) {
    assert!(
        !selector_is_open(world),
        "expected the direct panel list closed"
    );
}
