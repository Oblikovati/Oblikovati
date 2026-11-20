// SPDX-License-Identifier: GPL-2.0-only
//! Check the restored constraint configuration through Application Options.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

const INFERENCE_LABELS: [&str; 9] = [
    "Horizontal",
    "Vertical",
    "Parallel",
    "Perpendicular",
    "Intersection",
    "Midpoint",
    "On Curve",
    "Tangent",
    "Coincident",
];

#[given("an active sketch and Tools ribbon")]
fn sketch_with_recorded_preferences(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    prepare_recorded_preferences(world);
}
fn prepare_recorded_preferences(world: &mut ApplicationWorld) {
    world.open_application_options();
    world.click_text("Constraint Settings");
    world.click_text("General");
    world.click_text("Warn of overconstrained condition");
    world.click_text("Inference");
    world.click_text("Parallel and perpendicular priority");
    let point = world.run_in_window("Constraint Settings", "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(point));
    world.close_application_options();
    world.click_text("Sketch");
    world.click_text("Tools");
    world.pump(Vec::new());
}
#[when("the user open Application Options Sketch and Constraint Settings")]
fn open_constraint_options(world: &mut ApplicationWorld) {
    world.click_text("Application Options");
    world.pump(Vec::new());
    world.click_text("Constraint Settings");
    world.click_text("General");
}
#[then("Edit dimension when created and Create dimensions from input values are checked")]
fn general_options_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Edit dimension when created"));
    assert!(world.checked("Create dimensions from input values"));
}
#[then("Warn of overconstrained condition is selected")]
fn warn_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Warn of overconstrained condition"));
}
#[when("the user choose Inference")]
fn choose_inference(world: &mut ApplicationWorld) {
    world.click_text("Inference");
}
#[then("Infer constraints and Persist constraints and all nine types are checked")]
fn inference_selected(world: &mut ApplicationWorld) {
    for label in [
        "Infer constraints",
        "Persist constraints",
        "Horizontal",
        "Vertical",
        "Parallel",
        "Perpendicular",
        "Intersection",
        "Midpoint",
        "On Curve",
        "Tangent",
        "Coincident",
    ] {
        assert!(world.checked(label), "{label}");
    }
}
#[then("Parallel and perpendicular priority is selected")]
fn relative_priority_selected(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Parallel and perpendicular priority"));
}
#[when("the user choose Relax Mode")]
fn choose_relax(world: &mut ApplicationWorld) {
    world.click_text("Relax Mode");
}
#[then("Enable Relax Mode is unchecked and dependent controls are disabled")]
fn relax_disabled(world: &mut ApplicationWorld) {
    assert!(!world.checked("Enable Relax Mode"));
    for label in [
        "Preserve dimensions with equation",
        "Collinear",
        "Select All",
        "Clear All",
    ] {
        assert_ne!(world.run(label).alpha, u8::MAX, "{label}");
    }
}
#[when("the user cancel and close Application Options")]
fn close_options(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.close_application_options();
}
#[then("the sketch returns to Ready")]
fn sketch_ready(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_at_ready);
}

#[test]
fn input_dimension_changes_are_staged_until_parent_apply() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.open_application_options();
    world.click_text("Constraint Settings");
    world.click_text("General");
    world.click_text("Create dimensions from input values");
    world.click_text("Cancel");
    world.click_text("Constraint Settings");
    world.click_text("General");
    assert!(world.checked("Create dimensions from input values"));
    world.click_text("Create dimensions from input values");
    let point = world.run_in_window("Constraint Settings", "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(point));
    assert!(
        world
            .state()
            .sketch_options
            .unwrap()
            .create_dimensions_on_value_input
    );
    world.click_text("Apply");
    assert!(
        !world
            .state()
            .sketch_options
            .unwrap()
            .create_dimensions_on_value_input
    );
    world.close();
}

