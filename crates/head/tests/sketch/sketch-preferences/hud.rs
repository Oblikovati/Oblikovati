// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for HUD input dependencies and cancellation.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[test]
fn escape_closes_the_hud_before_application_options() {
    let mut world = ApplicationWorld::new();
    pointer_enabled(&mut world);
    world.escape();
    assert!(super::super::diagnostics::window_geometry("HUD Settings").is_none());
    assert!(super::super::diagnostics::window_geometry("Application Options").is_some());
    world.escape();
    both_closed(&mut world);
    world.close();
}

#[test]
fn accepting_hud_changes_waits_for_application_apply() {
    let mut world = ApplicationWorld::new();
    pointer_enabled(&mut world);
    world.click_text("Enable Pointer Input");
    world.click_text("OK");
    assert!(world.state().sketch_options.unwrap().pointer_input);
    world.click_text("Apply");
    assert!(!world.state().sketch_options.unwrap().pointer_input);
    world.click_text("HUD Settings");
    assert!(!world.checked("Enable Pointer Input"));
    world.close();
}

#[given("Pointer Input is enabled with Cartesian Coordinates")]
fn pointer_enabled(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.open_application_options();
    world.click_text("HUD Settings");
    assert!(world.checked("Enable Pointer Input"));
    assert!(world.radio_selected("Pointer Cartesian Coordinates"));
}
#[given("Dimension Input is enabled with Polar Coordinates")]
fn dimension_enabled(world: &mut ApplicationWorld) {
    assert!(world.checked("Enable Dimension Input"));
    assert!(world.radio_selected("Dimension Polar Coordinates"));
}
#[when("the user unchecks Enable Pointer Input")]
fn disable_pointer(world: &mut ApplicationWorld) {
    world.click_text("Enable Pointer Input");
}
#[then("its coordinate radio controls are disabled")]
fn pointer_modes_disabled(world: &mut ApplicationWorld) {
    assert_modes(world, "Pointer", false);
}
#[then("the dimension coordinate controls remain active")]
fn dimension_modes_enabled(world: &mut ApplicationWorld) {
    assert_modes(world, "Dimension", true);
}
#[when("the user unchecks Enable Dimension Input")]
fn disable_dimension(world: &mut ApplicationWorld) {
    world.click_text("Enable Dimension Input");
}
#[then("its coordinate radio controls are disabled too")]
fn dimension_modes_disabled(world: &mut ApplicationWorld) {
    assert_modes(world, "Dimension", false);
}
fn assert_modes(world: &ApplicationWorld, name: &str, enabled: bool) {
    for mode in ["Cartesian", "Polar"] {
        let label = format!("{name} {mode} Coordinates");
        let alpha = world.run(&label).alpha;
        assert_eq!(alpha == u8::MAX, enabled, "{label}");
    }
}
#[when("the user clicks Cancel and reopens HUD Settings")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.click_text("HUD Settings");
}
#[then("both input checkboxes are checked again")]
fn inputs_restored(world: &mut ApplicationWorld) {
    assert!(world.checked("Enable Pointer Input"));
    assert!(world.checked("Enable Dimension Input"));
}
#[then("Pointer Cartesian and Dimension Polar selections are restored")]
fn modes_restored(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Pointer Cartesian Coordinates"));
    assert!(world.radio_selected("Dimension Polar Coordinates"));
}
#[then("HUD Settings and Application Options close in sequence")]
fn both_closed(_world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry("HUD Settings").is_none());
    assert!(super::super::diagnostics::window_geometry("Application Options").is_none());
}
