// SPDX-License-Identifier: GPL-2.0-only
//! Create a feature in a single model-state member (#4119).
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, RepRef},
};

#[given(
    "The MCAD Software has an empty part with Primary, Capture State, and Copy State model states"
)]
fn empty_three_states(world: &mut ApplicationWorld) {
    if super::model_state_fixture::expect_empty_named_states(world) {
        return;
    }
    world.add_part();
    let client = Client::new(world.app().owner.document_host());
    for name in ["Capture State", "Copy State"] {
        client
            .model_states()
            .create(&CreateModelStateArgs {
                name: name.into(),
                ..Default::default()
            })
            .unwrap();
    }
    let states = client.model_states().list(&Default::default()).unwrap();
    let copy = states
        .model_states
        .unwrap()
        .into_iter()
        .find(|state| state.name == "Copy State")
        .unwrap();
    client
        .model_states()
        .activate(&RepRef { id: copy.id })
        .unwrap();
    world.pump(Vec::new());
}
#[given("Copy State is active and the edit scope is a single member")]
fn copy_single_scope(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let states = browser.model_states.unwrap();
    assert_eq!(states.active().unwrap().name, "Copy State");
    assert_eq!(
        states.edit_scope(),
        oblikovati_model::model_state::ModelStateEditScope::SingleMember
    );
}
#[when("the user creates a rectangular sketch and extrudes it as a solid")]
fn create_sketch_extrusion(world: &mut ApplicationWorld) {
    for label in ["Extrude", "XY Plane", "Rectangle"] {
        world.click_text(label);
    }
    for point in [[0., 0.], [1., 1.]] {
        super::super::x11_input::click(world.sketch_pixel_of(point));
        world.pump(Vec::new());
    }
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.click_text("Finish Sketch");
    world.expect_text("1 Profile");
    world.click_text("OK");
}
#[then("the extrusion is computed in Copy State")]
fn extrusion_computed(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.features.len(), 1);
    assert!(!browser.features[0].suppressed);
}
#[then("its model-state indicator lists the extrusion as suppressed in Primary and Capture State")]
fn other_members_suppressed(world: &mut ApplicationWorld) {
    let position = world.run("States").center();
    let (browser, size) = diagnostics::window_geometry("Model").unwrap();
    assert!(
        position[0] < browser[0] + size[0],
        "States {position:?} outside Model {browser:?} size {size:?}"
    );
    let point = diagnostics::pointer_pixels(position);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    world.expect_text("Primary: Suppressed");
    world.expect_text("Capture State: Suppressed");
    world.expect_text("Copy State: Computed");
}
#[when("the user shows the solid using All Visible on the active view representation")]
fn show_solid(world: &mut ApplicationWorld) {
    super::super::x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    super::part_views::expanded_primary_view(world);
    right_click_row(world, "Primary");
    choose_menu_row(world, "All Visible");
    world.pump(Vec::new());
}
#[when("activates Primary")]
fn activate_primary(world: &mut ApplicationWorld) {
    super::model_state_fixture::expand_state_branch(world);
    let view_y = super::part_views::view_branch(world).center()[1];
    let state_y = world.run("Model State").center()[1];
    let member = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text.trim() == "Primary" && run.center()[1] > state_y && run.center()[1] < view_y
        })
        .expect("Primary model-state row");
    let point = diagnostics::pointer_pixels(member.center());
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
}
#[when("the user reactivates Copy State")]
fn reactivate_copy(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(world.run("Copy State").center());
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
        "Copy State"
    );
}
#[then("the solid is absent and the extrusion row is dimmed and struck through")]
fn absent_and_suppressed(world: &mut ApplicationWorld) {
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 0);
    let name = world.state().browser.unwrap().features[0].name.clone();
    let run = world.run(&format!("{name} (suppressed)"));
    let regular = world.run("Model State");
    assert!(
        run.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(run.alpha)
            < regular
                .color
                .iter()
                .map(|value| u32::from(*value))
                .sum::<u32>()
                * u32::from(regular.alpha)
    );
    assert!(diagnostics::has_strike(run));
}
#[then("the solid is visible and the extrusion row is no longer struck through")]
fn visible_and_computed(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
    let name = world.state().browser.unwrap().features[0].name.clone();
    assert!(!diagnostics::has_strike(world.run(&name)));
}
