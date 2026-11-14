// SPDX-License-Identifier: GPL-2.0-only
//! Check the quick access redo control against the active document history.
use super::*;

#[given("Redo is present on the quick access toolbar")]
fn redo_present(world: &mut ApplicationWorld) {
    assert!(on_row(world, "Redo"), "expected Redo on the toolbar");
}

#[given("no redo action is available in the active document")]
fn no_redo_available(world: &mut ApplicationWorld) {
    assert_eq!(world.state().redo_label, None);
}

#[when("the user inspects the quick access toolbar")]
fn inspect_toolbar(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
}

#[then("Redo remains visible but disabled")]
fn redo_is_disabled(world: &mut ApplicationWorld) {
    redo_present(world);
    assert!(!world.run_is_enabled("Redo"), "expected Redo disabled");
}

#[then("the active part remains open")]
fn active_part_remains(world: &mut ApplicationWorld) {
    part_stays(world);
}
