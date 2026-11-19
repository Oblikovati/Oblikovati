// SPDX-License-Identifier: GPL-2.0-only
//! Native entry and accepted-scope behavior for the current sketch.
use super::super::{ApplicationWorld, diagnostics, ribbon_panel_expansion};
use cucumber::{given, then, when};
use oblikovati_api::types::{INFERENCE_SCOPE_ALL_GEOMETRY, INFERENCE_SCOPE_CURRENT_COMMAND};

const DIALOG: &str = "Constraint Inference Scope";
const EXPANSION: &str = "##ribbon-additional-Constrain";

#[given("an active planar sketch")]
fn active_sketch(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
}
#[when("the user open Constrain and choose Constraint Inference Scope")]
fn open_scope(world: &mut ApplicationWorld) {
    ribbon_panel_expansion::click_expansion_arrow(world, "Constrain");
    let point = world
        .run_in_window(EXPANSION, "ConstraintInferenceScope")
        .center();
    world.click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIALOG).is_some());
}
#[then("Geometry in current command, All Geometry and Select are available")]
fn scope_choices(world: &mut ApplicationWorld) {
    for label in ["Geometryincurrentcommand", "AllGeometry", "Select"] {
        assert_eq!(world.run_in_window(DIALOG, label).alpha, u8::MAX);
    }
}
#[then("Geometry in current command is selected and the geometry selector is disabled")]
fn current_and_disabled(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Geometry in current command"));
    selector_disabled(world);
}
#[given("Geometry in current command is selected in the active sketch")]
fn current_scope(world: &mut ApplicationWorld) {
    active_sketch(world);
    open_scope(world);
    current_and_disabled(world);
}
#[when("the user choose All Geometry")]
fn choose_all(world: &mut ApplicationWorld) {
    click_dialog(world, "AllGeometry");
}
#[then("the geometry selector is disabled")]
fn selector_disabled(world: &mut ApplicationWorld) {
    assert_ne!(
        world.run_in_window(DIALOG, "Geometry:0selected").alpha,
        u8::MAX
    );
}
#[when("the user accept and reopen Constraint Inference Scope")]
fn accept_and_reopen(world: &mut ApplicationWorld) {
    click_dialog(world, "OK");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIALOG).is_none());
    open_scope(world);
}
#[then("All Geometry remains selected")]
fn all_remains(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("All Geometry"));
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_ALL_GEOMETRY
    );
}
#[when("the user choose Geometry in current command and accept")]
fn restore_current(world: &mut ApplicationWorld) {
    click_dialog(world, "Geometryincurrentcommand");
    click_dialog(world, "OK");
    world.pump(Vec::new());
}
#[then("the original scope is restored in this session")]
fn scope_restored(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_CURRENT_COMMAND
    );
    assert!(diagnostics::window_geometry(DIALOG).is_none());
}
fn click_dialog(world: &mut ApplicationWorld, label: &str) {
    let point = world.run_in_window(DIALOG, label).center();
    world.click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
