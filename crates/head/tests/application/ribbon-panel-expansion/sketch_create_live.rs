// SPDX-License-Identifier: GPL-2.0-only
//! Sketch expansion fixtures and checks for the ribbon benchmark (#3856).
use super::super::workspace_layout::browser_feature::{
    SKETCH, open_visible_extruded_part, right_click_row,
};
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};
use cucumber::{given, then, when};

const CREATE: &str = "##ribbon-additional-Create";
const CONSTRAIN: &str = "##ribbon-additional-Constrain";

fn wide_sketch(world: &mut ApplicationWorld) {
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    world.record_initial_sketch();
}
#[given(
    "a part containing an existing sketch is active and the sketch is being edited in the MCAD Software"
)]
fn existing_sketch_under_edit(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
    wide_sketch(world);
}
#[given("the Sketch ribbon is expanded and the Create panel is visible")]
#[given("the Sketch ribbon is expanded and the Constrain panel is visible")]
fn full_sketch(world: &mut ApplicationWorld) {
    world.expect_text("Create");
    world.expect_text("Constrain");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the Create panel's additional command area is closed")]
fn create_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_none());
}
#[given("the Constrain panel's additional command area is closed")]
fn constrain_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}
#[when("the user selects the expansion arrow on the Create panel title")]
fn open_create(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Create");
}
#[when("the user opens the Constrain panel expansion")]
fn open_constrain(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Constrain");
}
#[then("an additional command area opens below that panel over the workspace")]
fn create_below_ribbon(_: &mut ApplicationWorld) {
    let (panel, _) = diagnostics::window_geometry(CREATE).expect("Create expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
}
#[then("an additional command area opens below the panel over the workspace")]
fn constrain_below_ribbon(_: &mut ApplicationWorld) {
    let (panel, _) = diagnostics::window_geometry(CONSTRAIN).expect("Constrain expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(panel[1] >= ribbon[1] + size[1]);
}
#[then("the Create Block command is visible")]
fn create_block_visible(world: &mut ApplicationWorld) {
    world.run_in_window(CREATE, "CreateBlock");
}
#[then("Precise Input is visible but unavailable in this captured state")]
fn precise_unavailable(world: &mut ApplicationWorld) {
    assert!(world.run_in_window(CREATE, "PreciseInput").alpha < u8::MAX);
}
#[then("the sketch and model browser remain visible")]
fn sketch_and_browser(world: &mut ApplicationWorld) {
    assert!(world.state().active_sketch.is_some());
    assert!(diagnostics::window_geometry("Model").is_some());
}
#[then("the Create additional command area remains open")]
fn create_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_some());
}
#[then("the Create additional command area closes")]
fn create_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CREATE).is_none());
}
#[then("the Constrain additional command area closes")]
fn constrain_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}
