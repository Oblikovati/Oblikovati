// SPDX-License-Identifier: GPL-2.0-only
//! Preserve a pinned expansion when its title opens customization (#3856).
use super::{ApplicationWorld, diagnostics, opened_unpinned, toggle_pin};

#[test]
fn pinned_expansion_survives_title_menu_customization() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    opened_unpinned(&mut world);
    toggle_pin(&mut world);
    let title = world
        .run_in_window("##ribbon-additional-PartModify", "Modify")
        .center();
    world.right_click(diagnostics::pointer_pixels(title));
    world.pump(Vec::new());
    world.click_text("Customize User Commands");
    assert!(diagnostics::window_geometry("Customize").is_some());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.click_text("Close");
    assert!(diagnostics::window_geometry("Customize").is_none());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_selecting_current_appearance_closes_unpinned_expansion() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    opened_unpinned(&mut world);
    select_normal_from_expansion(&mut world);
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    world.expect_text("Extrude");
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_selecting_current_appearance_keeps_pinned_expansion() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    opened_unpinned(&mut world);
    toggle_pin(&mut world);
    select_normal_from_expansion(&mut world);
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    world.close();
}

#[cfg(target_os = "linux")]
fn select_normal_from_expansion(world: &mut ApplicationWorld) {
    let title = world.run_in_window("##ribbon-additional-PartModify", "Modify");
    super::super::x11_input::right_click(diagnostics::pointer_pixels(title.center()));
    world.pump(Vec::new());
    for label in ["Ribbon Appearance", "Normal"] {
        let position = diagnostics::pointer_pixels(world.run(label).center());
        super::super::x11_input::click(position);
        world.pump(Vec::new());
    }
}
