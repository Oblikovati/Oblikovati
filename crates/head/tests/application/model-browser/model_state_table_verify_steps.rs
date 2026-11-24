// SPDX-License-Identifier: GPL-2.0-only
//! Native verification scenario for a model-state equation (#4119).
use super::model_state_table_extra::{click, grid_run, row_run};
use super::model_state_table_parameter_steps::{
    given_existing_parameter_table, grid_cell_text, set_parameter_cell,
};
use super::*;
use oblikovati_api::client::Client;

#[given("The MCAD Software has a model-state table containing the length parameter d0")]
fn parameter_table(world: &mut ApplicationWorld) {
    given_existing_parameter_table(world);
}
#[given("no parameter named abc exists in the part")]
fn no_abc_parameter(world: &mut ApplicationWorld) {
    let parameters = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap()
        .available_parameters
        .unwrap();
    assert!(parameters.iter().all(|parameter| parameter.name != "abc"));
}
#[when("the user replaces the active member d0 value with \"abc\" and presses Enter")]
fn enter_invalid_equation(world: &mut ApplicationWorld) {
    set_parameter_cell(world, "Copy State", "abc");
    super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
fn yellow_equation(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "abc");
    let cell = [
        grid_run(world, "d0")[0] + 25.,
        row_run(world, "Copy State")[1],
    ];
    let point = diagnostics::pointer_pixels(cell);
    let pixel = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert!(
        pixel[0] > pixel[1] && pixel[1] > pixel[2],
        "expected yellow cell, got {pixel:?}"
    );
}
#[then("the d0 cell displays \"abc\" with a yellow background")]
#[then("the yellow d0 cell remains visible")]
fn equation_is_yellow(world: &mut ApplicationWorld) {
    yellow_equation(world);
}
#[then("OK remains enabled")]
fn ok_is_enabled(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"));
}
#[when("the user selects Verify")]
fn verify(world: &mut ApplicationWorld) {
    click(world, "Verify");
}
#[then("no separate message is displayed")]
fn no_message(world: &mut ApplicationWorld) {
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("Tablecontainsanerror"))
    );
}
#[when("the user selects OK")]
fn accept(world: &mut ApplicationWorld) {
    click(world, "OK");
}
#[then(
    "a message reports an error in the table and identifies the unevaluable d0 equation \"abc\""
)]
fn error_message(world: &mut ApplicationWorld) {
    world.expect_text("Table contains an error");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("d0") && run.text.contains("abc"))
    );
}
#[when("the user dismisses the message")]
fn dismiss_message(world: &mut ApplicationWorld) {
    let button = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text.trim() == "OK")
        .unwrap()
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(button));
    world.pump(Vec::new());
}
#[then("the model-state table remains open with \"abc\" in the d0 cell")]
fn table_still_open(world: &mut ApplicationWorld) {
    world.expect_text("Model State Table");
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "abc");
}
#[then("the table closes without accepting the entry")]
fn entry_was_discarded(world: &mut ApplicationWorld) {
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "ModelStateTable")
    );
    let table = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let row = table
        .rows
        .unwrap()
        .into_iter()
        .find(|row| row.name == "Copy State")
        .unwrap()
        .ref_;
    let column = table
        .parameters
        .unwrap()
        .into_iter()
        .find(|column| column.parameter_id != 0)
        .unwrap();
    let cell = column
        .cells
        .unwrap()
        .into_iter()
        .find(|cell| cell.row == row)
        .unwrap();
    let saved = cell.expression.as_deref().unwrap_or(&column.base);
    let published = super::model_state_fixture::has_named_states(world);
    assert_eq!(saved, if published { "1 in" } else { "2 in" });
}

#[test]
fn inactive_member_with_invalid_prism_is_rejected_without_publication() {
    use oblikovati_api::wire::SetModelStateTableArgs;
    let mut world = ApplicationWorld::new();
    super::model_state_table_parameter_steps::three_state_prism(&mut world);
    super::model_state_table_parameter_steps::persist_parameter_column(&mut world);
    let client = Client::new(world.app().owner.document_host());
    let before = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let history = client.transactions().history(&Default::default()).unwrap();
    let mut parameters = before.parameters.clone().unwrap();
    let column = parameters
        .iter_mut()
        .find(|item| item.parameter_id != 0)
        .unwrap();
    let primary = before
        .rows
        .as_ref()
        .unwrap()
        .iter()
        .find(|row| row.name == "Primary")
        .unwrap()
        .ref_
        .clone();
    let cell = column
        .cells
        .as_mut()
        .unwrap()
        .iter_mut()
        .find(|cell| cell.row == primary)
        .unwrap();
    cell.expression = Some(Box::new("0 in".into()));
    let draft = SetModelStateTableArgs {
        expected_generation: before.generation,
        rows: before.rows.clone(),
        columns: before.columns.clone(),
        properties: before.properties.clone(),
        suppressions: before.suppressions.clone(),
        parameters: Some(parameters),
    };
    let verification = client.model_states().verify_table(&draft).unwrap();
    assert!(
        !verification.valid,
        "zero-depth inactive member must fail geometric replay"
    );
    assert!(
        verification
            .diagnostics
            .unwrap()
            .iter()
            .any(|diagnostic| diagnostic.row.as_deref() == Some(&primary))
    );
    assert!(client.model_states().set_table(&draft).is_err());
    assert_eq!(
        client
            .model_states()
            .get_table(&Default::default())
            .unwrap(),
        before
    );
    assert_eq!(
        client
            .transactions()
            .history(&Default::default())
            .unwrap()
            .position,
        history.position
    );
    world.close();
}
