// SPDX-License-Identifier: GPL-2.0-only
//! Exercise model-state scope with native pointer input.
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};
use oblikovati_model::model_state::ModelStateEditScope::{AllMembers, SingleMember};

#[given("an empty part with Primary and a second model state is active in the MCAD Software")]
fn active_variant(world: &mut ApplicationWorld) {
    world.add_part();
    let client = Client::new(world.app().owner.document_host());
    let member = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Variant".into(),
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
#[given("the second state is active and the edit scope is Single Member")]
fn single_scope(world: &mut ApplicationWorld) {
    let states = world.state().browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().name, "Variant");
    assert_eq!(states.edit_scope(), SingleMember);
}
#[when("I click the edit-scope control on the Model State branch")]
#[when("the user clicks the edit-scope control on the Model State branch")]
#[when("I click the edit-scope control again")]
#[when("the user clicks the edit-scope control again")]
fn toggle_scope(world: &mut ApplicationWorld) {
    let scope = world
        .state()
        .browser
        .unwrap()
        .model_states
        .unwrap()
        .edit_scope();
    world.click_text(if scope == SingleMember { "1" } else { "All" });
    world.pump(Vec::new());
}
#[then("its tooltip identifies All Members as the edit scope")]
fn all_scope_tooltip(world: &mut ApplicationWorld) {
    world.expect_text("All Members");
}
#[then("its tooltip identifies Single Member as the edit scope")]
fn single_scope_tooltip(world: &mut ApplicationWorld) {
    world.expect_text("Single Member");
}
#[then("the browser root indicates All")]
fn root_all(world: &mut ApplicationWorld) {
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("Part1") && run.text.contains("(All)"))
    );
}
#[then("the All indicator is removed from the browser root")]
fn root_single(world: &mut ApplicationWorld) {
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("Part1") && run.text.contains("(All)"))
    );
}
#[then("the second model state remains active")]
fn variant_remains_active(world: &mut ApplicationWorld) {
    let states = world.state().browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().name, "Variant");
    assert!(matches!(states.edit_scope(), SingleMember | AllMembers));
}
