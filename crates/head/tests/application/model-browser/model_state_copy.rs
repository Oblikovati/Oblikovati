// SPDX-License-Identifier: GPL-2.0-only
//! Native scenario for copying and activating a model state.
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};

#[given("a named non-primary model state is active")]
fn active_named_state(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let member = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Large".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state;
    client
        .model_states()
        .activate(&RepRef { id: member.id })
        .unwrap();
    world.pump(Vec::new());
}

#[given("the Model State branch is expanded")]
fn expanded_state_branch(world: &mut ApplicationWorld) {
    super::model_state_fixture::expand_state_branch(world);
}

#[then("the duplicate action identifies the active state by name")]
fn duplicate_named(world: &mut ApplicationWorld) {
    world.expect_text("Duplicate Large");
}

#[when("I choose to duplicate the active state")]
#[when("the user chooses to duplicate the active state")]
fn duplicate_active(world: &mut ApplicationWorld) {
    world.click_text("Duplicate Large");
    world.pump(Vec::new());
}

#[then("a new model-state row appears with its generated name selected for editing")]
fn copy_selected(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("Large Copy1"));
}

#[then("the source model state remains marked active")]
fn source_active(world: &mut ApplicationWorld) {
    world.expect_text("Large (Active)");
    assert!(world.checked("Large (Active)"));
}

#[when("I enter a distinct name and press Enter")]
#[when("the user enters a distinct name and presses Enter")]
fn rename_copy(world: &mut ApplicationWorld) {
    world.type_text("Small");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
}

#[then("the source and renamed duplicate both remain in the tree")]
fn both_members(world: &mut ApplicationWorld) {
    expect_rows(world, &["Large (Active)", "Small"]);
}

#[when("I double-click the duplicate")]
#[when("the user double-clicks the duplicate")]
fn activate_copy(world: &mut ApplicationWorld) {
    world.double_click(world.run("Small").center().map(|v| v as i32));
}

#[then("the duplicate is marked active")]
fn copy_active(world: &mut ApplicationWorld) {
    world.expect_text("Small (Active)");
    assert!(world.checked("Small (Active)"));
}

#[then("the Model State branch, browser root and document tab identify the duplicate")]
fn copy_identified(world: &mut ApplicationWorld) {
    assert!((world.run("<Small>").center()[1] - world.run("Model State").center()[1]).abs() < 3.);
    let count = world
        .frame
        .runs
        .iter()
        .filter(|row| row.text.contains("Part1") && row.text.contains("Small"))
        .count();
    assert_eq!(
        count, 2,
        "expected root and document tab to show the duplicate"
    );
}
