// SPDX-License-Identifier: GPL-2.0-only
//! Check Home preview capture through real native GUI frames.
use super::*;

#[test]
fn home_tab_capture_survives_hover_during_activation() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let home = world.expect_tab("Home").center().map(|value| value as i32);
    world.click(home);
    world.pump(Vec::new());
    world.save_capture("home-tab-activated");
    world.click_tab(part);
    world.pump(Vec::new());
    world.hover(home);
    world.pump(Vec::new());
    super::home_and_previews::expect_preview_above(&mut world, "Home");
    world.save_capture("home-tab-clean-preview");
    world.expect_active(part);
    world.hover([900, 500]);
    assert!(window_geometry("##tab-preview").is_none());
    world.close();
}
