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
