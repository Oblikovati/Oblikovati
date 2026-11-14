// SPDX-License-Identifier: GPL-2.0-only
//! Switch model states using the browser and inspect the native scene and feature row.
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef, SetFeatureSuppressedArgs},
};

#[given("The MCAD Software has a part with Primary and Copy State model states")]
fn states_with_extrusion(world: &mut ApplicationWorld) {
    open_extruded_part(world);
    let client = Client::new(world.app().owner.document_host());
    client
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Copy State".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}
#[given("the extrusion is suppressed in Primary and computed in Copy State")]
fn set_primary_suppression(world: &mut ApplicationWorld) {
    let id = world.state().browser.unwrap().features[0].id.0;
    Client::new(world.app().owner.document_host())
        .features()
        .set_suppressed(&SetFeatureSuppressedArgs {
            id,
            suppressed: true,
        })
        .unwrap();
    world.pump(Vec::new());
}
#[given("Copy State is active and its solid body is visible")]
fn activate_copy_fixture(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let states = client
        .model_states()
        .list(&Default::default())
        .unwrap()
        .model_states
        .unwrap();
    let id = states
        .iter()
        .find(|state| state.name == "Copy State")
        .unwrap()
        .id;
    client.model_states().activate(&RepRef { id }).unwrap();
    world.pump(Vec::new());
    world.click_text("Model State");
    expect_solid_visible(world);
}
#[when("the user activates the Primary model state")]
fn activate_primary(world: &mut ApplicationWorld) {
    world.double_click(diagnostics::pointer_pixels(world.run("Primary").center()));
}
#[when("the user activates Copy State")]
fn activate_copy(world: &mut ApplicationWorld) {
    world.double_click(diagnostics::pointer_pixels(
        world.run("Copy State").center(),
    ));
}
#[then("the Primary model state is marked active")]
fn primary_marked(world: &mut ApplicationWorld) {
    world.expect_text("Primary (Active)");
    assert!(world.checked("Primary (Active)"));
}
#[then("Copy State is marked active")]
fn copy_marked(world: &mut ApplicationWorld) {
    world.expect_text("Copy State (Active)");
    assert!(world.checked("Copy State (Active)"));
}
#[then("the solid body is absent from the viewport")]
fn no_solid(world: &mut ApplicationWorld) {
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(!super::super::solid_visibility::visible_solid_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}
#[then("the solid body is visible in the viewport")]
fn solid_visible(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}
#[then("the extrusion row is dimmed and struck through")]
fn suppressed_row(world: &mut ApplicationWorld) {
    let run = world.run("Extrude1 (suppressed)");
    let regular = world.run("Model State");
    assert!(
        run.color.iter().map(|v| u32::from(*v)).sum::<u32>() * u32::from(run.alpha)
            < regular.color.iter().map(|v| u32::from(*v)).sum::<u32>() * u32::from(regular.alpha),
        "suppressed {run:?}, regular {regular:?}"
    );
    assert!(
        diagnostics::has_strike(run),
        "expected a line across the feature name"
    );
}
#[then("the extrusion row is no longer struck through")]
fn computed_row(world: &mut ApplicationWorld) {
    let run = world.run("Extrude1");
    assert!(
        !diagnostics::has_strike(run),
        "expected no line across the feature name"
    );
}
