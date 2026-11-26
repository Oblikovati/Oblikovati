// SPDX-License-Identifier: GPL-2.0-only
//! Exercise model-state scope with native pointer input.
use super::model_state_fixture::{COPY_STATE, expect_empty_named_states, has_named_states};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};
use oblikovati_model::model_state::ModelStateEditScope::{AllMembers, SingleMember};

#[given("an empty part with Primary and a second model state is active in the MCAD Software")]
fn active_variant(world: &mut ApplicationWorld) {
    if expect_empty_named_states(world) {
        return;
    }
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
    let expected = scope_member_name(world);
    let states = world.state().browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().name, expected);
    assert_eq!(states.edit_scope(), SingleMember);
}
#[when("I click the edit-scope control on the Model State branch")]
#[when("the user clicks the edit-scope control on the Model State branch")]
#[when("I click the edit-scope control again")]
#[when("the user clicks the edit-scope control again")]
fn toggle_scope(world: &mut ApplicationWorld) {
    let scope = edit_scope(world);
    world.click(scope_badge(world, scope == AllMembers));
    world.pump(Vec::new());
    assert_ne!(edit_scope(world), scope);
    world.hover(scope_badge(world, scope == SingleMember));
}

fn edit_scope(world: &mut ApplicationWorld) -> oblikovati_model::model_state::ModelStateEditScope {
    world
        .state()
        .browser
        .unwrap()
        .model_states
        .unwrap()
        .edit_scope()
}

fn scope_badge(world: &ApplicationWorld, all: bool) -> [i32; 2] {
    let branch = world.run_in_window("Model", "ModelState");
    let label = if all { "All" } else { "1" };
    let suffix = format!(">{label}");
    let run = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.list == branch.list
                && (run.center()[1] - branch.center()[1]).abs() < 3.
                && run.minimum[0] > branch.maximum[0]
                && (run.text == label || run.text.ends_with(&suffix))
        })
        .expect("expected the scope badge on the Model State branch");
    diagnostics::pointer_pixels([run.maximum[0] - 3., run.center()[1]])
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
        super::feature_rows::root_caption(world)
            .text
            .contains("(All)")
    );
}
#[then("the All indicator is removed from the browser root")]
fn root_single(world: &mut ApplicationWorld) {
    assert!(
        !super::feature_rows::root_caption(world)
            .text
            .contains("(All)")
    );
}
#[then("the second model state remains active")]
fn variant_remains_active(world: &mut ApplicationWorld) {
    let expected = scope_member_name(world);
    let states = world.state().browser.unwrap().model_states.unwrap();
    assert_eq!(states.active().unwrap().name, expected);
    assert!(matches!(states.edit_scope(), SingleMember | AllMembers));
}

fn scope_member_name(world: &mut ApplicationWorld) -> &'static str {
    if has_named_states(world) {
        COPY_STATE
    } else {
        "Variant"
    }
}
