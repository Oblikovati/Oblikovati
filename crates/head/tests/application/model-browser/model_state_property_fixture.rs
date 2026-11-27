// SPDX-License-Identifier: GPL-2.0-only
//! Create the Category column and its member value through the native table.
use super::model_state_table::{
    accept_table, add_category, category_in_grid, edit_active_cell, open_properties,
    select_category,
};
use super::*;
use cucumber::given;
use oblikovati_api::{client::Client, types::VariantValue, wire::ModelStateTableProperty};
use oblikovati_model::model_state::ModelStateMember;

#[given("the user adds and accepts the Category property column in the model-state table")]
fn add_saved_category(world: &mut ApplicationWorld) {
    open_properties(world);
    select_category(world);
    add_category(world);
    category_in_grid(world);
    accept_table(world);
    category_column(world);
}

#[given("the user sets the active state Category to Prototype and accepts the table")]
fn set_saved_prototype(world: &mut ApplicationWorld) {
    open_properties(world);
    edit_active_cell(world);
    world.type_text("Prototype");
    accept_table(world);
    expect_saved_category(world, "Prototype");
}

#[given("the user reopens the model-state table and selects the Properties category")]
fn reopen_properties_for_fixture(world: &mut ApplicationWorld) {
    open_properties(world);
    world.expect_text("Chosen properties");
}

pub(super) fn active_table_member(world: &mut ApplicationWorld) -> ModelStateMember {
    let states = world.state().browser.unwrap().model_states.unwrap();
    states
        .active()
        .expect("expected an active table member")
        .clone()
}

pub(super) fn category_column(world: &mut ApplicationWorld) -> ModelStateTableProperty {
    let table = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    table
        .properties
        .unwrap_or_default()
        .into_iter()
        .find(|column| {
            column.property.name == "Category" && column.property.set == "Summary Information"
        })
        .expect("expected the saved Category column")
}

pub(super) fn expect_saved_category(world: &mut ApplicationWorld, expected: &str) {
    let member = active_table_member(world);
    let value = category_column(world)
        .cells
        .unwrap()
        .into_iter()
        .find(|cell| cell.row.member_id == member.id)
        .and_then(|cell| cell.value.map(|value| value.value));
    assert_eq!(
        value,
        Some(VariantValue::String(expected.into())),
        "{member:?}"
    );
}
