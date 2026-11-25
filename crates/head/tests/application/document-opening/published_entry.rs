// SPDX-License-Identifier: GPL-2.0-only
//! Open the published assembly from Home and preserve the active model on cancellation.
use super::{ApplicationWorld, choose_open, look_in, open_dialog, select_row, settle};
use cucumber::{given, then, when};
use oblikovati_gui::GuiBrowserKind;
use std::path::PathBuf;

const ASSEMBLY: &str = "single-occurrence.oad";
const HOME: &str = "##home-workspace";
const PART: &str = "extrusion/extrusion-rect-8mm.opd";

pub(super) fn fixture(file: &str) -> PathBuf {
    let path = super::super::scenario_root()
        .join("fixtures/cad")
        .join(file);
    assert!(path.is_file(), "expected published fixture at {path:?}");
    path
}

#[given("Home is active with no model document open in the MCAD Software")]
fn home_is_active(world: &mut ApplicationWorld) {
    world.run_in_window(HOME, "Open");
    let state = world.state();
    assert!(state.documents.is_empty());
    assert!(state.document.is_none());
}

#[given("a saved document fixture is available")]
fn saved_fixture_available(world: &mut ApplicationWorld) {
    world.saved_path = Some(fixture(ASSEMBLY));
}

#[given("`fixtures/cad/single-occurrence.oad` and its referenced native component are available")]
fn assembly_with_component(world: &mut ApplicationWorld) {
    assert_eq!(world.saved_path.as_ref(), Some(&fixture(ASSEMBLY)));
    fixture(PART);
}

#[when("the user opens the saved fixture through the Open dialog")]
fn open_from_home(world: &mut ApplicationWorld) {
    let position = world.run_in_window(HOME, "Open").center();
    world.click(position.map(|coordinate| coordinate as i32));
    settle(world);
    world.expect_text("Look in");
    let path = world.saved_path.clone().unwrap();
    look_in(world, path.parent().unwrap());
    select_row(world, ASSEMBLY);
    choose_open(world);
}

#[then("the fixture document becomes active in a document tab")]
fn assembly_tab_active(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_tab(&name);
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        world.saved_path.as_ref().and_then(|path| path.to_str())
    );
}

#[then("its document workspace is displayed")]
fn assembly_workspace(world: &mut ApplicationWorld) {
    world.expect_text("Assemble");
    world.run_in_window("Model", "LayoutComponent:1");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.kind, GuiBrowserKind::Assembly);
    let [occurrence] = browser.occurrences.as_slice() else {
        panic!("expected one loaded occurrence in the published assembly");
    };
    assert!(!occurrence.deferred && !occurrence.suppressed && occurrence.error.is_none());
    let [_, _, width, height] = world.viewport_bounds();
    assert!(width > 0 && height > 0);
}

#[then("Home remains available")]
fn home_remains_available(world: &mut ApplicationWorld) {
    world.expect_tab("Home");
}

#[given("a model document is active in the MCAD Software")]
fn model_active(world: &mut ApplicationWorld) {
    world.open_fixture(&fixture(PART));
    world.previous_document = Some(world.active_id());
}

#[given("`fixtures/cad/extrusion/extrusion-rect-8mm.opd` is the active document")]
fn exact_part_active(world: &mut ApplicationWorld) {
    let id = world.active_id();
    assert_eq!(
        world.state().document_storage[&id].path.as_deref(),
        fixture(PART).to_str()
    );
}

#[when("the user opens the Open dialog")]
fn open_while_model_active(world: &mut ApplicationWorld) {
    open_dialog(world);
}

#[then("the same model document remains active")]
fn original_model_active(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    world.expect_active(world.previous_document.unwrap());
    let id = world.active_id();
    let name = world.document_name(id);
    world.expect_tab(&name);
    assert_eq!(world.state().documents.len(), 1);
    exact_part_active(world);
}
