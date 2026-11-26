// SPDX-License-Identifier: GPL-2.0-only
//! Verify Find criteria and saved-search dialogs with native input (#4117).
use super::find::{
    EXTRUSION_ROW, FILLET_ROW, add_is_enabled, expect_control, expect_empty_input, expect_listed,
    input_value,
};
use super::*;
use crate::work::HeadWork;
use oblikovati_viewport::ViewportPhysicalKey;

fn click(world: &mut ApplicationWorld, label: &str) {
    let center = diagnostics::pointer_pixels(world.run(label).center());
    click_at(world, center);
}

fn click_at(world: &mut ApplicationWorld, center: [i32; 2]) {
    if world.app().window.native.view.uses_wayland() {
        world.click(center);
    } else {
        super::super::x11_input::click(center);
    }
    settle(world);
}

fn choose(world: &mut ApplicationWorld, label: &str, choice: &str) {
    let field = world.run(label);
    let point = [field.minimum[0] - 40., field.center()[1]];
    click_at(world, diagnostics::pointer_pixels(point));
    click(world, choice);
}

fn open_native_find(world: &mut ApplicationWorld) {
    open_browser_menu(world);
    click(world, "Find");
    world.expect_text("Find Now");
}

fn native_text(world: &mut ApplicationWorld, text: &str) {
    if world.app().window.native.view.uses_wayland() {
        world.type_text(text);
        return;
    }
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
    world.save_capture("find-property-label");
    world.expect_text("Sketch Parameter Name");
    expect_control(&world, "Condition", "contains");
    expect_empty_input(&world, "Value");
    assert!(!add_is_enabled(&world), "expected an empty criterion");
    world.save_capture("find-sketch-parameter-empty");
    change_and_clear_value(&mut world);
    narrow_find_dialog(&mut world);
    click(&mut world, "Close");
    world.close();
}

fn narrow_find_dialog(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry("Find: Sketches###find-dialog").unwrap();
    let corner = [position[0] + size[0] - 5., position[1] + size[1] - 5.];
    let start = diagnostics::pointer_pixels(corner);
    world.drag(start, [start[0] - 250, start[1]]);
    settle(world);
    let (_, resized) = diagnostics::window_geometry("Find: Sketches###find-dialog").unwrap();
    assert!(resized[0] < size[0] - 100., "dialog was not resized");
    world.save_capture("find-property-narrow");
    world.expect_text("Sketch Parameter Name");
    world.expect_text("Condition");
    world.expect_text("Or");
}

#[test]
fn native_find_property_remains_readable_at_larger_text_scale() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    open_native_find(&mut world);
    choose(&mut world, "Find", "Sketches");
    choose(&mut world, "Property", "Sketch Parameter Name");
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(2., 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    settle(&mut world);
    world.save_capture("find-property-scale-200");
    world.expect_text("Sketch Parameter Name");
    world.expect_text("Condition");
    world.expect_text("Value");
    world.expect_text("Close");
    click(&mut world, "Close");
    world.close();
}

fn change_and_clear_value(world: &mut ApplicationWorld) {
    let value = world.run("Value");
    let point = [value.minimum[0] - 40., value.center()[1]];
    click_at(world, diagnostics::pointer_pixels(point));
    native_text(world, "probe");
    expect_control(world, "Value", "probe");
    assert_eq!(input_value(world, "Value"), "probe");
    assert!(add_is_enabled(world));
    clear_native_input(world);
    expect_empty_input(world, "Value");
    assert!(
        !add_is_enabled(world),
        "cleared criterion must not be added"
    );
}

fn clear_native_input(world: &mut ApplicationWorld) {
    if world.app().window.native.view.uses_wayland() {
        world.key(ViewportPhysicalKey::A, true);
        world.key(ViewportPhysicalKey::Backspace, false);
    } else {
        super::super::x11_input::press_control_a();
        super::super::x11_input::press_keysym(0xff08);
    }
    settle(world);
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
