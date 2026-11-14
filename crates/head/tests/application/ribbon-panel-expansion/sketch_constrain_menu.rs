// SPDX-License-Identifier: GPL-2.0-only
//! Native title-menu interactions while editing a sketch.
use super::{ApplicationWorld, diagnostics, sketch_constrain};
use cucumber::{given, then, when};

const CONSTRAIN: &str = "##ribbon-additional-Constrain";

#[given("the full Sketch ribbon is visible")]
#[then("the full Sketch ribbon is visible")]
fn full_sketch_ribbon(world: &mut ApplicationWorld) {
    world.expect_text("Constrain");
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[given("the Constrain additional command area is open")]
fn constrain_area_open(world: &mut ApplicationWorld) {
    sketch_constrain::open_constrain(world);
    assert!(diagnostics::window_geometry(CONSTRAIN).is_some());
}
#[when("the user right-clicks the lower Constrain title")]
fn right_click_constrain_title(world: &mut ApplicationWorld) {
    let title = world.run_in_window(CONSTRAIN, "Constrain").center();
    world.right_click(diagnostics::pointer_pixels(title));
    world.pump(Vec::new());
}
#[then(
    "the title context menu offers Ribbon Appearance, Show Panels, Customize User Commands, Undock Ribbon and Docking Positions"
)]
fn title_menu_choices(world: &mut ApplicationWorld) {
    for label in [
        "Ribbon Appearance",
        "Show Panels",
        "Customize User Commands",
        "Undock Ribbon",
        "Docking Positions",
    ] {
        world.expect_text(label);
    }
}
#[then("the Constrain additional command area remains open")]
fn constrain_remains_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_some());
}
#[then("Top is checked and Left and Right are available")]
fn docking_options(world: &mut ApplicationWorld) {
    assert!(world.checked("Top"));
    world.expect_text("Left");
    world.expect_text("Right");
}
#[then("the title context menu and Constrain additional command area close")]
#[then("the command context menu and Constrain additional command area close")]
fn menu_and_constrain_close(world: &mut ApplicationWorld) {
    world.expect_no_text("Undock Ribbon");
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}

fn sketch_with_constrain_area() -> ApplicationWorld {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.record_initial_sketch();
    constrain_area_open(&mut world);
    world
}

#[test]
fn escape_without_a_title_menu_keeps_the_constrain_area_open() {
    let mut world = sketch_with_constrain_area();
    world.escape();
    world.pump(Vec::new());
    constrain_remains_open(&mut world);
    assert!(
        world
            .edited_geometry()
            .same_recipe(&world.observed_sketch.as_ref().unwrap().geometry)
    );
    world.close();
}

#[test]
fn dismissing_the_title_menu_keeps_a_pinned_constrain_area_open() {
    let mut world = sketch_with_constrain_area();
    super::toggle_pin(&mut world);
    right_click_constrain_title(&mut world);
    title_menu_choices(&mut world);
    world.escape();
    world.pump(Vec::new());
    world.expect_no_text("Undock Ribbon");
    constrain_remains_open(&mut world);
    assert!(
        world
            .edited_geometry()
            .same_recipe(&world.observed_sketch.as_ref().unwrap().geometry)
    );
    world.close();
}

#[when("the user right-clicks Edit Coordinate System")]
fn right_click_coordinate_command(world: &mut ApplicationWorld) {
    let command = world
        .run_in_window(CONSTRAIN, "EditCoordinateSystem")
        .center();
    world.right_click(diagnostics::pointer_pixels(command));
    world.pump(Vec::new());
}
#[then(
    "the command context menu lists Ribbon Appearance, Add to Quick Access Toolbar, Move to Main Panel, Show Panels, Customize User Commands, Undock Ribbon and Docking Positions"
)]
#[then("the command context menu is visible")]
fn coordinate_command_menu(world: &mut ApplicationWorld) {
    title_menu_choices(world);
    for label in ["Add to Quick Access Toolbar", "Move to Main Panel"] {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: unavailable command must stay disabled"
        );
    }
}
#[then("the command context menu closes")]
fn command_menu_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("Add to Quick Access Toolbar");
    world.expect_no_text("Undock Ribbon");
}
#[then(
    "the Edit Coordinate System help tooltip appears while the pointer remains over that command"
)]
fn unavailable_command_tooltip(world: &mut ApplicationWorld) {
    for _ in 0..32 {
        if world.shows("Change the sketch origin and coordinate axes.")
            && world.shows("This command is not available in this build.")
        {
            return;
        }
        world.pump(Vec::new());
    }
    panic!("expected the unavailable Edit Coordinate System tooltip");
}

#[test]
fn command_submenu_escape_closes_submenu_then_unpinned_area() {
    let mut world = sketch_with_constrain_area();
    right_click_coordinate_command(&mut world);
    world.click_text("Show Panels");
    world.pump(Vec::new());
    world.expect_text("Layout");
    world.escape();
    world.pump(Vec::new());
    world.expect_text("Add to Quick Access Toolbar");
    constrain_remains_open(&mut world);
    world.escape();
    world.pump(Vec::new());
    command_menu_closes(&mut world);
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
    assert!(
        world
            .edited_geometry()
            .same_recipe(&world.observed_sketch.as_ref().unwrap().geometry)
    );
    world.close();
}

#[test]
fn inference_scope_context_menu_closes_on_tab_switch() {
    let mut world = sketch_with_constrain_area();
    let command = world
        .run_in_window(CONSTRAIN, "ConstraintInferenceScope")
        .center();
    world.right_click(diagnostics::pointer_pixels(command));
    world.pump(Vec::new());
    coordinate_command_menu(&mut world);
    world.click_text("Tools");
    world.pump(Vec::new());
    command_menu_closes(&mut world);
    world.expect_text("Application Options");
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
    world.click_text("Sketch");
    world.pump(Vec::new());
    world.expect_text("Constrain");
    assert!(
        world
            .edited_geometry()
            .same_recipe(&world.observed_sketch.as_ref().unwrap().geometry)
    );
    world.close();
}

#[then("sketch edit remains active")]
fn sketch_edit_active(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}
#[then("the Constrain additional command area remains closed")]
fn constrain_remains_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(CONSTRAIN).is_none());
}
