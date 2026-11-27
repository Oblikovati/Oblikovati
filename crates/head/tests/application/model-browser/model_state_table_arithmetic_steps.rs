// SPDX-License-Identifier: GPL-2.0-only
//! Native model-state parameter expression and parameter-panel checks (#4119).
use super::model_state_table_extra::{click, open};
use super::model_state_table_parameter_steps::{
    accept_parameter_table, grid_cell_text, persist_parameter_column, reopen_parameter_table,
    set_parameter_cell, tapered_three_state_prism, three_state_prism, tree_expression,
};
use super::*;
use crate::HeadWork;
use oblikovati_api::{client::Client, wire::SetDocumentUnitsArgs};

#[path = "model_state_taper_parameter.rs"]
mod taper;

#[given("The MCAD Software has a model-state table containing length parameter d0")]
fn table_with_inch_parameter(world: &mut ApplicationWorld) {
    if super::model_state_fixture::has_named_states(world) {
        super::model_state_parameter_fixture::expect_prepared_distance_table(world);
        return;
    }
    tapered_three_state_prism(world);
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("in".into())),
            ..Default::default()
        })
        .unwrap();
    persist_parameter_column(world);
    open(world);
    click(world, "Parameters");
}
#[given("the Primary and Capture State d0 values are 1 in")]
fn inherited_values(world: &mut ApplicationWorld) {
    for row in ["Primary", "Capture State"] {
        assert_eq!(grid_cell_text(world, "d0", row), "1in");
    }
}
#[when("the user enters \"1 in + 0.5 in\" in the Copy State d0 cell")]
fn set_expression(world: &mut ApplicationWorld) {
    set_parameter_cell(world, "Copy State", "1 in + 0.5 in");
}
#[when("accepts and reopens the model-state table")]
#[when("accepts and reopens the table")]
fn accept_and_reopen(world: &mut ApplicationWorld) {
    accept_parameter_table(world);
    reopen_parameter_table(world);
}
#[then("the Copy State d0 cell displays the entered arithmetic expression")]
fn cell_expression(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "1in+0.5in");
}
#[then("the parameter tree displays the active member expression")]
fn tree_expression_matches(world: &mut ApplicationWorld) {
    tree_expression(world, "1 in + 0.5 in");
}
#[when("the user closes the table and opens Parameters")]
fn open_parameters(world: &mut ApplicationWorld) {
    click(world, "Cancel");
    open_manage_parameters(world);
}
fn open_manage_parameters(world: &mut ApplicationWorld) {
    click(world, "Manage");
    let point = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Parameters" && run.center()[1] < 150.)
        .expect("expected Parameters in Manage ribbon")
        .center();
    world.desktop_click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[then("d0 has nominal and model values of 1.5 in")]
fn nominal_and_model_values(world: &mut ApplicationWorld) {
    world.expect_text("Parameters");
    world.expect_text("Nominal Value");
    world.expect_text("Model Value");
    let snapshot = world.app().owner.gui_parameter_snapshot().unwrap().unwrap();
    let row = snapshot
        .rows
        .iter()
        .find(|row| row.detail.parameter_info.name == "d0")
        .unwrap();
    assert_eq!(row.detail.parameter_info.expression, "1 in + 0.5 in");
    assert!((row.detail.model_value - 3.81).abs() < 1e-8);
    let nominal = &row.detail.parameter_info.value;
    let model = &row.model_value_text;
    assert!(
        world.shows(nominal) && world.shows(model),
        "expected nominal {nominal:?} and model {model:?} in the panel"
    );
}

#[test]
fn native_parameter_panel_distinguishes_nominal_and_toleranced_model_value() {
    use oblikovati_api::wire::{ParameterToleranceArgs, ParameterUpdateArgs};
    let mut world = ApplicationWorld::new();
    three_state_prism(&mut world);
    let client = Client::new(world.app().owner.document_host());
    client
        .parameters()
        .set_tolerance(&ParameterToleranceArgs {
            name: "width".into(),
            mode: "symmetric".into(),
            upper: "1 cm".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .update(&ParameterUpdateArgs {
            name: "width".into(),
            model_value_type: Some(Box::new("upper".into())),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    open_manage_parameters(&mut world);
    world.expect_text("Nominal Value");
    world.expect_text("Model Value");
    let y = world.run("width").center()[1];
    for value in ["40mm", "50mm"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| (run.center()[1] - y).abs() < 4. && run.text == value),
            "expected {value} on the width row: {:?}",
            world.frame.runs
        );
    }
    world.close();
}

#[given("The MCAD Software has a Primary model state with d0 set to 1 in")]
fn primary_parameter_fixture(world: &mut ApplicationWorld) {
    three_state_prism(world);
    Client::new(world.app().owner.document_host())
        .documents()
        .set_units(&SetDocumentUnitsArgs {
            length_unit: Some(Box::new("in".into())),
            ..Default::default()
        })
        .unwrap();
    persist_parameter_column(world);
}
#[given("Copy State has d0 set to \"1 in + 0.5 in\"")]
fn copy_expression_fixture(world: &mut ApplicationWorld) {
    use oblikovati_api::wire::SetModelStateTableArgs;
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let mut parameters = view.parameters.unwrap();
    let column = parameters
        .iter_mut()
        .find(|column| column.parameter_id == 3)
        .unwrap();
    let cell = column
        .cells
        .as_mut()
        .unwrap()
        .iter_mut()
        .find(|cell| cell.row.member_id == view.active_member_id)
        .unwrap();
    cell.expression = Some(Box::new("1 in + 0.5 in".into()));
    client
        .model_states()
        .set_table(&SetModelStateTableArgs {
            expected_generation: view.generation,
            rows: view.rows,
            columns: view.columns,
            properties: view.properties,
            suppressions: view.suppressions,
            parameters: Some(parameters),
        })
        .unwrap();
    world.pump(Vec::new());
}
#[when("the user opens Parameters")]
#[when("opens Parameters again")]
fn open_parameter_editor(world: &mut ApplicationWorld) {
    open_manage_parameters(world);
}
fn assert_parameter_row(world: &mut ApplicationWorld, expression: &str, value: &str) {
    let row = world.run("d0").center()[1];
    let expression = expression
        .chars()
        .filter(|character| !character.is_whitespace())
        .collect::<String>();
    let value = value
        .chars()
        .filter(|character| !character.is_whitespace())
        .collect::<String>();
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| (run.center()[1] - row).abs() < 4. && run.text == expression)
    );
    assert!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| (run.center()[1] - row).abs() < 4. && run.text == value)
            .count()
            >= 2,
        "expected nominal and model values {value:?} in d0 row"
    );
}
#[then("d0 shows the arithmetic equation and nominal and model values of 1.5 in")]
fn copy_parameter_values(world: &mut ApplicationWorld) {
    assert_parameter_row(world, "1 in + 0.5 in", "1.5 in");
}
#[when("the user closes Parameters and activates Primary")]
fn activate_primary(world: &mut ApplicationWorld) {
    click(world, "Done");
    super::model_state_suppression::activate_state_row(world, "Primary");
}
#[then("d0 shows equation 1 in and nominal and model values of 1 in")]
fn primary_parameter_values(world: &mut ApplicationWorld) {
    assert_parameter_row(world, "1 in", "1 in");
}
