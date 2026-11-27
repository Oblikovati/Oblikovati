// SPDX-License-Identifier: GPL-2.0-only
//! Verify the browser root menu with native pointer input (#4116).
use super::*;

#[test]
fn native_root_menu_selects_and_clears_the_document() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    assert!(world.state().browser.unwrap().path.is_some());
    let root = diagnostics::pointer_pixels(root_caption(&mut world).center());
    world.desktop_right_click(root);
    settle(&mut world);
    verify_root_menu(&mut world);
    world.save_capture("browser-root-context-selected");
    world.desktop_escape();
    settle(&mut world);
    menu_closed_edges_stay(&mut world);
    let empty = super::super::super::viewport_entity_selection::empty_viewport_point(&mut world);
    world.desktop_click(empty);
    settle(&mut world);
    assert!(!world.state().browser.unwrap().root_selected);
    expect_nothing_chosen(&mut world);
    world.save_capture("browser-root-context-cleared");
    world.close();
}

fn verify_root_menu(world: &mut ApplicationWorld) {
    root_lit(world);
    file_rows(world);
    drawing_rows(world);
    root_rows(world);
    for label in ["Create Drawing View", "Open Drawing"] {
        assert!(
            world.run(label).alpha < u8::MAX,
            "{label}: expected unavailable"
        );
    }
    assert!(world.run_is_enabled("Open File Location"));
    assert_eq!(world.run("Copy File Path").alpha, u8::MAX);
}

#[test]
fn saved_root_opens_its_containing_folder() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    let document = world.active_id();
    let saved = world.state().document_storage[&document]
        .path
        .clone()
        .unwrap();
    let expected = std::path::Path::new(&saved).parent().unwrap().to_owned();
    let root = diagnostics::pointer_pixels(root_caption(&mut world).center());
    world.desktop_right_click(root);
    settle(&mut world);
    assert!(world.run_is_enabled("Open File Location"));
    let action = diagnostics::pointer_pixels(world.run("Open File Location").center());
    world.desktop_click(action);
    settle(&mut world);
    assert_eq!(world.app().window.opened_directories, [expected]);
    world.expect_active(document);
    world.close();
}

#[test]
fn unsaved_root_disables_open_file_location() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let root = diagnostics::pointer_pixels(root_caption(&mut world).center());
    world.desktop_right_click(root);
    settle(&mut world);
    assert!(!world.run_is_enabled("Open File Location"));
    let action = diagnostics::pointer_pixels(world.run("Open File Location").center());
    world.desktop_click(action);
    settle(&mut world);
    assert!(world.app().window.opened_directories.is_empty());
    world.close();
}
