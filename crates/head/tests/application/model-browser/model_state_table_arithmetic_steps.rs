// SPDX-License-Identifier: GPL-2.0-only
//! Native model-state parameter expression and parameter-panel checks (#4119).
use super::model_state_table_extra::{click, open};
use super::model_state_table_parameter_steps::{
    accept_parameter_table, grid_cell_text, persist_parameter_column, reopen_parameter_table,
    set_parameter_cell, three_state_prism, tree_expression,
};
use super::*;
use crate::HeadWork;
use oblikovati_api::{client::Client, wire::SetDocumentUnitsArgs};

#[given("The MCAD Software has a model-state table containing length parameter d0")]
fn table_with_inch_parameter(world: &mut ApplicationWorld) {
    three_state_prism(world);
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
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
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
    assert!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text.contains("1.5"))
            .count()
            >= 2,
        "expected both nominal and model values in the panel"
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
