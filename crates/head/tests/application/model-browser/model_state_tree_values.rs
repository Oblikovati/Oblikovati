// SPDX-License-Identifier: GPL-2.0-only
//! Native active-member parameter tree checks (#4119).
use super::model_state_table_extra::open;
use super::model_state_table_parameter_steps::{
    grid_cell_text, persist_parameter_column, three_state_prism, tree_expression,
};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    types::MODEL_STATE_TABLE_PARAMETER_COLUMN,
    wire::{
        ModelStateTableColumnRef, ModelStateTableParameter, ModelStateTableParameterCell,
        ModelStateTableView, ParameterSetArgs, SetModelStateTableArgs,
    },
};

#[given("The MCAD Software has model states Primary and Copy State")]
fn state_fixture(world: &mut ApplicationWorld) {
    three_state_prism(world);
    persist_parameter_column(world);
    let client = Client::new(world.app().owner.document_host());
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "d1".into(),
            expression: "0 deg".into(),
            ..Default::default()
        })
        .unwrap();
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    client
        .model_states()
        .set_table(&tree_table_request(view))
        .unwrap();
    world.pump(Vec::new());
}
fn tree_table_request(view: ModelStateTableView) -> SetModelStateTableArgs {
    let d1 = view
        .available_parameters
        .as_ref()
        .unwrap()
        .iter()
        .find(|parameter| parameter.name == "d1")
        .unwrap();
    let mut columns = view.columns.clone().unwrap();
    columns.push(ModelStateTableColumnRef {
        kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
        parameter_id: d1.id,
        ..Default::default()
    });
    let parameters = tree_parameter_values(&view, d1.id);
    SetModelStateTableArgs {
        expected_generation: view.generation,
        rows: view.rows,
        columns: Some(columns),
        properties: view.properties,
        suppressions: view.suppressions,
        parameters: Some(parameters),
    }
}
fn tree_parameter_values(
    view: &ModelStateTableView,
    angle_id: u64,
) -> Vec<ModelStateTableParameter> {
    let mut parameters = view.parameters.clone().unwrap();
    let copy = view
        .rows
        .as_ref()
        .unwrap()
        .iter()
        .find(|row| row.name == "Copy State")
        .unwrap();
    let length = parameters
        .iter_mut()
        .find(|parameter| parameter.parameter_id != angle_id)
        .unwrap();
    length
        .cells
        .as_mut()
        .unwrap()
        .iter_mut()
        .find(|cell| cell.row == copy.ref_)
        .unwrap()
        .expression = Some(Box::new("1 in + 0.5 in".into()));
    parameters.push(angle_parameter(view, angle_id));
    parameters
}
fn angle_parameter(view: &ModelStateTableView, angle_id: u64) -> ModelStateTableParameter {
    ModelStateTableParameter {
        parameter_id: angle_id,
        base: "0 deg".into(),
        cells: Some(
            view.rows
                .as_ref()
                .unwrap()
                .iter()
                .map(|row| ModelStateTableParameterCell {
                    row: row.ref_.clone(),
                    expression: (row.name == "Copy State").then(|| Box::new("5 deg".into())),
                })
                .collect(),
        ),
    }
}
#[given("the table includes length parameter d0 and angular parameter d1")]
fn table_parameters_exist(world: &mut ApplicationWorld) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let names = view.available_parameters.unwrap();
    assert!(names.iter().any(|parameter| parameter.name == "d0"));
    assert!(
        names
            .iter()
            .any(|parameter| parameter.name == "d1" && parameter.unit == "deg")
    );
    assert_eq!(view.parameters.unwrap().len(), 2);
}
#[given("Primary has d0 equal to 1 in and d1 equal to 0 degrees")]
fn primary_expressions(world: &mut ApplicationWorld) {
    assert_member_expressions(world, "Primary", "1 in", "0 deg");
}
#[given("Copy State has d0 equal to \"1 in + 0.5 in\" and d1 equal to 5 degrees")]
fn copy_expressions(world: &mut ApplicationWorld) {
    assert_member_expressions(world, "Copy State", "1 in + 0.5 in", "5 deg");
}
fn assert_member_expressions(world: &mut ApplicationWorld, member: &str, d0: &str, d1: &str) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let row = view
        .rows
        .as_ref()
        .unwrap()
        .iter()
        .find(|row| row.name == member)
        .unwrap();
    for (name, expected) in [("d0", d0), ("d1", d1)] {
        let choice = view
            .available_parameters
            .as_ref()
            .unwrap()
            .iter()
            .find(|choice| choice.name == name)
            .unwrap();
        let parameter = view
            .parameters
            .as_ref()
            .unwrap()
            .iter()
            .find(|parameter| parameter.parameter_id == choice.id)
            .unwrap();
        let cell = parameter
            .cells
            .as_ref()
            .unwrap()
            .iter()
            .find(|cell| cell.row == row.ref_)
            .unwrap();
        assert_eq!(
            cell.expression
                .as_deref()
                .map(String::as_str)
                .unwrap_or(&parameter.base),
            expected
        );
    }
}
#[when("the user activates Primary and opens the model-state table")]
fn primary_table(world: &mut ApplicationWorld) {
    world.click_text("Model State");
    let point = diagnostics::pointer_pixels(world.run("Primary").center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert_eq!(
        world
            .state()
            .browser
            .unwrap()
            .model_states
            .unwrap()
            .active()
            .unwrap()
            .name,
        "Primary"
    );
    open(world);
    super::model_state_table_extra::click(world, "Parameters");
}
fn row_background(world: &mut ApplicationWorld, name: &str) -> [u8; 4] {
    let row = super::model_state_table_extra::row_run(world, name);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let point = diagnostics::pointer_pixels([row[0] - 75., row[1]]);
    let offset = (point[1] as usize * capture.extent[0] as usize + point[0] as usize) * 4;
    std::array::from_fn(|channel| capture.rgba[offset + channel])
}
fn assert_green_row(world: &mut ApplicationWorld, name: &str) {
    let pixel = row_background(world, name);
    assert!(
        pixel[1] > pixel[0] && pixel[1] > pixel[2],
        "expected pale green active row, got {pixel:?}"
    );
}
#[then("the Primary row is highlighted pale green as the active member")]
fn active_member_green(world: &mut ApplicationWorld) {
    assert_green_row(world, "Primary");
    let inactive = row_background(world, "Copy State");
    assert!(
        inactive[1] <= inactive[0],
        "expected inactive row without green highlight, got {inactive:?}"
    );
}
#[then("the parameter tree shows d0 at 1 in and d1 at 0 degrees")]
fn primary_values(world: &mut ApplicationWorld) {
    tree_expression(world, "1 in");
    world.expect_text("d1=0deg");
}
#[then("the Copy State row still shows its length expression and 5 degree angle")]
fn copy_values(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "1in+0.5in");
    assert_eq!(grid_cell_text(world, "d1", "Copy State"), "5deg");
}

#[cfg(target_os = "linux")]
#[test]
fn active_row_highlight_follows_model_state_switch() {
    let mut world = ApplicationWorld::new();
    state_fixture(&mut world);
    primary_table(&mut world);
    active_member_green(&mut world);
    super::model_state_table_extra::click(&mut world, "Cancel");
    let point = diagnostics::pointer_pixels(world.run("Copy State").center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    open(&mut world);
    assert_green_row(&mut world, "Copy State");
    let inactive = row_background(&mut world, "Primary");
    assert!(
        inactive[1] <= inactive[0],
        "expected inactive row without green highlight, got {inactive:?}"
    );
    world.close();
}
