// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for staged feature suppression in the model-state table (#4119).
use super::model_state_table_extra::{grid_run, row_run};
use super::model_state_table_parameter_steps::{grid_cell_text, three_state_prism};
use super::*;
use oblikovati_api::{
    client::Client,
    types::MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
    wire::{ModelStateTableColumnRef, RepRef, SetFeatureSuppressedArgs, SetModelStateTableArgs},
};

#[given("The MCAD Software has a part with an extrusion suppressed in Primary and Capture State")]
fn suppressed_fixture(world: &mut ApplicationWorld) {
    three_state_prism(world);
    let client = Client::new(world.app().owner.document_host());
    let states = client
        .model_states()
        .list(&Default::default())
        .unwrap()
        .model_states
        .unwrap();
    let capture = states
        .iter()
        .find(|state| state.name == "Capture State")
        .unwrap()
        .id;
    let copy = states
        .iter()
        .find(|state| state.name == "Copy State")
        .unwrap()
        .id;
    for id in [1, capture] {
        client.model_states().activate(&RepRef { id }).unwrap();
        client
            .features()
            .set_suppressed(&SetFeatureSuppressedArgs {
                id: 1,
                suppressed: true,
            })
            .unwrap();
    }
    client
        .model_states()
        .activate(&RepRef { id: copy })
        .unwrap();
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let mut columns = view.columns.unwrap_or_default();
    columns.push(ModelStateTableColumnRef {
        kind: MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
        feature_id: 1,
        ..Default::default()
    });
    client
        .model_states()
        .set_table(&SetModelStateTableArgs {
            expected_generation: view.generation,
            rows: view.rows,
            columns: Some(columns),
            properties: view.properties,
            suppressions: view.suppressions,
            parameters: view.parameters,
        })
        .unwrap();
    world.pump(Vec::new());
}
#[given("the extrusion is computed in Copy State")]
fn copy_computed(world: &mut ApplicationWorld) {
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 1);
}
#[then("the extrusion column shows Suppress for Primary and Capture State")]
fn primary_and_capture_suppressed(world: &mut ApplicationWorld) {
    for row in ["Primary", "Capture State"] {
        assert_eq!(grid_cell_text(world, "Extrusion", row), "Suppress");
    }
}
#[then("the extrusion column shows Compute for Copy State")]
fn copy_cell_computed(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "Extrusion", "Copy State"), "Compute");
}
#[when("the user edits the Primary extrusion cell")]
fn open_primary_cell(world: &mut ApplicationWorld) {
    let point = [
        grid_run(world, "Extrusion")[0],
        row_run(world, "Primary")[1],
    ];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[then("the cell offers Compute and Suppress")]
fn choices_visible(world: &mut ApplicationWorld) {
    let count = |label: &str| {
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == label)
            .count()
    };
    assert!(count("Compute") >= 2 && count("Suppress") >= 3);
}
#[when("the user selects Compute")]
fn select_compute(world: &mut ApplicationWorld) {
    let point = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == "Compute")
        .unwrap()
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[then("the Primary cell displays Compute")]
fn primary_computed(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "Extrusion", "Primary"), "Compute");
}
#[then("the Primary extrusion cell displays Suppress")]
fn primary_restored(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "Extrusion", "Primary"), "Suppress");
}
#[then("the other two extrusion values are unchanged")]
fn other_values(world: &mut ApplicationWorld) {
    assert_eq!(
        grid_cell_text(world, "Extrusion", "Capture State"),
        "Suppress"
    );
    assert_eq!(grid_cell_text(world, "Extrusion", "Copy State"), "Compute");
}
