// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard navigation of project settings.
use super::super::{diagnostics, x11_input};
use super::*;

fn settle(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        world.pump(Vec::new());
    }
}

#[test]
fn native_keyboard_expands_and_collapses_folder_options() {
    let mut world = ApplicationWorld::new();
    open_projects_dialog(&mut world);
    let group = diagnostics::pointer_pixels(world.run("Folder Options").center());
    x11_input::click(group);
    settle(&mut world);
    x11_input::press_keysym(0xff53);
    settle(&mut world);
    folder_entries(&mut world);
    world.save_capture("project-dialog-native-expanded-3895");
    x11_input::press_keysym(0xff51);
    settle(&mut world);
    folder_entries_hidden(&mut world);
    expect_project_marked(&world);
    world.close();
}
