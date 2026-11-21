// SPDX-License-Identifier: GPL-2.0-only
//! Select and edit nearby sketch dimensions with native pointer input.
use super::super::ApplicationWorld;
use super::super::x11_input;
use crate::HeadWork;

#[test]
fn native_overlapping_dimensions_remain_individually_editable() {
    for (length, updated, scale) in [
        ("10", "12", 1.),
        ("100", "120", 1.),
        ("10", "12", 1.5),
        ("10", "12", 2.),
    ] {
        check_dimension_edits(length, updated, scale);
    }
}

fn check_dimension_edits(length: &str, updated: &str, scale: f32) {
    let mut world = ApplicationWorld::new();
    create_midpoint_dimensions(&mut world, length, scale);
    if std::env::var_os("OBK_CAPTURE_DIMENSION_LABELS").is_some() {
        world.save_capture(&format!(
            "dimension-labels-{length}-{scale}-{}",
            std::process::id()
        ));
    }
    edit_dimension_with_mouse(&mut world, length, "d0", updated);
    world.expect_annotation(updated);
    if std::env::var_os("OBK_CAPTURE_DIMENSION_LABELS").is_some() {
        world.save_capture(&format!(
            "dimension-labels-after-edit-{length}-{scale}-{}",
            std::process::id()
        ));
    }
    edit_dimension_with_mouse(&mut world, "30.00", "d1", "45");
    world.expect_annotation("45.00");
    assert_eq!(world.dimensions_needed(), 0);
    world.close();
}

/// Create real `d0` and `d1` sketch dimensions through the command's native input.
/// For example, `create_midpoint_dimensions(world, "10", 1.)` sets `d0` to 10 mm.
pub(in super::super) fn create_midpoint_dimensions(
    world: &mut ApplicationWorld,
    length: &str,
    scale: f32,
) {
    world.start_sketch_on("YZ Plane");
    place_midpoint_dimensions(world, length);
    let options = oblikovati_model::ui_scale::UiScaleOptions::new(scale, 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(options).unwrap();
    world.pump(Vec::new());
}

/// Place the dimensions in the active sketch, for example a 10 mm `d0`.
pub(in super::super) fn place_midpoint_dimensions(world: &mut ApplicationWorld, length: &str) {
    x11_input::focus_window();
    world.pump(Vec::new());
    world.choose_from_flyout("Line", "Midpoint Line");
    world.expect_status("Select midpoint of line");
    let origin = world.sketch_pixel_of([0., 0.]);
    x11_input::click(origin);
    world.pump(Vec::new());
    world.expect_status("Select end of line");
    type_length_and_angle(world, length);
    x11_input::press_escape();
    world.pump(Vec::new());
}

fn type_length_and_angle(world: &mut ApplicationWorld, length: &str) {
    x11_input::type_digits(length);
    world.pump(Vec::new());
    x11_input::press_keysym(0xff09);
    world.pump(Vec::new());
    x11_input::type_digits("30");
    world.pump(Vec::new());
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
}

/// Open a visible dimension annotation and edit it with the native keyboard.
/// For example, edit annotation `"10"` after confirming it belongs to `"d0"`.
pub(in super::super) fn edit_dimension_with_mouse(
    world: &mut ApplicationWorld,
    label: &str,
    expected_name: &str,
    replacement: &str,
) {
    let position = world.exact_run_center(label);
    eprintln!("clicking dimension {label:?} at {position:?}");
    x11_input::click(position);
    x11_input::click(position);
    world.pump(Vec::new());
    let text = world.window_text("Edit Dimension");
    assert!(
        text.split_whitespace().any(|word| word == expected_name),
        "{label}: expected {expected_name:?} editor, got {text:?}"
    );
    x11_input::type_digits(replacement);
    world.pump(Vec::new());
    x11_input::press_keysym(0xff0d);
    world.pump(Vec::new());
}
