// SPDX-License-Identifier: GPL-2.0-only
//! Open disposable copies of the published tab fixtures.
use super::*;
use std::path::Path;

pub(super) fn open_disposable_tab(
    world: &mut ApplicationWorld,
    relative: &str,
    kind: oblikovati_api::types::DocumentType,
) -> u64 {
    let source = super::super::scenario_root().join(relative);
    let working = world.scratch_folder().join(source.file_name().unwrap());
    std::fs::copy(&source, &working).unwrap_or_else(|error| panic!("copy {source:?}: {error}"));
    world.open_fixture(&working);
    let id = world.active_id();
    let state = world.state();
    assert_eq!(state.document_type, Some(kind));
    let storage = &state.document_storage[&id];
    assert_eq!(
        storage.path.as_deref().map(Path::new),
        Some(working.as_path())
    );
    assert!(!storage.unsaved_edits);
    id
}

pub(super) fn open_disposable_part(world: &mut ApplicationWorld, file: &str) -> u64 {
    let fixture = format!("fixtures/cad/quick-access-toolbar/{file}");
    let id = open_disposable_tab(world, &fixture, oblikovati_api::types::DOCUMENT_PART);
    world.parts.push(id);
    world.previous_document = Some(id);
    id
}

#[given("`fixtures/cad/quick-access-toolbar/PartOne.opd` is open and active")]
#[given("`fixtures/cad/quick-access-toolbar/PartOne.opd` is open")]
fn part_one_open(world: &mut ApplicationWorld) {
    open_disposable_part(world, "PartOne.opd");
}

#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/quick-access-toolbar/PartTwo.opd` are open"
)]
#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/quick-access-toolbar/PartTwo.opd` are open in that order"
)]
#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/quick-access-toolbar/PartTwo.opd` are open as adjacent tabs in that order"
)]
#[given(
    "`fixtures/cad/quick-access-toolbar/PartOne.opd` and `fixtures/cad/quick-access-toolbar/PartTwo.opd` are open as neighboring tabs in that order"
)]
fn saved_tabs_open(world: &mut ApplicationWorld) {
    let first = open_disposable_part(world, "PartOne.opd");
    let second = open_disposable_part(world, "PartTwo.opd");
    assert!(
        world
            .state()
            .documents
            .iter()
            .map(|row| row.id)
            .eq([first, second])
    );
    let left = world.expect_tab("PartOne").center()[0];
    let right = world.expect_tab("PartTwo").center()[0];
    assert!(
        left < right,
        "PartOne at {left}: expected it left of PartTwo at {right}"
    );
}

#[given("PartOne is active")]
fn part_one_is_active(world: &mut ApplicationWorld) {
    let id = first(world);
    assert_eq!(world.document_name(id), "PartOne");
    world.activate_through_tab(id);
    world.previous_document = Some(id);
}

pub(super) fn edit_active_saved_sketch(world: &mut ApplicationWorld) {
    use super::super::workspace_layout::browser_feature::{choose_menu_row, right_click_row};
    right_click_row(world, "Sketch1");
    choose_menu_row(world, "Edit Sketch");
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
}
