// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::workspace_layout::browser_feature::{SKETCH, right_click_row};
use super::*;

#[given("the full Sketch ribbon displays the Pattern panel and the Finish action")]
#[given("the full Sketch ribbon displays the Pattern panel")]
fn full_pattern_fixture(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.active_id());
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    world.record_initial_sketch();
    pattern_shown(world);
    world.observed_view = Some(finish_position(world));
}

fn pattern_shown(world: &mut ApplicationWorld) {
    assert!(ribbon_runs(world, "Pattern").next().is_some());
    world.expect_text("Rectangular");
}

fn finish_position(world: &ApplicationWorld) -> [i32; 2] {
    diagnostics::pointer_pixels(world.run("Finish Sketch").center())
}

#[when("the user right-clicks the Sketch tab")]
fn sketch_context(world: &mut ApplicationWorld) {
    let tab = ribbon_runs(world, "Sketch")
        .next()
        .expect("expected Sketch tab");
    world.right_click(diagnostics::pointer_pixels(tab.center()));
    world.expect_text("Show Panel Titles");
}

#[then("Pattern is checked in the panel list")]
fn pattern_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Pattern"));
}

#[when("the user selects Pattern")]
fn select_pattern(world: &mut ApplicationWorld) {
    assert!(side_selector::popup_depth() >= 2, "expected Show Panels");
    toggle_panel(world, "Pattern");
}

#[then("the menus close and the Pattern panel disappears from the ribbon")]
#[then("the Sketch ribbon is visible without the Pattern panel")]
fn pattern_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Show Panel Titles");
    assert!(ribbon_runs(world, "Pattern").next().is_none());
    world.expect_no_text("Rectangular");
    world.expect_text("Finish Sketch");
}

#[then("the remaining ribbon panels reflow and Finish remains visible")]
fn sketch_reflows(world: &mut ApplicationWorld) {
    let before = world
        .observed_view
        .expect("expected the prior Finish position");
    assert!(finish_position(world)[0] < before[0]);
}

#[when("the user reopens Show Panels from the Sketch tab context menu")]
#[when("the user opens Show Panels from the Sketch tab context menu")]
fn sketch_show_panels(world: &mut ApplicationWorld) {
    open_show_panels_at(world, "Sketch");
}

#[then("Pattern is unchecked")]
fn pattern_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Pattern"));
}

#[then("the Pattern panel returns to the Sketch ribbon")]
fn pattern_returns(world: &mut ApplicationWorld) {
    pattern_shown(world);
    assert_eq!(Some(finish_position(world)), world.observed_view);
}

#[then("Finish remains visible")]
fn finish_visible(world: &mut ApplicationWorld) {
    world.expect_text("Finish Sketch");
}

#[when("the user hides Pattern using Show Panels in the Sketch tab context menu")]
fn hide_pattern(world: &mut ApplicationWorld) {
    sketch_show_panels(world);
    pattern_checked(world);
    select_pattern(world);
}

#[when("the user finishes the sketch without changing geometry")]
fn finish_without_editing(world: &mut ApplicationWorld) {
    sketch_unchanged(world);
    world.finish_sketch();
}

#[when("the user reopens the same sketch from the model browser")]
fn reopen_same_sketch(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    assert_eq!(Some(world.active_id()), world.previous_document);
    world.expect_text("Finish Sketch");
    sketch_unchanged(world);
}

fn sketch_unchanged(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert!(geometry.same_recipe(&world.observed_sketch.as_ref().unwrap().geometry));
}
