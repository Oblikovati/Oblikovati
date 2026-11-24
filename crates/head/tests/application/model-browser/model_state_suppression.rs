// SPDX-License-Identifier: GPL-2.0-only
//! Switch model states using the browser and inspect the native scene and feature row.
use super::model_state_fixture::{COPY_STATE, expand_state_branch, has_named_states};
use super::*;
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{CreateModelStateArgs, ModelStateTableSuppression, RepRef, SetFeatureSuppressedArgs},
};

#[given("The MCAD Software has a part with Primary and Copy State model states")]
fn states_with_extrusion(world: &mut ApplicationWorld) {
    if has_named_states(world) {
        current_state_extrusion(world);
        return;
    }
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
    if has_named_states(world) {
        expect_table_suppression(world, &[("Primary", true), (COPY_STATE, false)]);
        return;
    }
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
    if has_named_states(world) {
        assert!(world.checked("Copy State (Active)"));
        expect_solid_visible(world);
        return;
    }
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
    activate_state_row(world, "Primary");
}
#[given("the user activates Copy State")]
#[when("the user activates Copy State")]
fn activate_copy(world: &mut ApplicationWorld) {
    activate_state_row(world, COPY_STATE);
}

pub(super) fn activate_state_row(world: &mut ApplicationWorld, name: &str) {
    expand_state_branch(world);
    let checked = format!("{name} (Active)");
    if !world.shows(&checked) {
        world.double_click(diagnostics::pointer_pixels(world.run(name).center()));
    }
    assert!(world.checked(&checked));
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
    let status = world
        .app()
        .window
        .native
        .scene_status()
        .expect("expected a published scene");
    assert_eq!(
        status.visible_bodies, 0,
        "expected the suppressed part to publish no bodies"
    );
    assert!(!super::super::solid_visibility::visible_solid_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
}
#[then("the solid body is visible in the viewport")]
fn solid_visible(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
    let status = world
        .app()
        .window
        .native
        .scene_status()
        .expect("expected a published scene");
    assert_eq!(
        status.visible_bodies, 1,
        "expected the computed part to publish one body"
    );
}
#[then("the extrusion row is dimmed and struck through")]
fn suppressed_row(world: &mut ApplicationWorld) {
    let name = current_state_extrusion(world).name;
    let run = world.run(&format!("{name} (suppressed)"));
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

pub(super) fn current_state_extrusion(
    world: &mut ApplicationWorld,
) -> oblikovati_gui::GuiBrowserFeature {
    let browser = world.state().browser.unwrap();
    let [feature] = browser.features.as_slice() else {
        panic!("expected one extrusion, got {:?}", browser.features);
    };
    assert_eq!(feature.kind, "Extrusion");
    feature.clone()
}

pub(super) fn expect_table_suppression(world: &mut ApplicationWorld, expected: &[(&str, bool)]) {
    let feature = current_state_extrusion(world);
    let table = Client::new(world.app().owner.document_host())
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let rows = table.rows.unwrap();
    let column = table
        .suppressions
        .unwrap()
        .into_iter()
        .find(|column| column.feature_id == feature.id.0)
        .expect("expected the extrusion suppression column");
    for &(name, suppressed) in expected {
        let member = rows
            .iter()
            .find(|row| row.name == name)
            .unwrap_or_else(|| panic!("expected a model-state member named {name:?}"));
        assert_eq!(
            member_suppression(&column, member.ref_.member_id),
            suppressed,
            "member {name:?}"
        );
    }
}

fn member_suppression(column: &ModelStateTableSuppression, member: u64) -> bool {
    let cell = column
        .cells
        .as_deref()
        .unwrap()
        .iter()
        .find(|cell| cell.row.member_id == member)
        .expect("expected the member's suppression cell");
    cell.suppressed.as_deref().copied().unwrap_or(column.base)
}
#[then("the extrusion row is no longer struck through")]
fn computed_row(world: &mut ApplicationWorld) {
    let name = world.state().browser.unwrap().features[0].name.clone();
    let run = world.run(&name);
    assert!(
        !diagnostics::has_strike(run),
        "expected no line across the feature name"
    );
}

#[cfg(target_os = "linux")]
#[test]
fn native_state_switch_updates_scene_and_suppression_row() {
    let mut world = ApplicationWorld::new();
    states_with_extrusion(&mut world);
    set_primary_suppression(&mut world);
    activate_copy_fixture(&mut world);
    native_double_click(&mut world, "Primary");
    primary_marked(&mut world);
    no_solid(&mut world);
    suppressed_row(&mut world);
    native_double_click(&mut world, "Copy State");
    copy_marked(&mut world);
    solid_visible(&mut world);
    computed_row(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
fn native_double_click(world: &mut ApplicationWorld, label: &str) {
    let point = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(point);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
}
