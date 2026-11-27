// SPDX-License-Identifier: GPL-2.0-only
//! Tools clipboard availability and panel visibility.
use super::*;

#[test]
fn tools_clipboard_remains_unavailable_and_can_be_hidden() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    assert!(!world.run_is_enabled("Copy"));
    toggle_clipboard_panel(&mut world);
    world.expect_no_text("Copy");
    world.click_text("3D Model");
    world.click_text("Tools");
    world.expect_no_text("Copy");
    toggle_clipboard_panel(&mut world);
    assert!(!world.run_is_enabled("Copy"));
    world.close();
}

fn toggle_clipboard_panel(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(title);
    world.click_text("Show Panels");
    world.click_text("Clipboard");
}

#[test]
fn unavailable_copy_moves_and_remains_in_the_expanded_panel_after_restart() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    right_click_copy(&mut world);
    world.click_text("Move to Expanded Panel");
    world.expect_no_text("Copy");
    world.reopen_application();
    world.add_part();
    world.click_text("Tools");
    world.expect_no_text("Copy");
    world.click_text("Clipboard");
    assert!(!world.run_is_enabled("Copy"));
    right_click_copy(&mut world);
    world.click_text("Move to Main Panel");
    assert!(!world.run_is_enabled("Copy"));
    world.close();
}

fn right_click_copy(world: &mut ApplicationWorld) {
    let position = diagnostics::pointer_pixels(world.run("Copy").center());
    world.right_click(position);
}

#[test]
fn exported_builtin_location_is_staged_then_applied_on_import() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Tools");
    right_click_copy(&mut world);
    world.click_text("Move to Expanded Panel");
    let path = world.scratch_folder().join("builtin-ribbon.xml");
    export_builtin_location(&mut world, &path);
    world.click_text("Clipboard");
    right_click_copy(&mut world);
    world.click_text("Move to Main Panel");
    import_builtin_location(&mut world, &path);
    world.expect_no_text("Copy");
    world.click_text("Clipboard");
    assert!(!world.run_is_enabled("Copy"));
    world.close();
}

fn export_builtin_location(world: &mut ApplicationWorld, path: &std::path::Path) {
    open_tools_customization(world);
    click_in_customize(world, "Export");
    settings_file_choice(world, "Export Customization Settings", path, "Save");
    click_in_customize(world, "Close");
}

fn import_builtin_location(world: &mut ApplicationWorld, path: &std::path::Path) {
    open_tools_customization(world);
    click_in_customize(world, "Import");
    settings_file_choice(world, "Import Customization Settings", path, "Open");
    assert!(!world.run_is_enabled("Copy"));
    click_in_customize(world, "Apply");
    click_in_customize(world, "Close");
}
