// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the selected tab in the Model dock for each open document.
use super::{ApplicationWorld, diagnostics, dock_menu_steps, workspace_layout};
use cucumber::{given, then, when};

#[given("the Model Browser and a second dockable panel share a tabbed dock at the left")]
fn model_and_log_share_left_dock(world: &mut ApplicationWorld) {
    super::browser_scenarios::model_browser_left(world);
    world.docking_node = diagnostics::window_dock_id("Model").filter(|id| *id != 0);
    assert!(world.docking_node.is_some(), "expected Model in a dock");
    dock_menu_steps::open_model_menu(world);
    dock_menu_steps::choose_log(world);
    shared_left_dock(world);
}

#[given("the user selects the second panel in `PartOne.opd`")]
fn select_log_in_part_one(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.other_document.expect("expected PartOne"));
    select_dock_tab(world, "Log");
}

#[given("the user selects the Model Browser in `docking-visible-solid-current.opd`")]
fn select_model_in_solid(world: &mut ApplicationWorld) {
    switch_to_solid(world);
    select_dock_tab(world, "Model");
}

#[given("`PartOne.opd` is active")]
fn activate_part_one(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.other_document.expect("expected PartOne"));
}

#[when("the user switches to `docking-visible-solid-current.opd`")]
fn switch_to_solid(world: &mut ApplicationWorld) {
    world.activate_through_tab(world.previous_document.expect("expected the solid Part"));
}

#[then("both panels remain available in the same left dock")]
fn shared_left_dock(world: &mut ApplicationWorld) {
    for panel in ["Model", "Log"] {
        assert_eq!(diagnostics::window_dock_id(panel), world.docking_node);
        assert!(diagnostics::window_dock_tab_bounds(panel).is_some());
    }
}

#[then("the Model Browser is active and shows the current document")]
fn model_is_active(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_tab_visible("Model"), Some(true));
    world.expect_browser_of(world.previous_document.expect("expected the solid Part"));
}

#[then("the viewport shows the current document")]
fn solid_part_shown(world: &mut ApplicationWorld) {
    workspace_layout::browser_feature::expect_solid_visible(world);
}

#[when("the user switches back to `PartOne.opd`")]
fn return_to_part_one(world: &mut ApplicationWorld) {
    activate_part_one(world);
}

#[then("the second panel is active again")]
fn log_is_active(_: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_tab_visible("Log"), Some(true));
    assert_eq!(diagnostics::window_dock_tab_visible("Model"), Some(false));
}

#[then("the viewport shows `PartOne.opd`")]
fn part_one_shown(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.other_document);
    super::super::viewport_background::expect_empty_viewport(world);
}

fn select_dock_tab(world: &mut ApplicationWorld, title: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(title).expect("expected dock tab");
    world.click(diagnostics::pointer_pixels(std::array::from_fn(|axis| {
        (min[axis] + max[axis]) * 0.5
    })));
    assert_eq!(diagnostics::window_dock_tab_visible(title), Some(true));
}

#[test]
fn document_tabs_restore_the_model_dock_selection_when_leaving_home() {
    let mut world = ApplicationWorld::new();
    super::retention_scenario::two_saved_documents(&mut world);
    model_and_log_share_left_dock(&mut world);
    select_log_in_part_one(&mut world);
    select_model_in_solid(&mut world);
    activate_part_one(&mut world);
    log_is_active(&mut world);

    open_home(&mut world);
    switch_to_solid(&mut world);
    model_is_active(&mut world);
    shared_left_dock(&mut world);

    open_home(&mut world);
    activate_part_one(&mut world);
    log_is_active(&mut world);
    shared_left_dock(&mut world);
    part_one_shown(&mut world);
    world.close();
}

fn open_home(world: &mut ApplicationWorld) {
    world.click(diagnostics::pointer_pixels(
        world.expect_tab("Home").center(),
    ));
    assert!(diagnostics::window_geometry("##home-workspace").is_some());
}
