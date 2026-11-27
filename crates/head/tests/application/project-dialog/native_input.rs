// SPDX-License-Identifier: GPL-2.0-only
//! Native keyboard navigation of project settings.
use super::super::{diagnostics, x11_input};
use super::settings_tree::{folder_entries, folder_entries_hidden};
use super::*;
use oblikovati_viewport::ViewportPhysicalKey;

/// Send a key through the native window. Example: `press_project_key(ViewportPhysicalKey::N, true)`.
pub(in super::super) fn press_project_key(key: ViewportPhysicalKey, control: bool) {
    match (key, control) {
        (ViewportPhysicalKey::N, true) => x11_input::press_control_letter(b'n'),
        (ViewportPhysicalKey::Enter, false) => x11_input::press_keysym(0xff0d),
        (ViewportPhysicalKey::LeftArrow, false) => x11_input::press_keysym(0xff51),
        (ViewportPhysicalKey::RightArrow, false) => x11_input::press_keysym(0xff53),
        pair => panic!("project key {pair:?}: expected Ctrl+N, Enter, Left Arrow, or Right Arrow"),
    }
}

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
