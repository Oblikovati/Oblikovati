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
    let icon = super::super::quick_access_icons::standard_icon("Redo").unwrap();
    assert!(icon.alpha < 255, "expected a disabled Redo icon");
}

#[then("the active part remains open")]
fn active_part_remains(world: &mut ApplicationWorld) {
    part_stays(world);
}

#[test]
fn undo_icon_keeps_its_history_menu() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    world.right_click(row_center(&world, "Undo"));
    world.pump(Vec::new());
    world.expect_text("History Browser");
    world.close();
}
