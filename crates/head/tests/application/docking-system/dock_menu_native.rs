// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the dock-tab menu with native pointer input (#4133).
use super::super::x11_input;
use super::{ApplicationWorld, diagnostics, open_docking_fixture};

#[test]
fn native_dock_menu_opens_and_closes_log_beside_model() {
    let mut world = ApplicationWorld::new();
    open_docking_fixture(&mut world);
    let model = diagnostics::window_dock_id("Model").expect("Model dock");
    assert_ne!(model, 0);
    assert!(diagnostics::window_dock_tab_bounds("Log").is_none());
    open_model_menu(&mut world);
    for item in [
        "Log Panel",
        "Favorites",
        "Material Browser",
        "Appearance Browser",
    ] {
        world.expect_text(item);
    }
    click_text(&mut world, "Log Panel");
    wait_for_log_tab(&mut world);
    assert_eq!(diagnostics::window_dock_id("Log"), Some(model));
    click_text(&mut world, "Log");
    assert_eq!(diagnostics::window_dock_tab_visible("Log"), Some(true));
    world.save_capture("dock-menu-log-selected");
    close_tab(&mut world, "Log");
    assert!(diagnostics::window_dock_tab_bounds("Log").is_none());
    assert_eq!(diagnostics::window_dock_id("Model"), Some(model));
    assert_eq!(diagnostics::window_dock_tab_visible("Model"), Some(true));
    world.save_capture("dock-menu-log-closed");
    world.close();
}

fn open_model_menu(world: &mut ApplicationWorld) {
    let (min, max) = diagnostics::window_dock_tab_bounds("Model").expect("Model tab");
    let height = max[1] - min[1];
    let point = [max[0] + height, (min[1] + max[1]) * 0.5];
    x11_input::focus_window();
    x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}

fn click_text(world: &mut ApplicationWorld, label: &str) {
    let point = world.run(label).center();
    x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}

fn close_tab(world: &mut ApplicationWorld, label: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(label).expect("active dock tab");
    let point = [max[0] - (max[1] - min[1]) * 0.5, (min[1] + max[1]) * 0.5];
    x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}

fn wait_for_log_tab(world: &mut ApplicationWorld) {
    let deadline = std::time::Instant::now() + std::time::Duration::from_secs(2);
    while !world.shows("Log") && std::time::Instant::now() < deadline {
        std::thread::sleep(std::time::Duration::from_millis(20));
        world.pump(Vec::new());
    }
    world.expect_text("Log");
}
