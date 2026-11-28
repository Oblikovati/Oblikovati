// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Representations auto-expansion preference through native input (#4138).
use super::*;

const AUTO_EXPAND: &str = "Auto expand Representations Folder on Open";

#[test]
fn preference_controls_new_document_without_changing_existing_tree_state() {
    let mut world = ApplicationWorld::new();
    let first = world.add_part();
    world.expect_text("Model State");
    set_auto_expand(&mut world, false);
    let second = world.add_part();
    world.expect_text("Representations");
    world.expect_no_text("Model State");
    set_auto_expand(&mut world, true);
    world.add_part();
    world.expect_text("Model State");
    select_part(&mut world, second);
    world.expect_no_text("Model State");
    select_part(&mut world, first);
    world.expect_text("Model State");
    world.close();
}

fn set_auto_expand(world: &mut ApplicationWorld, enabled: bool) {
    super::dock_search::click_model_menu(world);
    super::dockspace_restart::native_click_label(world, "Display Preferences");
    assert!(world.run_is_enabled(AUTO_EXPAND));
    if world.checked(AUTO_EXPAND) != enabled {
        super::dockspace_restart::native_click_label(world, AUTO_EXPAND);
    }
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
}

fn select_part(world: &mut ApplicationWorld, id: u64) {
    let name = world.document_name(id);
    let position = world.expect_tab(&name).center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
}
