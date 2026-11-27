// SPDX-License-Identifier: GPL-2.0-only
//! Native parameter column and cell context menus (#4119).
use super::model_state_table_extra::{click, grid_run, open, row_run};
use super::model_state_table_parameter_steps::three_state_prism;
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, ParameterSetArgs},
};

#[given(
    "The MCAD Software displays a four-member model-state table with angular parameter d1 included"
)]
fn angular_parameter_table(world: &mut ApplicationWorld) {
    three_state_prism(world);
    let client = Client::new(world.app().owner.document_host());
    client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Fourth State".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .add(&ParameterSetArgs {
            name: "d1".into(),
            expression: "0 deg".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    open(world);
    click(world, "Parameters");
    click(world, "d1");
    click(world, "Add >");
    let _ = grid_run(world, "d1");
}
#[when("the user right-clicks the d1 column header")]
fn column_menu(world: &mut ApplicationWorld) {
    let point = grid_run(world, "d1");
    context_menu_at(world, point);
}
#[then("the column context menu offers Delete Column")]
fn column_delete(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("Delete Column"));
}
#[then("Key, Custom Parameter Column, range and increment commands are disabled")]
fn disabled_parameter_column_commands(world: &mut ApplicationWorld) {
    for label in [
        "Key",
        "Custom Parameter Column",
        "Specify Range for Column",
        "Specify Increment for Column",
    ] {
        assert!(!world.run_is_enabled(label), "expected {label} disabled");
    }
}
#[then("file-name, appearance and material column commands are disabled")]
fn disabled_visual_column_commands(world: &mut ApplicationWorld) {
    for label in ["File Name Column", "Appearance / Material Column"] {
        assert!(!world.run_is_enabled(label), "expected {label} disabled");
    }
}
#[when("the user presses Escape and right-clicks a d1 member cell")]
fn cell_menu(world: &mut ApplicationWorld) {
    world.desktop_escape();
    world.pump(Vec::new());
    let point = [grid_run(world, "d1")[0], row_run(world, "Copy State")[1]];
    context_menu_at(world, point);
}
#[then("the cell context menu offers Insert Row, Delete Row, and Delete Column")]
fn cell_actions(world: &mut ApplicationWorld) {
    for label in ["Insert Row", "Delete Row", "Delete Column"] {
        assert!(world.run_is_enabled(label), "expected {label} enabled");
    }
}
#[then(
    "the default-row, compare-base-row, custom-parameter, range and increment commands are disabled"
)]
fn disabled_cell_commands(world: &mut ApplicationWorld) {
    for label in [
        "Set As Default / Compare Base Row",
        "Custom Parameter Column",
        "Column Range / Increment",
        "Custom Parameter Cell",
        "Cell Range / Increment",
    ] {
        assert!(!world.run_is_enabled(label), "expected {label} disabled");
    }
}
#[then("the context menu closes and the table remains open")]
fn menu_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Delete Column");
    world.expect_text("Member");
}

fn context_menu_at(world: &mut ApplicationWorld, point: [f32; 2]) {
    let target = diagnostics::pointer_pixels(point);
    world.hover(target);
    world.desktop_right_click(target);
    world.pump(Vec::new());
}

#[test]
fn native_parameter_menus_stage_row_and_column_actions() {
    let mut world = ApplicationWorld::new();
    angular_parameter_table(&mut world);
    column_menu(&mut world);
    column_delete(&mut world);
    cell_menu(&mut world);
    click(&mut world, "Insert Row");
    world.expect_text("Model State1");
    let inserted = [
        grid_run(&world, "d1")[0],
        row_run(&world, "Model State1")[1],
    ];
    context_menu_at(&mut world, inserted);
    click(&mut world, "Delete Row");
    world.expect_no_text("Model State1");
    let header = grid_run(&world, "d1");
    context_menu_at(&mut world, header);
    click(&mut world, "Delete Column");
    let member_y = world.run("Member").center()[1];
    assert!(
        !world
            .frame
            .runs
            .iter()
            .any(|run| { run.text.trim() == "d1" && (run.center()[1] - member_y).abs() < 3. }),
        "the deleted parameter column must leave the grid"
    );
    click(&mut world, "Cancel");
}

#[test]
fn native_parameter_cell_menu_opens_without_a_prior_header_menu() {
    let mut world = ApplicationWorld::new();
    angular_parameter_table(&mut world);
    let cell = [grid_run(&world, "d1")[0], row_run(&world, "Copy State")[1]];
    context_menu_at(&mut world, cell);
    assert!(world.run_is_enabled("Insert Row"));
    assert!(world.run_is_enabled("Delete Column"));
}
