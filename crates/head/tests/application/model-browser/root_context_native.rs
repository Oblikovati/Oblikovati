// SPDX-License-Identifier: GPL-2.0-only
//! Verify the browser root menu with native pointer input (#4116).
use super::*;

#[test]
fn native_root_menu_selects_and_clears_the_document() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    assert!(world.state().browser.unwrap().path.is_some());
    let root = diagnostics::pointer_pixels(root_caption(&mut world).center());
    super::super::super::x11_input::right_click(root);
    settle(&mut world);
    verify_root_menu(&mut world);
    world.save_capture("browser-root-context-selected");
    super::super::super::x11_input::press_escape();
    settle(&mut world);
    menu_closed_edges_stay(&mut world);
    super::super::super::x11_input::click([1300, 800]);
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
    for label in ["Open File Location", "Create Drawing View", "Open Drawing"] {
        assert!(
            world.run(label).alpha < u8::MAX,
            "{label}: expected unavailable"
        );
    }
    assert_eq!(world.run("Copy File Path").alpha, u8::MAX);
}
