// SPDX-License-Identifier: GPL-2.0-only
//! Native steps for model-state parameter columns (#4119).
use super::model_state_table_extra::{click, grid_run, open, row_run};
use super::*;
use oblikovati_api::{client::Client, wire::CreateModelStateArgs};

pub(super) fn three_state_prism(world: &mut ApplicationWorld) {
    use oblikovati_model::feature::{AuthoredFeature, FeatureId};
    use oblikovati_persistence::package::DocumentPackage;
    let mut snapshot = crate::scene_fixture::selected_sketch_document();
    snapshot
        .parameters
        .insert_expression(3, "d0", oblikovati_api::types::MODEL_PARAM, "1 in")
        .unwrap();
    let feature = AuthoredFeature::sketch_prism(
        FeatureId(1),
        "Extrusion",
        crate::scene_fixture::selected_prism_definition(),
        &snapshot,
    )
    .unwrap()
    .with_owned_scalar("distance", 3)
    .unwrap();
    snapshot.work_geometry.append(feature).unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap();
    let path = world.scratch_folder().join("state-parameter.opd");
    std::fs::write(&path, package.encode().unwrap()).unwrap();
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            visible: true,
            ..Default::default()
        })
        .unwrap();
    client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Capture State".into(),
            ..Default::default()
        })
        .unwrap();
    let copy = client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Copy State".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    client
        .model_states()
        .activate(&oblikovati_api::wire::RepRef { id: copy })
        .unwrap();
    world.pump(Vec::new());
}
pub(super) fn grid_cell_text(world: &ApplicationWorld, column: &str, row: &str) -> String {
    let x = grid_run(world, column)[0];
    let y = row_run(world, row)[1];
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            (run.center()[1] - y).abs() < 5. && run.minimum[0] <= x && run.maximum[0] >= x - 70.
        })
        .map(|run| {
            run.text
                .chars()
                .filter(|character| !character.is_whitespace())
                .collect::<String>()
        })
        .find(|text| !text.is_empty())
        .unwrap_or_default()
}
pub(super) fn set_parameter_cell(world: &mut ApplicationWorld, row: &str, value: &str) {
    let point = [grid_run(world, "d0")[0], row_run(world, row)[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    for character in value.chars() {
        if character == '+' {
            super::super::x11_input::press_shifted_keysym(u32::from('='));
        } else {
            super::super::x11_input::press_path_character(character);
        }
    }
    world.pump(Vec::new());
}

#[given("The MCAD Software has a part with three model-state members")]
#[given("The MCAD Software has a part with Primary, Capture State, and Copy State model states")]
fn given_three_member_prism(world: &mut ApplicationWorld) {
    if super::model_state_fixture::has_named_states(world) {
        let browser = world.state().browser.unwrap();
        assert_eq!(browser.model_states.unwrap().list().len(), 3);
        assert_eq!(browser.features.len(), 1);
        assert_eq!(browser.features[0].kind, "Extrusion");
        return;
    }
    three_state_prism(world);
}
#[given("an extrusion has a length parameter d0 of 1 in")]
#[given("an extrusion has a length parameter d0 of 1 in in each state")]
pub(super) fn given_length_parameter(world: &mut ApplicationWorld) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert!(
        view.available_parameters
            .unwrap()
            .iter()
            .any(|parameter| parameter.name == "d0"
                && parameter.base_expression == "1 in"
                && parameter.feature_id != 0)
    );
}
#[given("the model-state table does not include d0")]
#[given("the model-state table does not include d0 among its chosen parameters")]
fn given_no_parameter_column(world: &mut ApplicationWorld) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert!(
        !view
            .columns
            .unwrap()
            .iter()
            .any(|column| column.kind == oblikovati_api::types::MODEL_STATE_TABLE_PARAMETER_COLUMN)
    );
}
#[given("Copy State is active")]
fn given_copy_active(world: &mut ApplicationWorld) {
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert!(
        view.rows
            .unwrap()
            .iter()
            .any(|row| row.name == "Copy State" && row.ref_.member_id == view.active_member_id)
    );
}
#[when("selects d0 under the extrusion on the Parameters tab")]
fn choose_parameter(world: &mut ApplicationWorld) {
    click(world, "Parameters");
    click(world, "d0");
}
#[when("transfers d0 to the chosen parameters")]
fn transfer_parameter(world: &mut ApplicationWorld) {
    click(world, "Add >");
}
#[then("the chosen parameters include d0")]
#[then("d0 is listed among the chosen parameters")]
fn chosen_parameter(world: &mut ApplicationWorld) {
    world.expect_text("Chosen parameters");
    let heading = world.run("Chosen parameters");
    assert!(
        world.frame.runs.iter().any(|run| {
            run.text.trim() == "d0"
                && run.center()[1] > heading.center()[1]
                && (run.minimum[0] - heading.minimum[0]).abs() < 15.
        }),
        "expected d0 in chosen parameters; visible runs: {:?}",
        world.frame.runs
    );
}
#[then("the grid has a d0 column showing 1 in for each member")]
fn initial_parameter_values(world: &mut ApplicationWorld) {
    for row in ["Primary", "Capture State", "Copy State"] {
        assert_eq!(grid_cell_text(world, "d0", row), "1in");
    }
}
#[when("the user changes the Copy State d0 cell to 2 in")]
#[when("changes the Copy State d0 cell to 2 in")]
fn change_copy_parameter(world: &mut ApplicationWorld) {
    set_parameter_cell(world, "Copy State", "2 in");
}
#[then("that cell displays 2 in")]
#[then("the Copy State d0 cell displays 2 in")]
#[then("the Copy State d0 cell still displays 2 in")]
fn copy_parameter_display(world: &mut ApplicationWorld) {
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "2in");
}
#[then("the other two d0 cells display 1 in")]
#[then("the Primary and Capture State d0 cells display 1 in")]
#[then("the Primary and Capture State d0 cells still display 1 in")]
fn other_parameter_display(world: &mut ApplicationWorld) {
    for row in ["Primary", "Capture State"] {
        assert_eq!(grid_cell_text(world, "d0", row), "1in");
    }
}
#[when("the user cancels and reopens the model-state table")]
#[when("the user cancels and reopens the table")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    click(world, "Cancel");
    open(world);
}
#[then("d0 is absent from the chosen parameters")]
fn parameter_not_chosen(world: &mut ApplicationWorld) {
    click(world, "Parameters");
    removed_parameter_is_hidden(world);
}
#[then("the grid has no d0 column")]
fn no_parameter_grid(world: &mut ApplicationWorld) {
    let header = world.run("Member").center()[1];
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text.trim() == "d0" && (run.center()[1] - header).abs() < 3.)
    );
}

