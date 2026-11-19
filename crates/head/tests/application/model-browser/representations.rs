// SPDX-License-Identifier: GPL-2.0-only
//! Inspect representation branches with native pointer input (#4118).
use super::part_views::{expanded_primary_view, view_branch};
use super::*;
use cucumber::{given, then, when};

#[given("the model browser shows an expanded Representations branch")]
fn representations_are_expanded(world: &mut ApplicationWorld) {
    world.expect_text("Representations");
    world.expect_text("Model State");
    view_branch(world);
}
#[given("the Model State and View branches are collapsed")]
fn representation_children_are_collapsed(world: &mut ApplicationWorld) {
    world.expect_no_text("Primary (Active)");
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| row.text.trim() == "Primary" && row.list == view_branch(world).list)
    );
}
#[when("the user expands the Model State branch")]
fn expand_model_state_branch(world: &mut ApplicationWorld) {
    world.click_text("Model State");
}
#[then("a checked Primary entry appears beneath it")]
fn checked_primary_model_state(world: &mut ApplicationWorld) {
    world.expect_text("Primary (Active)");
    assert!(world.checked("Primary (Active)"));
}
#[when("the user expands the View branch")]
fn expand_view_branch(world: &mut ApplicationWorld) {
    expanded_primary_view(world);
}
#[then("Primary, Isometric, Front, Top and Right entries appear beneath it")]
fn all_builtin_view_rows(world: &mut ApplicationWorld) {
    let branch = view_branch(world);
    for name in ["Primary", "Isometric", "Front", "Top", "Right"] {
        assert!(
            world.frame.runs.iter().any(|row| row.text.trim() == name
                && row.list == branch.list
                && row.center()[1] > branch.center()[1]),
            "expected {name} beneath View"
        );
    }
}
#[when("the user right-clicks the Primary view entry")]
fn open_primary_view_menu(world: &mut ApplicationWorld) {
    right_click_row(world, "Primary");
}
#[then("the entry is selected")]
fn primary_view_entry_selected(world: &mut ApplicationWorld) {
    assert!(world.row_highlighted("Primary"));
}
#[then("its context menu offers Activate and Copy")]
fn primary_view_menu_actions(world: &mut ApplicationWorld) {
    world.expect_text("Activate");
    world.expect_text("Copy");
}
#[then("Lock is unavailable")]
fn primary_view_lock_unavailable(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Lock"));
}
#[when("the user collapses the View branch")]
fn collapse_view_branch(world: &mut ApplicationWorld) {
    world.expect_no_text("Lock");
    let branch = view_branch(world).clone();
    world.click([branch.minimum[0] as i32 - 10, branch.center()[1] as i32]);
    world.pump(Vec::new());
}
#[then("its entries are hidden")]
fn view_rows_hidden(world: &mut ApplicationWorld) {
    let branch = view_branch(world);
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|row| row.text.trim() == "Isometric" && row.list == branch.list)
    );
}
#[then("the checked Primary model-state entry remains visible")]
fn model_state_primary_still_visible(world: &mut ApplicationWorld) {
    checked_primary_model_state(world);
}
#[then("the part viewport remains empty")]
fn empty_part_viewport(world: &mut ApplicationWorld) {
    assert!(world.state().browser.unwrap().bodies.is_empty());
    world.expect_blank_region([400, 1200], [300, 700]);
}
