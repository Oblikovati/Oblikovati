// SPDX-License-Identifier: GPL-2.0-only
//! Verify profile selection and separate Part and Sketch menu assignments.
use super::super::*;
use super::open_marking_menu_tab;
use oblikovati_api::types::{self, Environment};

#[given("Environment is \"2D Sketch\"")]
pub(in super::super) fn choose_sketch_environment(world: &mut ApplicationWorld) {
    if !world
        .window_text("Customize")
        .split_whitespace()
        .any(|text| text == "2DSketch")
    {
        click_in_customize(world, "Part");
        world.click_text("2D Sketch");
    }
    world.run_in_window("Customize", "2DSketch");
}

#[when("the user opens the Environment selector")]
fn open_environment(world: &mut ApplicationWorld) {
    click_in_customize(world, "2DSketch");
}

#[then(
    "the available choices include \"2D Sketch\", \"3D Sketch\", \"Assembly\", \"Drawing\", \"Part\", \"Presentation\" and \"Sheet Metal\""
)]
fn environment_choices(world: &mut ApplicationWorld) {
    let list = world.run("Sheet Metal").list;
    for name in [
        "2DSketch",
        "3DSketch",
        "Assembly",
        "Drawing",
        "Part",
        "Presentation",
        "SheetMetal",
    ] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == name),
            "expected {name} in the Environment selector"
        );
    }
}

#[then("\"2D Sketch\" remains selected")]
fn sketch_remains_selected(world: &mut ApplicationWorld) {
    world.hover([1250, 780]);
    assert!(world.row_accented_beside("2D Sketch", "Part"));
}

#[test]
fn marking_environment_applies_both_profiles_together() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_marking_menu_tab(&mut world);
    world.run_in_window("Customize", "Part");
    let part_before = north_command(&mut world, types::BASE_ENVIRONMENT);
    let sketch_before = north_command(&mut world, types::SKETCH_ENVIRONMENT);
    change_north(&mut world, "Fillet");
    choose_sketch_environment(&mut world);
    change_north(&mut world, "Rectangle");
    assert_eq!(
        north_command(&mut world, types::BASE_ENVIRONMENT),
        part_before
    );
    assert_eq!(
        north_command(&mut world, types::SKETCH_ENVIRONMENT),
        sketch_before
    );
    click_in_customize(&mut world, "Apply");
    expect_saved_assignments(&mut world);
    world.reopen_application();
    expect_saved_assignments(&mut world);
    world.close();
}

fn change_north(world: &mut ApplicationWorld, command: &str) {
    let north = world.run_in_window("Customize", "North");
    let point = [north.minimum[0] - 70., north.center()[1]];
    world.click(diagnostics::pointer_pixels(point));
    world.click_text(command);
}

fn expect_saved_assignments(world: &mut ApplicationWorld) {
    assert_eq!(
        north_command(world, types::BASE_ENVIRONMENT),
        "Modify.Fillet"
    );
    assert_eq!(
        north_command(world, types::SKETCH_ENVIRONMENT),
        "Sketch.Rectangle"
    );
}

fn north_command(world: &mut ApplicationWorld, environment: Environment) -> String {
    world
        .state()
        .marking_menus
        .unwrap()
        .get(environment)
        .quadrants
        .unwrap()
        .into_iter()
        .find(|slot| slot.quadrant == types::QUADRANT_NORTH)
        .expect("expected a configured north slot")
        .command_id
}