#[when("adds d0 from the extrusion to the chosen parameters")]
pub(super) fn add_parameter_from_feature(world: &mut ApplicationWorld) {
    choose_parameter(world);
    transfer_parameter(world);
}
#[when("accepts the table")]
pub(super) fn accept_parameter_table(world: &mut ApplicationWorld) {
    click(world, "OK");
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
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    panic!(
        "expected model-state table to close after accepting its checked draft; committed host view: {view:?}; rendered: {:?}",
        world.frame.runs
    );
}
#[when("reopens the model-state table")]
pub(super) fn reopen_parameter_table(world: &mut ApplicationWorld) {
    open(world);
}
#[then("the parameter tree displays d0 as 2 in")]
fn tree_displays_two_inches(world: &mut ApplicationWorld) {
    tree_expression(world, "2 in");
    super::model_state_parameter_fixture::expect_native_extrusion_centroid(world, 25.4);
}
#[then("the parameter tree displays d0 as 1 in")]
fn tree_displays_one_inch(world: &mut ApplicationWorld) {
    tree_expression(world, "1 in");
    super::model_state_parameter_fixture::expect_native_extrusion_centroid(world, 12.7);
}
pub(super) fn tree_expression(world: &ApplicationWorld, expression: &str) {
    let expected = format!(
        "d0={}",
        expression
            .chars()
            .filter(|character| !character.is_whitespace())
            .collect::<String>()
    );
    let heading = world.run("Chosen parameters");
    let header = world.run("Member");
    assert!(
        world.frame.runs.iter().any(|run| {
            run.center()[0] < heading.minimum[0]
                && run.center()[1] < header.center()[1]
                && run
                    .text
                    .chars()
                    .filter(|character| !character.is_whitespace())
                    .collect::<String>()
                    == expected
        }),
        "expected {expected:?} in parameter tree; visible runs: {:?}",
        world.frame.runs
    );
}

