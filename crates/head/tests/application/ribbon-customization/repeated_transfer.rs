// SPDX-License-Identifier: GPL-2.0-only
//! Native selection checks for consecutive Add clicks in Ribbon Customization.
use super::list_keyboard_steps::{capture_plain_source_rows, expect_source_selection};
use super::transaction_steps::target_rows;
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
