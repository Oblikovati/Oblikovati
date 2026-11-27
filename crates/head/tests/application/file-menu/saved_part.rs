// SPDX-License-Identifier: GPL-2.0-only
//! Prepare the saved Part and its Home recent entry for File menu inspection.

use super::super::home_recent_documents::{home_ellipsized_run, home_shows, open_home};
use super::*;
use std::path::Path;

#[given("`fixtures/cad/planar-sketch-empty-xy.opd` is open and saved in the MCAD Software")]
#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` has been opened in the MCAD Software and is listed in recent documents"
)]
fn named_saved_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/planar-sketch-empty-xy.opd");
    open_saved_menu_part(world, &path);
}

#[given("`fixtures/cad/workspace-layout-native.opd` is open and saved in the MCAD Software")]
#[given(
    "`fixtures/cad/workspace-layout-native.opd` has been opened in the MCAD Software and is listed in recent documents"
)]
fn current_named_saved_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/workspace-layout-native.opd");
    open_saved_menu_part(world, &path);
}

/// Open a saved Part through File Open, then check its sketch and recent entry.
pub(super) fn open_saved_menu_part(world: &mut ApplicationWorld, path: &Path) {
    world.open_fixture(path);
    let document = world.active_id();
    let state = world.state();
    let saved = &state.document_storage[&document];
    assert_eq!(saved.path.as_deref().map(Path::new), Some(path));
    assert!(!saved.unsaved_edits, "expected the saved Part");
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    let browser = state.browser.expect("expected the Part browser");
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1")
    );
    assert!(
        state
            .recent_documents
            .iter()
            .any(|entry| Path::new(&entry.path) == path)
    );
    world.saved_path = Some(path.to_owned());
    world.previous_document = Some(document);
}

#[given("Home is active with at least one recent document visible")]
fn home_with_recent_part(world: &mut ApplicationWorld) {
    open_home(world);
    expect_saved_menu_part_on_home(world);
}

/// Check Home displays the recent Part that the scenario opened.
pub(super) fn expect_saved_menu_part_on_home(world: &mut ApplicationWorld) {
    let state = world.state();
    let path = world
        .saved_path
        .as_ref()
        .expect("expected the saved Part path");
    let recent = state
        .recent_documents
        .iter()
        .find(|entry| Path::new(&entry.path) == path)
        .expect("expected the opened Part in recent documents");
    assert!(
        home_shows(world, &recent.name) || home_ellipsized_run(world, &recent.name).is_some(),
        "expected {:?} in Home, rendered runs: {:?}",
        recent.name,
        world.text()
    );
}
