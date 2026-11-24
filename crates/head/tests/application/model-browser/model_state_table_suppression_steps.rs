// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for staged feature suppression in the model-state table (#4119).
use super::model_state_fixture::has_named_states;
use super::model_state_suppression::current_state_extrusion;
use super::model_state_suppression_fixture::expect_saved_suppression;
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
    if has_named_states(world) {
        expect_saved_suppression(world);
        return;
    }
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
    let name = current_state_extrusion(world).name;
    for row in ["Primary", "Capture State"] {
        assert_eq!(grid_cell_text(world, &name, row), "Suppress");
    }
}
#[then("the extrusion column shows Compute for Copy State")]
fn copy_cell_computed(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    assert_eq!(grid_cell_text(world, &name, "Copy State"), "Compute");
}
#[when("the user edits the Primary extrusion cell")]
fn open_primary_cell(world: &mut ApplicationWorld) {
    open_suppression_cell(world, "Primary");
}
fn open_suppression_cell(world: &mut ApplicationWorld, row: &str) {
    let name = current_state_extrusion(world).name;
    let point = [grid_run(world, &name)[0], row_run(world, row)[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
pub(super) fn select_suppression_cell(world: &mut ApplicationWorld, row: &str, label: &str) {
    open_suppression_cell(world, row);
    click_suppression_choice(world, label);
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
    click_suppression_choice(world, "Compute");
}
fn click_suppression_choice(world: &mut ApplicationWorld, label: &str) {
    let point = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text == label)
        .unwrap()
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[then("the Primary cell displays Compute")]
fn primary_computed(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    assert_eq!(grid_cell_text(world, &name, "Primary"), "Compute");
}
#[then("the Primary extrusion cell displays Suppress")]
fn primary_restored(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    assert_eq!(grid_cell_text(world, &name, "Primary"), "Suppress");
}
#[then("the other two extrusion values are unchanged")]
fn other_values(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    assert_eq!(grid_cell_text(world, &name, "Capture State"), "Suppress");
    copy_cell_computed(world);
}

#[given("The MCAD Software has an extrusion suppressed in Primary and computed in Copy State")]
fn apply_fixture(world: &mut ApplicationWorld) {
    suppressed_fixture(world);
}
#[given("the active view representation permits the extrusion's solid body to be visible")]
fn active_view_shows_solid(world: &mut ApplicationWorld) {
    copy_computed(world);
}
#[when("changes the Primary extrusion cell to Compute")]
fn change_primary_to_compute(world: &mut ApplicationWorld) {
    open_primary_cell(world);
    select_compute(world);
}
#[when("activates the Primary model state")]
fn activate_primary(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(world.run("Primary").center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
#[then("the solid produced by the extrusion is visible")]
#[then("the solid is visible again")]
fn solid_is_visible(world: &mut ApplicationWorld) {
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
}
#[then("the Primary extrusion cell displays Compute")]
fn primary_value_is_compute(world: &mut ApplicationWorld) {
    primary_computed(world);
}
#[when("the user changes the Primary extrusion cell to Suppress and accepts the table")]
fn suppress_primary_and_accept(world: &mut ApplicationWorld) {
    select_suppression_cell(world, "Primary", "Suppress");
    super::model_state_table_extra::click(world, "OK");
    for _ in 0..8 {
        if !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
        {
            return;
        }
        world.pump(Vec::new());
    }
    panic!("expected the suppression table to close after acceptance");
}
