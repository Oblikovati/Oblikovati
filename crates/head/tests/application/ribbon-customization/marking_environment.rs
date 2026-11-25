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

#[when("the user selects \"3D Sketch\" in the Environment selector")]
fn choose_sketch_3d_environment(world: &mut ApplicationWorld) {
    open_environment(world);
    world.click_text("3D Sketch");
    sketch_3d_remains_selected(world);
}

#[then("Environment remains \"3D Sketch\"")]
fn sketch_3d_remains_selected(world: &mut ApplicationWorld) {
    world.run_in_window("Customize", "3DSketch");
}

#[then(
    "the menu-location layout shows \"Bend\", \"Three Point Arc\", \"Undo\", \"Helical Curve\", \"General Dimension\", \"Include Geometry\" and \"Finish 3D Sketch\""
)]
fn sketch_3d_locations(world: &mut ApplicationWorld) {
    let mut previous_y = 0.;
    for name in [
        "Line",
        "ThreePointArc",
        "HelicalCurve",
        "IncludeGeometry",
        "Finish3DSketch",
        "GeneralDimension",
        "Undo",
        "Bend",
    ] {
        let y = world.run_in_window("Customize", name).center()[1];
        assert!(y > previous_y, "expected {name} in clockwise slot order");
        previous_y = y;
    }
}

#[when("the user opens the Sub-Environment selector")]
fn open_sub_environment(world: &mut ApplicationWorld) {
    click_in_customize(world, "NoneSelected");
}

#[then(
    "the available choices include \"Arc\", \"Circle\", \"Dimension\", \"Line\", \"Point\", \"Sketch Block\", \"Sketch Pattern\", \"Spline\" and \"Text\""
)]
fn sketch_sub_environment_choices(world: &mut ApplicationWorld) {
    sub_environment_choices(
        world,
        "Constraint Glyphs",
        &[
            "Arc",
            "Circle",
            "Dimension",
            "Line",
            "Point",
            "SketchBlock",
            "SketchPattern",
            "Spline",
            "Text",
        ],
    );
}

#[then(
    "the available choices include \"3D Line, 3D Arc\", \"3D Point\", \"Equation Curve\", \"Helical Curve\", \"Projected Curves\" and \"Spline\""
)]
fn sketch_3d_sub_environment_choices(world: &mut ApplicationWorld) {
    sub_environment_choices(
        world,
        "Projected Curves",
        &[
            "3DLine,3DArc",
            "3DPoint",
            "EquationCurve",
            "HelicalCurve",
            "ProjectedCurves",
            "Spline",
        ],
    );
}

fn sub_environment_choices(world: &ApplicationWorld, unique_row: &str, choices: &[&str]) {
    let list = world.run(unique_row).list;
    for &name in choices {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.list == list && run.text == name),
            "expected {name} in the Sub-Environment selector"
        );
    }
}

#[then("\"None Selected\" remains highlighted")]
fn none_selected_highlighted(world: &mut ApplicationWorld) {
    world.hover([1250, 780]);
    let plain = world.run("Spline").clone();
    let selected = world
        .frame
        .runs
        .iter()
        .find(|run| run.list == plain.list && run.text == "NoneSelected")
        .expect("None Selected in the Sub-Environment selector")
        .clone();
    let x = selected.maximum[0].max(plain.maximum[0]) as usize + 6;
    assert_ne!(
        world.pixel_at([x, selected.center()[1] as usize]),
        world.pixel_at([x, plain.center()[1] as usize])
    );
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
