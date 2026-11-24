// SPDX-License-Identifier: GPL-2.0-only
//! Author the untapered extrusion required by the published parameter-table cases.
use super::model_state_fixture::has_named_states;
use super::model_state_new_feature::prepare_rectangle_extrusion;
use super::model_state_table_parameter_steps::given_length_parameter;
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

pub(super) fn expect_native_extrusion_centroid(world: &mut ApplicationWorld, expected_mm: f64) {
    if world.previous_document.is_none() || !has_named_states(world) {
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
