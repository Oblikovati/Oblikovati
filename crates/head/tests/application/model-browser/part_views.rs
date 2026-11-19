// SPDX-License-Identifier: GPL-2.0-only
//! Drive the part-view copy and activation journey through the native window (#4118).
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::contract::PartDesignView as _;

#[given("the View representation branch is expanded with Primary active")]
fn expanded_primary_view(world: &mut ApplicationWorld) {
    let branch = view_branch(world).clone();
    world.click([branch.minimum[0] as i32 - 10, branch.center()[1] as i32]);
    world.pump(Vec::new());
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|row| row.text.trim() == "Primary"
                && row.list == branch.list
                && row.center()[1] > branch.center()[1]),
        "expected an actual Primary row beneath the View branch"
    );
    assert!(world.checked("Primary"));
}

#[when("the user chooses Copy from the Primary view representation context menu")]
fn copy_primary_view(world: &mut ApplicationWorld) {
    right_click_row(world, "Primary");
    choose_menu_row(world, "Copy");
}

#[then("a new view representation appears with its generated name selected for editing")]
fn copied_name_selected(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    assert!(world.text_is_selected("View1"));
}

#[then("Primary remains active")]
#[then("Primary is marked active")]
fn primary_view_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .part_design_views
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Primary"
    );
    assert!(world.checked("Primary"));
}

#[when("the user enters \"Capture View\" and presses Enter")]
fn rename_copied_view(world: &mut ApplicationWorld) {
    world.type_text("Capture View");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("the new view representation has the name \"Capture View\"")]
fn copied_view_renamed(world: &mut ApplicationWorld) {
    world.expect_text("Capture View");
}

#[when("the user double-clicks \"Capture View\"")]
fn activate_copied_view(world: &mut ApplicationWorld) {
    world.double_click(world.run("Capture View").center().map(|value| value as i32));
    world.pump(Vec::new());
}

#[then("\"Capture View\" is marked active")]
fn copied_view_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .part_design_views
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Capture View"
    );
    assert!(world.checked("Capture View"));
}

#[then("the View branch identifies \"Capture View\"")]
fn copied_view_branch_label(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Capture View");
}

#[when("the user double-clicks Primary")]
fn activate_primary_view(world: &mut ApplicationWorld) {
    world.double_click(world.run("Primary").center().map(|value| value as i32));
    world.pump(Vec::new());
}

#[then("the View branch identifies Primary")]
fn primary_view_branch_label(world: &mut ApplicationWorld) {
    expect_branch_label(world, "Primary");
}

fn expect_branch_label(world: &ApplicationWorld, name: &str) {
    let branch = view_branch(world);
    let active = world.run(&format!("<{name}>"));
    assert!((active.center()[1] - branch.center()[1]).abs() < 3.);
}

fn view_branch(world: &ApplicationWorld) -> &super::super::diagnostics::RenderedRun {
    let representations = world.run("Representations");
    world
        .frame
        .runs
        .iter()
        .filter(|row| {
            row.text.trim() == "View"
                && row.list == representations.list
                && row.center()[1] > representations.center()[1]
        })
        .min_by(|first, second| first.center()[1].total_cmp(&second.center()[1]))
        .expect("expected View inside the browser's Representations branch")
}

#[given("The MCAD Software has an empty part with a copied view active")]
fn empty_part_with_active_copy(world: &mut ApplicationWorld) {
    world.add_part();
    expanded_primary_view(world);
    copy_primary_view(world);
    rename_copied_view(world);
    activate_copied_view(world);
}

#[given("Publish is checked in the copied view context menu")]
fn published_view_menu_open(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(world.checked("Publish"));
}

#[when("the user chooses Publish from the copied view context menu")]
#[when("the user chooses Publish from the copied view context menu again")]
fn toggle_view_publication(world: &mut ApplicationWorld) {
    choose_menu_row(world, "Publish");
}

#[then("Publish is unchecked when the context menu is reopened")]
fn view_publication_unchecked(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(!world.checked("Publish"));
}

#[then("Publish is checked when the context menu is reopened")]
fn view_publication_checked(world: &mut ApplicationWorld) {
    right_click_row(world, "Capture View");
    assert!(world.checked("Publish"));
}

#[then("the copied view remains active")]
fn published_copy_remains_active(world: &mut ApplicationWorld) {
    copied_view_active(world);
}
