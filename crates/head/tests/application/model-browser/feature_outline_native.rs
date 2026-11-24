// SPDX-License-Identifier: GPL-2.0-only
//! Check selected feature edges and their removal with X11 pointer input (#4110).
use super::*;

#[test]
fn x11_feature_outline_follows_browser_selection() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    let row = diagnostics::pointer_pixels(world.run(FEATURE).center());
    super::super::x11_input::click(row);
    settle(&mut world);
    assert!(world.state().browser.unwrap().selected_feature.is_some());
    expect_outline(&mut world, true);
    assert!(diagnostics::window_geometry("##selection-toolbar").is_some());
    world.save_capture("feature-outline-selected");
    let empty = super::super::viewport_entity_selection::empty_viewport_point(&mut world);
    super::super::x11_input::click(empty);
    settle(&mut world);
    assert!(world.state().browser.unwrap().selected_feature.is_none());
    expect_outline(&mut world, false);
    assert!(diagnostics::window_geometry("##selection-toolbar").is_none());
    expect_solid_visible(&mut world);
    world.save_capture("feature-outline-cleared");
    world.close();
}
