// SPDX-License-Identifier: GPL-2.0-only
//! Native selection checks for consecutive Add and Remove clicks in Ribbon Customization.
use super::list_keyboard_steps::{
    capture_plain_source_rows, expect_source_selection, expect_target_selection,
};
use super::list_steps::source_rows;
use super::transaction_steps::{add_command, target_rows};
use super::*;

const FIRST_COMMANDS: [&str; 3] = ["2DSketches", "3DAnnotations", "3DPDF"];

#[when("the user clicks the right transfer button")]
#[when("the user clicks the add command button")]
pub(super) fn transfer_to_target(world: &mut ApplicationWorld) {
    click_in_customize(world, ">");
}

#[given("2D Sketches is selected in the available commands list")]
fn select_sketches(world: &mut ApplicationWorld) {
    capture_plain_source_rows(world);
    click_in_customize(world, FIRST_COMMANDS[0]);
    expect_source_selection(world, Some(FIRST_COMMANDS[0]), &FIRST_COMMANDS[..1]);
}

#[when("the user clicks the add command button twice")]
fn add_twice(world: &mut ApplicationWorld) {
    for _ in 0..2 {
        transfer_to_target(world);
    }
}

#[then("the target list contains 2D Sketches followed by 3D Annotations")]
fn first_two_staged(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), &FIRST_COMMANDS[..2]);
}

#[then("3D PDF is highlighted in the available commands list")]
fn pdf_highlighted(world: &mut ApplicationWorld) {
    expect_source_selection(world, Some(FIRST_COMMANDS[2]), &FIRST_COMMANDS[2..]);
}

#[when("the user clicks the add command button once more")]
fn add_again(world: &mut ApplicationWorld) {
    transfer_to_target(world);
}

#[then("the target list contains 2D Sketches, 3D Annotations and 3D PDF in that order")]
fn first_three_staged(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), FIRST_COMMANDS);
}

#[then("the next visible 3D PDF export entry is highlighted in the available commands list")]
fn pdf_export_highlighted(world: &mut ApplicationWorld) {
    expect_source_selection(world, Some("3DPDFExport"), &["3DPDFExport"]);
}

#[given("2D Sketches, 3D Annotations and 3D PDF have been staged in that order")]
fn stage_all_three(world: &mut ApplicationWorld) {
    for name in FIRST_COMMANDS {
        add_command(world, name);
    }
    first_three_staged(world);
}

#[given("the 2D Sketches target row is selected")]
fn select_first_staged(world: &mut ApplicationWorld) {
    click_in_customize(world, FIRST_COMMANDS[0]);
    expect_target_selection(world, FIRST_COMMANDS[0]);
}

#[when("the user clicks the remove command button again")]
fn remove_again(world: &mut ApplicationWorld) {
    super::transfer_to_source(world);
}

#[then("2D Sketches returns to the available commands list")]
fn sketches_returned(world: &mut ApplicationWorld) {
    expect_source_command(world, FIRST_COMMANDS[0]);
}

#[then("3D Annotations returns to the available commands list")]
fn annotations_returned(world: &mut ApplicationWorld) {
    expect_source_command(world, FIRST_COMMANDS[1]);
}

#[then("3D PDF returns to the available commands list")]
fn pdf_returned(world: &mut ApplicationWorld) {
    expect_source_command(world, FIRST_COMMANDS[2]);
}

fn expect_source_command(world: &ApplicationWorld, command: &str) {
    assert!(
        source_rows(world).iter().any(|row| row.text == command),
        "expected {command} to return to the available list"
    );
}

#[then("3D Annotations is highlighted in the target list above 3D PDF")]
fn second_staged_selected(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), &FIRST_COMMANDS[1..]);
    expect_target_selection(world, FIRST_COMMANDS[1]);
}

#[then("3D PDF is highlighted as the only target row")]
fn last_staged_selected(world: &mut ApplicationWorld) {
    assert_eq!(target_rows(world), &FIRST_COMMANDS[2..]);
    expect_target_selection(world, FIRST_COMMANDS[2]);
}
