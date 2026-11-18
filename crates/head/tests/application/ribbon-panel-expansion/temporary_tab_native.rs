// SPDX-License-Identifier: GPL-2.0-only
//! Selected-tab behavior through native X11 pointer events (#3856).
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};

#[test]
fn selected_tab_keeps_then_closes_temporary_expansion() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    super::super::ribbon_display::click_display_menu(&mut world);
    world.click_text("Minimize to Tabs");
    world.click_text("3D Model");
    click_expansion_arrow(&mut world, "Modify");
    let tab = diagnostics::pointer_pixels(world.run("3D Model").center());
    super::super::x11_input::click(tab);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_some());
    super::super::x11_input::click(tab);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("##ribbon-additional-PartModify").is_none());
    assert!(diagnostics::window_geometry("##ribbon-temporary-panel").is_none());
    world.close();
}
