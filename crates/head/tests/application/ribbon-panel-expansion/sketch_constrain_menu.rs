// SPDX-License-Identifier: GPL-2.0-only
//! Native title-menu interactions while editing a sketch.
use super::{ApplicationWorld, diagnostics, sketch_constrain};
use cucumber::{given, then, when};

const CONSTRAIN: &str = "##ribbon-additional-Constrain";

#[given("the full Sketch ribbon is visible")]
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
