// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for adding a second feature parameter to the member table (#4119).
use super::super::ApplicationWorld;
use super::super::model_state_table_extra::{click, grid_run, open};
use super::super::model_state_table_parameter_steps::{
    assert_chosen_parameter, assert_tree_expression, grid_cell_text, set_parameter_cell_value,
};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::CreateModelStateArgs};

const INACTIVE: [&str; 3] = ["Primary", "Capture State", "Fourth State"];
const COPY: &str = "Copy State";

#[given("the extrusion has angular parameter d1 at 0 degrees")]
fn extrusion_taper_parameter(world: &mut ApplicationWorld) {
    let table = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let parameters = table.available_parameters.unwrap();
    let distance = parameters
        .iter()
        .find(|parameter| parameter.name == "d0")
        .unwrap();
    let taper = parameters
        .iter()
        .find(|parameter| parameter.name == "d1")
        .unwrap();
    assert_ne!(taper.feature_id, 0);
    assert_eq!(taper.feature_id, distance.feature_id);
    assert_eq!(taper.unit, "deg");
    assert_eq!(taper.base_expression, "0 deg");
}

#[given("Copy State is active among four model states")]
fn four_members(world: &mut ApplicationWorld) {
    click(world, "Cancel");
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    if view.rows.as_ref().unwrap().len() == 3 {
        client
            .model_states()
            .create(&CreateModelStateArgs {
                name: INACTIVE[2].into(),
                ..Default::default()
            })
            .unwrap();
    }
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let rows = view.rows.unwrap();
    assert_eq!(rows.len(), 4);
    let active = rows
        .iter()
        .find(|row| row.ref_.member_id == view.active_member_id)
        .unwrap();
    assert_eq!(active.name, COPY);
    world.pump(Vec::new());
    open(world);
    click(world, "Parameters");
}

#[when("the user adds d1 to the chosen parameters")]
fn add_taper_parameter(world: &mut ApplicationWorld) {
    click(world, "d1");
    click(world, "Add >");
}

#[then("the table shows both d0 and d1 columns")]
fn both_columns(world: &mut ApplicationWorld) {
    for name in ["d0", "d1"] {
        grid_run(world, name);
    }
}

#[then("each d1 member cell shows 0 degrees")]
fn no_member_taper(world: &mut ApplicationWorld) {
    inherited_taper(world);
    assert_eq!(grid_cell_text(world, "d1", COPY), "0deg");
}

#[when("the user changes the Copy State d1 cell to \"5 deg\"")]
fn set_copy_taper(world: &mut ApplicationWorld) {
    set_parameter_cell_value(world, "d1", COPY, "5 deg");
}

#[then("the chosen parameters include d0 and d1")]
fn retained_columns(world: &mut ApplicationWorld) {
    click(world, "Parameters");
    for name in ["d0", "d1"] {
        assert_chosen_parameter(world, name);
    }
}

#[then("Copy State shows d1 as 5 degrees")]
fn copy_taper(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "d1", COPY), "5deg");
}

#[then("the other three members show d1 as 0 degrees")]
fn inherited_taper(world: &mut ApplicationWorld) {
    for member in INACTIVE {
        assert_eq!(grid_cell_text(world, "d1", member), "0deg");
    }
}

#[then("the parameter tree shows d1 at 5 degrees")]
fn active_taper(world: &mut ApplicationWorld) {
    assert_tree_expression(world, "d1", "5 deg");
}
