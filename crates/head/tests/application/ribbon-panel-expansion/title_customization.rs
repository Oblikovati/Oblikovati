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
