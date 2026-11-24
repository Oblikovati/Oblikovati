// SPDX-License-Identifier: GPL-2.0-only
//! Prepare per-member suppression in the native table of the saved solid Part.
use super::model_state_fixture::{CAPTURE_STATE, COPY_STATE};
use super::model_state_suppression::{current_state_extrusion, expect_table_suppression};
use super::model_state_table::{accept_table, open_property_table};
use super::model_state_table_extra::click;
use super::model_state_table_suppression_steps::select_suppression_cell;
use super::*;
use cucumber::given;

#[given(
    "the user sets the extrusion to Suppress in Primary and Capture State and Compute in Copy State through the model-state table"
)]
fn set_saved_extrusion_suppression(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    open_property_table(world);
    click(world, "Suppression");
    click(world, &name);
    click(world, "Add >");
    for row in ["Primary", CAPTURE_STATE] {
        select_suppression_cell(world, row, "Suppress");
    }
    accept_table(world);
    expect_saved_suppression(world);
}

pub(super) fn expect_saved_suppression(world: &mut ApplicationWorld) {
    expect_table_suppression(
        world,
        &[
            ("Primary", true),
            (CAPTURE_STATE, true),
            (COPY_STATE, false),
        ],
    );
}
