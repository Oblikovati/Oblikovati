// SPDX-License-Identifier: GPL-2.0-only
//! Verify Find criteria and saved-search dialogs with native input (#4117).
use super::find::{
    EXTRUSION_ROW, FILLET_ROW, add_is_enabled, expect_control, expect_empty_input, expect_listed,
    input_value,
};
use super::*;

fn click(world: &mut ApplicationWorld, label: &str) {
    let center = diagnostics::pointer_pixels(world.run(label).center());
    super::super::x11_input::click(center);
    settle(world);
}

fn choose(world: &mut ApplicationWorld, label: &str, choice: &str) {
    let field = world.run(label);
    let point = [field.minimum[0] - 40., field.center()[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    settle(world);
    click(world, choice);
}

fn open_native_find(world: &mut ApplicationWorld) {
    click(world, "Menu");
    click(world, "Find");
    world.expect_text("Find Now");
}

fn native_text(world: &mut ApplicationWorld, text: &str) {
    for character in text.chars() {
        super::super::x11_input::press_path_character(character);
        world.pump(Vec::new());
    }
}

#[test]
fn native_sketch_parameter_starts_empty_and_disables_add() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    open_native_find(&mut world);
    choose(&mut world, "Find", "Sketches");
    choose(&mut world, "Property", "Sketch Parameter Name");
    expect_control(&world, "Condition", "contains");
    expect_empty_input(&world, "Value");
    assert!(!add_is_enabled(&world), "expected an empty criterion");
    world.save_capture("find-sketch-parameter-empty");
    change_and_clear_value(&mut world);
    click(&mut world, "Close");
    world.close();
}

fn change_and_clear_value(world: &mut ApplicationWorld) {
    let value = world.run("Value");
    let point = [value.minimum[0] - 40., value.center()[1]];
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    settle(world);
    native_text(world, "probe");
    expect_control(world, "Value", "probe");
    assert_eq!(input_value(world, "Value"), "probe");
    assert!(add_is_enabled(world));
    super::super::x11_input::press_control_a();
    super::super::x11_input::press_keysym(0xff08);
    settle(world);
    expect_empty_input(world, "Value");
    assert!(
        !add_is_enabled(world),
        "cleared criterion must not be added"
    );
}

fn add_native_type(world: &mut ApplicationWorld, name: &str, grouped: bool) {
    choose(world, "Property", "Type");
    choose(world, "Value", name);
    click(world, "Add to list");
    if grouped {
        world.expect_text("must be grouped");
        click(world, "Yes");
    }
}

fn prepare_native_criteria(world: &mut ApplicationWorld) {
    open_block(world);
    open_native_find(world);
    add_native_type(world, "Extrusion", false);
    add_native_type(world, "Fillet", true);
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
}

fn check_pending_value(world: &mut ApplicationWorld) {
    click(world, EXTRUSION_ROW);
    assert!(!add_is_enabled(world));
    choose(world, "Value", "Hole");
    expect_control(world, "Value", "Hole");
    assert!(add_is_enabled(world));
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
    world.save_capture("find-pending-hole");
    click(world, FILLET_ROW);
    expect_control(world, "Value", "Fillet");
    assert!(world.radio_selected("Or"));
    click(world, EXTRUSION_ROW);
    expect_control(world, "Value", "Extrusion");
    assert!(world.radio_selected("And"));
    assert!(!add_is_enabled(world));
}

fn cancel_empty_name(world: &mut ApplicationWorld) {
    click(world, "Save Search");
    world.expect_text("Search name");
    expect_empty_input(world, "Search name");
    assert!(!world.run_is_enabled("OK"));
    assert!(world.run_is_enabled("Cancel"));
    world.save_capture("find-save-prompt-empty");
    click(world, "Cancel");
    world.expect_no_text("Search name");
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
}

fn cancel_saved_picker(world: &mut ApplicationWorld) {
    click(world, "Save Search");
    native_text(world, "Types");
    click(world, "OK");
    click(world, "Open Search");
    expect_rows(world, &["Types", "Open", "Cancel", "Delete"]);
    world.save_capture("find-open-picker");
    click(world, "Cancel");
    world.expect_text("Find Now");
    expect_listed(world, &[EXTRUSION_ROW, FILLET_ROW]);
}

#[test]
fn native_find_preserves_criteria_across_pending_value_and_cancel() {
    let mut world = ApplicationWorld::new();
    prepare_native_criteria(&mut world);
    check_pending_value(&mut world);
    cancel_empty_name(&mut world);
    cancel_saved_picker(&mut world);
    click(&mut world, "Close");
    world.close();
}
