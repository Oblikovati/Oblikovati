// SPDX-License-Identifier: GPL-2.0-only
//! Open and close workspace panels through the dock-tab menu (#4133).
use super::super::x11_input;
use super::{ApplicationWorld, diagnostics, open_docking_fixture};
use cucumber::{given, then, when};

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open in the MCAD Software")]
fn open_native_part(world: &mut ApplicationWorld) {
    open_docking_fixture(world);
}

#[given("Model is docked and Log is closed")]
fn model_docked_log_closed(world: &mut ApplicationWorld) {
    let model = diagnostics::window_dock_id("Model").expect("Model dock");
    assert_ne!(model, 0);
    assert!(diagnostics::window_dock_tab_bounds("Log").is_none());
    world.docking_node = Some(model);
}

#[when("the user clicks the plus button in the Model dock tab bar")]
fn open_model_menu(world: &mut ApplicationWorld) {
    let (min, max) = diagnostics::window_dock_tab_bounds("Model").expect("Model tab");
    let height = max[1] - min[1];
    let point = [max[0] + height, (min[1] + max[1]) * 0.5];
    if !world.app().window.native.view.uses_wayland() {
        x11_input::focus_window();
    }
    click_dock_control(world, point);
}

#[then("the panel menu offers Log Panel, Favorites, Material Browser and Appearance Browser")]
fn panel_menu_offers_windows(world: &mut ApplicationWorld) {
    for item in [
        "Log Panel",
        "Favorites",
        "Material Browser",
        "Appearance Browser",
    ] {
        world.expect_text(item);
    }
}

#[when("the user chooses Log Panel from that menu")]
fn choose_log(world: &mut ApplicationWorld) {
    click_text(world, "Log Panel");
    wait_for_log_tab(world);
}

#[then("Log is available beside Model in the same dock tab bar")]
fn log_beside_model(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_id("Log"), world.docking_node);
    assert!(world.shows("Log"));
}

#[when("the user selects Log and closes its dock tab")]
fn select_and_close_log(world: &mut ApplicationWorld) {
    click_text(world, "Log");
    assert_eq!(diagnostics::window_dock_tab_visible("Log"), Some(true));
    world.save_capture("dock-menu-log-selected");
    close_tab(world, "Log");
}

#[then("Log is removed from the tab bar")]
fn log_removed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_dock_tab_bounds("Log").is_none());
}

#[then("Model is still available in the same dock region")]
fn model_remains(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_id("Model"), world.docking_node);
    assert_eq!(diagnostics::window_dock_tab_visible("Model"), Some(true));
    world.save_capture("dock-menu-log-closed");
}

fn click_text(world: &mut ApplicationWorld, label: &str) {
    let point = world.run(label).center();
    click_dock_control(world, point);
}

fn close_tab(world: &mut ApplicationWorld, label: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(label).expect("active dock tab");
    let point = [max[0] - (max[1] - min[1]) * 0.5, (min[1] + max[1]) * 0.5];
    click_dock_control(world, point);
}

fn click_dock_control(world: &mut ApplicationWorld, point: [f32; 2]) {
    let position = diagnostics::pointer_pixels(point);
    if world.app().window.native.view.uses_wayland() {
        world.click(position);
        return;
    }
    x11_input::click(position);
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

#[test]
fn native_dock_menu_opens_and_closes_log_beside_model() {
    let mut world = ApplicationWorld::new();
    world.restore_layout_without_asset_tools();
    super::fixture_native::reopen_bundled_docking_part(&mut world);
    model_docked_log_closed(&mut world);
    open_model_menu(&mut world);
    panel_menu_offers_windows(&mut world);
    choose_log(&mut world);
    log_beside_model(&mut world);
    select_and_close_log(&mut world);
    log_removed(&mut world);
    model_remains(&mut world);
    world.close();
}
