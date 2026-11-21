// SPDX-License-Identifier: GPL-2.0-only
//! Native steps for copied model-state rows (#4119).
use super::model_state_table_extra::{click, grid_run, open, row_run};
use super::model_state_table_parameter_steps::{
    grid_cell_text, persist_parameter_column, three_state_prism,
};
use super::*;
use oblikovati_api::{
    client::Client,
    types::MODEL_STATE_TABLE_SUPPRESSION_COLUMN,
    wire::{ModelStateTableColumnRef, RepRef, SetFeatureSuppressedArgs, SetModelStateTableArgs},
};

fn setup_suppressed_rows(world: &mut ApplicationWorld) {
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
    client
        .model_states()
        .activate(&RepRef { id: capture })
        .unwrap();
    client
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id: 1,
            suppressed: true,
        })
        .unwrap();
    client
        .model_states()
        .activate(&RepRef { id: copy })
        .unwrap();
    world.pump(Vec::new());
    persist_parameter_column(world);
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let mut columns = view.columns.clone().unwrap();
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
    click_branch_arrow(world, "Model State");
    world.expect_text("Capture State");
    open(world);
}
#[given(
    "The MCAD Software has a model-state table with Primary, Capture State, and Copy State in that order"
)]
#[given(
    "The MCAD Software has a model-state table with Capture State followed by active Copy State"
)]
fn given_suppressed_rows(world: &mut ApplicationWorld) {
    setup_suppressed_rows(world);
}
#[given("Capture State has extrusion status Suppress and d0 value 1 in")]
fn capture_values(world: &mut ApplicationWorld) {
    assert_eq!(
        grid_cell_text(world, "Extrusion", "Capture State"),
        "Suppress"
    );
    assert_eq!(grid_cell_text(world, "d0", "Capture State"), "1in");
}
#[given("the extrusion is suppressed in Capture State and computed in Copy State")]
fn capture_and_copy_values(world: &mut ApplicationWorld) {
    assert_eq!(
        grid_cell_text(world, "Extrusion", "Capture State"),
        "Suppress"
    );
    assert_eq!(grid_cell_text(world, "Extrusion", "Copy State"), "Compute");
}
#[when("the user opens the context menu on the Capture State row number")]
fn open_capture_row_menu(world: &mut ApplicationWorld) {
    let point = [
        world.run("#").center()[0],
        row_run(world, "Capture State")[1],
    ];
    super::super::x11_input::right_click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[then("Insert Row and Delete Row are available")]
fn row_actions_available(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Insert Row"));
    assert!(world.run_is_enabled("Delete Row"));
}
#[then("the default-row, parameter-range and column commands appear disabled")]
fn row_other_actions_disabled(world: &mut ApplicationWorld) {
    for label in [
        "Set As Default / Compare Base Row",
        "Custom Parameter Column",
        "Column Range / Increment",
        "Custom Parameter Cell",
        "Cell Range / Increment",
        "Delete Column",
    ] {
        assert!(!world.run_is_enabled(label), "expected {label} disabled");
    }
}
#[when("the user chooses Insert Row")]
fn choose_insert_row(world: &mut ApplicationWorld) {
    click(world, "Insert Row");
}
#[when("the user inserts a row after Capture State")]
fn insert_capture_row(world: &mut ApplicationWorld) {
    open_capture_row_menu(world);
    choose_insert_row(world);
}
#[then("a generated member row appears immediately after Capture State")]
fn generated_row_position(world: &mut ApplicationWorld) {
    assert!(row_run(world, "Model State1")[1] > row_run(world, "Capture State")[1]);
    assert!(row_run(world, "Model State1")[1] < row_run(world, "Copy State")[1]);
}
#[then("the new row shows extrusion status Suppress and d0 value 1 in")]
fn generated_row_values(world: &mut ApplicationWorld) {
    grid_run(world, "Extrusion");
    assert_eq!(
        grid_cell_text(world, "Extrusion", "Model State1"),
        "Suppress"
    );
    assert_eq!(grid_cell_text(world, "d0", "Model State1"), "1in");
}
#[then("only the original three member rows are shown in their original order")]
fn original_rows_restored(world: &mut ApplicationWorld) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert_eq!(
        view.rows
            .unwrap()
            .iter()
            .map(|row| row.name.as_str())
            .collect::<Vec<_>>(),
        ["Primary", "Capture State", "Copy State"]
    );
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("ModelState1"))
    );
}
#[when("accepts the model-state table")]
fn accept_inserted_row(world: &mut ApplicationWorld) {
    click(world, "OK");
}
#[then("the generated member appears between Capture State and Copy State in the model browser")]
fn generated_browser_position(world: &mut ApplicationWorld) {
    let states = Client::new(world.app().owner.document_host())
        .model_states()
        .list(&Default::default())
        .unwrap()
        .model_states
        .unwrap();
    assert_eq!(
        states
            .iter()
            .map(|state| state.name.as_str())
            .collect::<Vec<_>>(),
        ["Primary", "Capture State", "Model State1", "Copy State"]
    );
    world.expect_text("Model State1");
}
#[then("Copy State remains active")]
fn copy_stays_active(world: &mut ApplicationWorld) {
    world.expect_text("Copy State (Active)");
}
#[when("the user activates the generated member")]
fn activate_generated(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(world.run("Model State1").center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
#[then("the generated member is marked active")]
fn generated_is_active(world: &mut ApplicationWorld) {
    world.expect_text("Model State1 (Active)");
}
#[then("the extrusion row is struck through")]
fn generated_extrusion_suppressed(world: &mut ApplicationWorld) {
    assert!(diagnostics::has_strike(world.run("Extrusion (suppressed)")));
}
#[then("the solid is absent from the viewport")]
fn generated_solid_absent(world: &mut ApplicationWorld) {
    let status = world
        .app()
        .window
        .native
        .scene_status()
        .expect("expected scene status");
    assert_eq!(status.visible_bodies, 0);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(!super::super::solid_visibility::visible_solid_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}
