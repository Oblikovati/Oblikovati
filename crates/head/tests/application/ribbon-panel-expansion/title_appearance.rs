// SPDX-License-Identifier: GPL-2.0-only
//! Appearance selection from an expanded panel title.
use super::{ApplicationWorld, WINDOW, diagnostics};
use cucumber::{given, then, when};

#[given("the ribbon appearance is \"Normal\"")]
fn normal_appearance(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
    assert!(world.checked("Normal"));
    world.escape();
    world.escape();
}
#[when("the user right-clicks the lower Modify title and opens \"Ribbon Appearance\"")]
fn open_appearance(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run_in_window(WINDOW, "Modify").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
}
#[then("\"Normal\" is checked and the Modify additional commands remain visible")]
fn normal_checked_with_expansion(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"));
    assert!(diagnostics::window_geometry(WINDOW).is_some());
    world.expect_text("Move Bodies");
}
#[when("the user selects the checked \"Normal\" option")]
fn select_normal(world: &mut ApplicationWorld) {
    assert!(world.checked("Normal"));
    world.click_text("Normal");
}
#[then("the full ribbon remains visible with command labels")]
fn full_ribbon_with_labels(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
    world.expect_text("Hole");
}
