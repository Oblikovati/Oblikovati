// SPDX-License-Identifier: GPL-2.0-only
//! Author the untapered extrusion required by the published parameter-table cases.
use super::model_state_fixture::has_named_states;
use super::model_state_new_feature::prepare_rectangle_extrusion;
use super::model_state_table::open_property_table;
use super::model_state_table_extra::click;
use super::model_state_table_parameter_steps::{
    accept_parameter_table, add_parameter_from_feature, given_length_parameter, grid_cell_text,
    set_parameter_cell,
};
use super::*;
use cucumber::given;
use oblikovati_api::client::Client;

#[given(
    "the user creates an undimensioned rectangular sketch and extrudes it 1 in with zero taper"
)]
fn author_parameter_fixture(world: &mut ApplicationWorld) {
    let part = world
        .previous_document
        .expect("expected the saved blank Part");
    world.expect_active(part);
    prepare_rectangle_extrusion(world);
    super::super::extrude_resume_native::set_extrusion_distance(world, "1 in");
    assert!(world.run_is_enabled("OK"));
    world.click_text("OK");
    world.pump(Vec::new());
    world.expect_active(part);
    given_length_parameter(world);
    expect_solid_visible(world);
}

#[given("the user opens the model-state table and adds length parameter d0 to the table")]
#[given("the user opens the model-state table and adds parameter d0")]
#[given("the user opens the model-state table with d0 included")]
fn add_saved_distance_column(world: &mut ApplicationWorld) {
    open_property_table(world);
    add_parameter_from_feature(world);
    accept_parameter_table(world);
    reopen_saved_distance_table(world);
}

#[given(
    "the user adds d0 to the model-state table and sets Copy State d0 to 2 in while leaving the other members at 1 in"
)]
fn set_saved_copy_distance(world: &mut ApplicationWorld) {
    open_property_table(world);
    add_parameter_from_feature(world);
    set_parameter_cell(world, "Copy State", "2 in");
    accept_parameter_table(world);
    reopen_saved_distance_table(world);
}

fn reopen_saved_distance_table(world: &mut ApplicationWorld) {
    open_property_table(world);
    click(world, "Parameters");
    expect_prepared_distance_table(world);
}

pub(super) fn expect_prepared_distance_table(world: &ApplicationWorld) {
    let primary = grid_cell_text(world, "d0", "Primary");
    assert_eq!(primary, "1in", "expected the original authored distance");
}

pub(super) fn expect_native_extrusion_centroid(world: &mut ApplicationWorld, expected_mm: f64) {
    if !has_named_states(world) {
        return;
    }
    let mass = Client::new(world.app().owner.document_host())
        .analysis()
        .mass_properties(&Default::default())
        .unwrap();
    assert!(mass.volume_mm3 > 0., "expected a solid extrusion: {mass:?}");
    assert!(
        (mass.centroid_z_mm - expected_mm).abs() < 1e-6,
        "expected the extrusion centroid at {expected_mm} mm, got {mass:?}"
    );
}
