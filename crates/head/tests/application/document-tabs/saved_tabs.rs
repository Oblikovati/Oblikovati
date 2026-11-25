// SPDX-License-Identifier: GPL-2.0-only
//! Open disposable copies of the published PartOne and PartTwo files.
use super::*;
use std::path::Path;

fn open_disposable_part(world: &mut ApplicationWorld, file: &str) -> u64 {
    let source = super::super::scenario_root()
        .join("fixtures/cad/quick-access-toolbar")
        .join(file);
    let working = world.scratch_folder().join(file);
    std::fs::copy(&source, &working).unwrap_or_else(|error| panic!("copy {source:?}: {error}"));
    world.open_fixture(&working);
    let id = world.active_id();
    let state = world.state();
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_PART)
    );
    let storage = &state.document_storage[&id];
    assert_eq!(
        storage.path.as_deref().map(Path::new),
        Some(working.as_path())
    );
    assert!(!storage.unsaved_edits);
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