#[when("the user closes the table and activates Primary")]
fn close_table_and_activate_primary(world: &mut ApplicationWorld) {
    click(world, "Cancel");
    let point = diagnostics::pointer_pixels(world.run("Primary").center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}

pub(super) fn persist_parameter_column(world: &mut ApplicationWorld) {
    use oblikovati_api::{
        types::MODEL_STATE_TABLE_PARAMETER_COLUMN,
        wire::{
            ModelStateTableColumnRef, ModelStateTableParameter, ModelStateTableParameterCell,
            SetModelStateTableArgs,
        },
    };
    let client = Client::new(world.app().owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let parameter = view
        .available_parameters
        .as_ref()
        .unwrap()
        .iter()
        .find(|item| item.name == "d0")
        .unwrap();
    let rows = view.rows.as_ref().unwrap();
    let cells = rows
        .iter()
        .map(|row| ModelStateTableParameterCell {
            row: row.ref_.clone(),
            expression: (row.name == "Copy State").then(|| Box::new("2 in".into())),
        })
        .collect();
    let request = SetModelStateTableArgs {
        expected_generation: view.generation,
        rows: view.rows.clone(),
        columns: Some(vec![ModelStateTableColumnRef {
            kind: MODEL_STATE_TABLE_PARAMETER_COLUMN,
            parameter_id: parameter.id,
            ..Default::default()
        }]),
        properties: view.properties.clone(),
        suppressions: view.suppressions.clone(),
        parameters: Some(vec![ModelStateTableParameter {
            parameter_id: parameter.id,
            base: parameter.base_expression.clone(),
            cells: Some(cells),
        }]),
    };
    client.model_states().set_table(&request).unwrap();
    world.pump(Vec::new());
}
#[given("The MCAD Software has a model-state table containing parameter d0")]
pub(super) fn given_existing_parameter_table(world: &mut ApplicationWorld) {
    if super::model_state_fixture::has_named_states(world) {
        super::model_state_parameter_fixture::expect_prepared_distance_table(world);
        return;
    }
    three_state_prism(world);
    persist_parameter_column(world);
    open(world);
    click(world, "Parameters");
}
#[given(
    "the d0 values for Primary, Capture State, and Copy State are 1 in, 1 in, and 2 in respectively"
)]
#[then(
    "the d0 values for Primary, Capture State, and Copy State are 1 in, 1 in, and 2 in respectively"
)]
fn expected_member_parameter_values(world: &mut ApplicationWorld) {
    for row in ["Primary", "Capture State"] {
        assert_eq!(grid_cell_text(world, "d0", row), "1in");
    }
    assert_eq!(grid_cell_text(world, "d0", "Copy State"), "2in");
}
#[when("the user selects d0 in the chosen parameters")]
fn select_chosen_parameter(world: &mut ApplicationWorld) {
    let heading = world.run("Chosen parameters").clone();
    let point = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text.trim() == "d0"
                && run.center()[1] > heading.center()[1]
                && (run.minimum[0] - heading.minimum[0]).abs() < 15.
        })
        .expect("expected d0 in chosen parameters")
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
#[when("transfers it out of the chosen parameters")]
fn remove_chosen_parameter(world: &mut ApplicationWorld) {
    click(world, "< Remove");
}
#[then("d0 is absent from the chosen parameters and the grid")]
fn removed_parameter_is_hidden(world: &mut ApplicationWorld) {
    let heading = world.run("Chosen parameters");
    assert!(
        !world.frame.runs.iter().any(|run| {
            run.text.trim() == "d0"
                && run.center()[1] > heading.center()[1]
                && (run.minimum[0] - heading.minimum[0]).abs() < 15.
        }),
        "d0 is still in chosen parameters"
    );
    no_parameter_grid(world);
}
#[then("d0 is present in the chosen parameters and the grid")]
fn restored_parameter_is_visible(world: &mut ApplicationWorld) {
    chosen_parameter(world);
    grid_run(world, "d0");
}

#[test]
fn native_invalid_equation_is_checked_without_changing_the_part() {
    let mut world = ApplicationWorld::new();
    given_existing_parameter_table(&mut world);
    let before = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    set_parameter_cell(&mut world, "Copy State", "abc");
    super::super::x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_eq!(grid_cell_text(&world, "d0", "Copy State"), "abc");
    let cell = [
        grid_run(&world, "d0")[0] + 25.,
        row_run(&world, "Copy State")[1],
    ];
    let point = diagnostics::pointer_pixels(cell);
    let pixel = world.pixel_at([point[0] as usize, point[1] as usize]);
    assert!(
        pixel[0] > pixel[1] && pixel[1] > pixel[2],
        "expected yellow cell, got {pixel:?}"
    );
    click(&mut world, "Verify");
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("Table contains an error"))
    );
    click(&mut world, "OK");
    world.expect_text("Table contains an error");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text.contains("d0") && run.text.contains("abc"))
    );
    let modal_ok = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| run.text.trim() == "OK")
        .unwrap()
        .center();
    super::super::x11_input::click(diagnostics::pointer_pixels(modal_ok));
    world.pump(Vec::new());
    assert_eq!(grid_cell_text(&world, "d0", "Copy State"), "abc");
    click(&mut world, "Cancel");
    let after = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert_eq!(before, after);
    world.close();
}
