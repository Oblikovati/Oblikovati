// SPDX-License-Identifier: GPL-2.0-only
//! Native mouse and keyboard coverage for the unified model-state table (#4119).
use super::*;
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, ParameterSetArgs},
};

pub(super) fn click(world: &mut ApplicationWorld, label: &str) {
    world.desktop_click(diagnostics::pointer_pixels(world.run(label).center()));
    world.pump(Vec::new());
}
fn setup(world: &mut ApplicationWorld) {
    world.add_part();
    let client = Client::new(world.app().owner.document_host());
    client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Copy State".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}
pub(super) fn open(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(world.run("Model State").center());
    world.desktop_right_click(point);
    world.pump(Vec::new());
    click(world, "Edit Table");
}
fn add_parameter(world: &mut ApplicationWorld) {
    click(world, "Parameters");
    click(world, "size");
    click(world, "Add >");
}
pub(super) fn grid_run(world: &ApplicationWorld, name: &str) -> [f32; 2] {
    let member = world.run("Member").center();
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text.trim() == name && (run.center()[1] - member[1]).abs() < 3.)
        .unwrap_or_else(|| panic!("expected {name:?} in the model-state grid header"))
        .center()
}
pub(super) fn row_run(world: &ApplicationWorld, name: &str) -> [f32; 2] {
    let header = world.run("Member").center();
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text
                .chars()
                .filter(|character| !character.is_whitespace())
                .collect::<String>()
                == name
                    .chars()
                    .filter(|character| !character.is_whitespace())
                    .collect::<String>()
                && run.center()[1] > header[1]
                && (run.minimum[0] - world.run("Member").minimum[0]).abs() < 12.
        })
        .unwrap_or_else(|| {
            panic!(
                "expected {name:?} in the model-state grid; matching runs: {:?}; header: {:?}",
                world
                    .frame
                    .runs
                    .iter()
                    .filter(|run| run.text.contains("Copy") || run.text.contains("Member"))
                    .collect::<Vec<_>>(),
                world.run("Member")
            )
        })
        .center()
}

#[test]
fn native_parameter_and_inserted_row_cancel_without_changes() {
    let mut world = ApplicationWorld::new();
    setup(&mut world);
    open(&mut world);
    add_parameter(&mut world);
    let x = world.run("#").center()[0];
    let y = row_run(&world, "Copy State")[1];
    super::super::x11_input::right_click(diagnostics::pointer_pixels([x, y]));
    world.pump(Vec::new());
    click(&mut world, "Insert Row");
    world.expect_text("Model State1");
    click(&mut world, "Cancel");
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert_eq!(view.rows.unwrap().len(), 2);
    assert!(view.columns.unwrap().is_empty());
    world.close();
}

#[test]
fn native_parameter_edit_accepts_and_survives_undo_redo() {
    let mut world = ApplicationWorld::new();
    setup(&mut world);
    open(&mut world);
    add_parameter(&mut world);
    let x = grid_run(&world, "size")[0];
    let y = row_run(&world, "Copy State")[1];
    super::super::x11_input::click(diagnostics::pointer_pixels([x, y]));
    world.pump(Vec::new());
    super::super::x11_input::press_control_a();
    for character in "20 mm".chars() {
        super::super::x11_input::press_path_character(character);
    }
    world.pump(Vec::new());
    click(&mut world, "OK");
    let view = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    assert_eq!(
        view.parameters.as_ref().unwrap()[0].cells.as_ref().unwrap()[1]
            .expression
            .as_deref()
            .map(String::as_str),
        Some("20 mm")
    );
    world.undo();
    world.redo();
    open(&mut world);
    world.expect_text("20 mm");
    click(&mut world, "Cancel");
    world.close();
}