#[test]
fn changed_document_settings_cannot_be_overwritten_by_a_stale_dialog() {
    use crate::HeadSketchWork;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.open_application_options();
    world.click_text("Constraint Settings");
    let document = world.state().document.unwrap();
    let original = world.state().document_sketch_settings.unwrap();
    let mut changed = original.clone();
    changed.infer_constraints = false;
    world
        .app()
        .owner
        .set_document_sketch_settings_from_gui(document, &original, &changed)
        .unwrap();
    world.pump(Vec::new());
    assert!(world.shows("Sketch settings changed"));
    assert_ne!(
        world.run_in_window("Constraint Settings", "OK").alpha,
        u8::MAX
    );
    world.click_text("Cancel");
    assert_eq!(world.state().document_sketch_settings, Some(changed));
    world.close();
}

#[cfg(target_os = "linux")]
fn x11_click_label(world: &mut ApplicationWorld, label: &str) {
    let center = world.run(label).center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[cfg(target_os = "linux")]
#[test]
fn x11_input_changes_and_reopens_inference_settings() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::super::x11_input::focus_window();
    world.pump(Vec::new());
    for label in [
        "Tools",
        "Application Options",
        "Constraint Settings",
        "Inference",
        "Infer constraints",
        "OK",
    ] {
        x11_click_label(&mut world, label);
    }
    x11_click_label(&mut world, "Constraint Settings");
    x11_click_label(&mut world, "Inference");
    if std::env::var_os("OBK_CAPTURE_CONSTRAINT_SETTINGS").is_some() {
        for tab in ["Inference", "Relax Mode", "General"] {
            x11_click_label(&mut world, tab);
            world.save_capture(&format!("constraint-settings-{tab}-{}", std::process::id()));
        }
        x11_click_label(&mut world, "Inference");
    }
    assert!(!world.checked("Infer constraints"));
    assert!(!world.checked("Persist constraints"));
    super::super::x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    super::super::x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    assert!(super::super::diagnostics::window_geometry("Constraint Settings").is_none());
    assert!(super::super::diagnostics::window_geometry("Application Options").is_none());
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_child_option_stays_staged_until_parent_apply() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::super::x11_input::focus_window();
    world.pump(Vec::new());
    for label in [
        "Tools",
        "Application Options",
        "Constraint Settings",
        "General",
        "Create dimensions from input values",
        "OK",
    ] {
        x11_click_label(&mut world, label);
    }
    assert!(
        world
            .state()
            .sketch_options
            .unwrap()
            .create_dimensions_on_value_input
    );
    x11_click_label(&mut world, "Apply");
    assert!(
        !world
            .state()
            .sketch_options
            .unwrap()
            .create_dimensions_on_value_input
    );
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_inference_grid_clear_select_and_cancel() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    super::super::x11_input::focus_window();
    world.pump(Vec::new());
    for label in [
        "Tools",
        "Application Options",
        "Constraint Settings",
        "Inference",
    ] {
        x11_click_label(&mut world, label);
    }
    x11_click_label(&mut world, "Clear All");
    for label in INFERENCE_LABELS {
        assert!(!world.checked(label), "{label}");
    }
    x11_click_label(&mut world, "Select All");
    for label in INFERENCE_LABELS {
        assert!(world.checked(label), "{label}");
    }
    x11_click_label(&mut world, "Horizontal");
    assert!(!world.checked("Horizontal"));
    x11_click_label(&mut world, "Cancel");
    x11_click_label(&mut world, "Constraint Settings");
    x11_click_label(&mut world, "Inference");
    assert!(world.checked("Horizontal"));
    x11_click_label(&mut world, "Cancel");
    world.close();
}

#[given(
    "`fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited and the Tools ribbon visible"
)]
fn open_fixture_with_tools(world: &mut ApplicationWorld) {
    super::constraint_settings_inference::open_empty_xy_fixture(world);
    prepare_recorded_preferences(world);
}

#[when("the user open Application Options to Sketch and Constraint Settings")]
fn open_options_to_sketch(world: &mut ApplicationWorld) {
    open_constraint_options(world);
}

#[then("Enable Relax Mode is unchecked and its dependent controls are disabled")]
fn relax_and_dependents_disabled(world: &mut ApplicationWorld) {
    relax_disabled(world);
}

#[when("the user click Cancel and close Application Options")]
fn cancel_and_close_options(world: &mut ApplicationWorld) {
    close_options(world);
}
