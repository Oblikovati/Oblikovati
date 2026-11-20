// SPDX-License-Identifier: GPL-2.0-only
//! Exercise accepted inference settings through the native window.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

#[given("Constraint Settings > Inference shows Infer constraints and Persist constraints checked")]
fn inference_settings_are_open(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.open_application_options();
    open_inference(world);
    assert!(world.checked("Infer constraints"));
    assert!(world.checked("Persist constraints"));
}
fn open_inference(world: &mut ApplicationWorld) {
    world.click_text("Constraint Settings");
    world.click_text("Inference");
}
fn accept(world: &mut ApplicationWorld) {
    let point = world.run_in_window("Constraint Settings", "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(point));
}
#[when("the user clear Infer constraints and accept")]
fn clear_inference(world: &mut ApplicationWorld) {
    world.click_text("Infer constraints");
    accept(world);
}
#[when("the user reopen the Inference tab")]
fn reopen_inference(world: &mut ApplicationWorld) {
    open_inference(world);
}
#[then("Infer constraints is unchecked and Persist constraints is unchecked and disabled")]
fn inference_disabled(world: &mut ApplicationWorld) {
    assert!(!world.checked("Infer constraints"));
    assert!(!world.checked("Persist constraints"));
    assert_ne!(world.run("Persist constraints").alpha, u8::MAX);
}
#[when("the user enable Infer constraints")]
fn enable_inference(world: &mut ApplicationWorld) {
    world.click_text("Infer constraints");
}
#[then("Persist constraints becomes available and remains unchecked")]
fn persistence_enabled_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Persist constraints"));
    assert_eq!(world.run("Persist constraints").alpha, u8::MAX);
}
#[when("the user check Persist constraints and accept")]
fn restore_persistence(world: &mut ApplicationWorld) {
    world.click_text("Persist constraints");
    accept(world);
}
#[then("both settings are restored")]
fn settings_restored(world: &mut ApplicationWorld) {
    let settings = world.state().document_sketch_settings.unwrap();
    assert!(settings.infer_constraints && settings.persist_inferred_constraints);
}

const NINE_TYPES: [&str; 9] = [
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
fn click_setting(world: &mut ApplicationWorld, text: &str) {
    world.click_text(text);
}
fn assert_types(world: &ApplicationWorld, expected: bool) {
    for label in NINE_TYPES {
        let row = world.run(label);
        assert_eq!(world.checked_run(row), expected, "{label}");
    }
}
#[given("Infer constraints and Persist constraints and all nine inference types are checked")]
fn all_inference_types_checked(world: &mut ApplicationWorld) {
    inference_settings_are_open(world);
    assert_types(world, true);
}
#[given("Parallel and perpendicular priority is selected")]
fn select_relative_priority(world: &mut ApplicationWorld) {
    click_setting(world, "Parallel and perpendicular priority");
    accept(world);
    world.close_application_options();
}
#[when("the user click Clear All")]
fn clear_all_types(world: &mut ApplicationWorld) {
    click_setting(world, "Clear All");
}
#[then("all nine inference types are unchecked")]
fn all_types_unchecked(world: &mut ApplicationWorld) {
    assert_types(world, false);
}
#[when("the user click Select All and then clear Infer constraints")]
fn select_all_and_clear_inference(world: &mut ApplicationWorld) {
    click_setting(world, "Select All");
    click_setting(world, "Infer constraints");
}
#[then(
    "Persist constraints and the checked inference type controls and bulk-selection buttons are disabled"
)]
fn disabled_inference_controls(world: &mut ApplicationWorld) {
    assert_types(world, true);
    for label in NINE_TYPES
        .into_iter()
        .chain(["Persist constraints", "Select All", "Clear All"])
    {
        assert_ne!(world.run(label).alpha, u8::MAX, "{label}");
    }
}
#[then("the priority choices remain available")]
fn priority_stays_available(world: &mut ApplicationWorld) {
    assert_eq!(
        world.run("Parallel and perpendicular priority").alpha,
        u8::MAX
    );
}
#[when("the user click Cancel and reopen Constraint Settings > Inference")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    click_setting(world, "Cancel");
    open_inference(world);
}
#[then("Infer constraints and Persist constraints and all nine inference types are checked")]
fn all_types_restored(world: &mut ApplicationWorld) {
    assert!(world.checked("Infer constraints") && world.checked("Persist constraints"));
    assert_types(world, true);
}
#[then("Parallel and perpendicular priority remains selected")]
fn relative_priority_restored(world: &mut ApplicationWorld) {
    assert!(world.radio_selected("Parallel and perpendicular priority"));
}

#[given("`fixtures/cad/planar-sketch-empty-xy.opd` is open with Sketch1 being edited")]
pub(super) fn open_empty_xy_fixture(world: &mut ApplicationWorld) {
    open_named_sketch_fixture(world, "fixtures/cad/planar-sketch-empty-xy.opd");
    assert!(world.edited_geometry().lines().is_empty());
}

pub(super) fn open_named_sketch_fixture(world: &mut ApplicationWorld, name: &str) {
    let path = super::super::scenario_root().join(name);
    world.reopen_through_dialog(&path);
    world.pump(Vec::new());
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected the saved Sketch1 row")
        .center()
        .map(|v| v as i32);
    world.double_click(row);
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(world.state().sketch_at_ready);
}

#[given("Infer constraints and Persist constraints are checked")]
fn prepare_inference_baseline(world: &mut ApplicationWorld) {
    world.open_application_options();
    open_inference(world);
    for label in ["Infer constraints", "Persist constraints"] {
        if !world.checked(label) {
            world.click_text(label);
        }
    }
    assert!(world.checked("Infer constraints") && world.checked("Persist constraints"));
    accept(world);
    world.close_application_options();
}

#[when("the user open Constraint Settings to Inference")]
fn open_inference_from_sketch(world: &mut ApplicationWorld) {
    world.open_application_options();
    open_inference(world);
}

#[then("Infer constraints is unchecked")]
fn infer_constraints_is_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked("Infer constraints"));
}

#[then("Persist constraints is unchecked and disabled")]
fn persist_constraints_is_disabled(world: &mut ApplicationWorld) {
    assert!(!world.checked("Persist constraints"));
    assert_ne!(world.run("Persist constraints").alpha, u8::MAX);
}

#[given("Infer constraints, Persist constraints and all nine inference types are checked")]
fn prepare_all_inference_types(world: &mut ApplicationWorld) {
    world.open_application_options();
    open_inference(world);
    for label in ["Infer constraints", "Persist constraints"] {
        if !world.checked(label) {
            world.click_text(label);
        }
    }
    world.click_text("Select All");
    assert_types(world, true);
}

#[when("the user open Constraint Settings to Inference and click Clear All")]
fn open_inference_and_clear(world: &mut ApplicationWorld) {
    open_inference_from_sketch(world);
    clear_all_types(world);
}

#[when("the user click Select All and clear Infer constraints")]
fn select_all_then_disable_inference(world: &mut ApplicationWorld) {
    select_all_and_clear_inference(world);
}

#[then("Persist constraints, the checked inference types and bulk-selection buttons are disabled")]
fn inference_controls_are_disabled(world: &mut ApplicationWorld) {
    disabled_inference_controls(world);
}

#[when("the user click Cancel and reopen Constraint Settings to Inference")]
fn cancel_then_reopen_inference(world: &mut ApplicationWorld) {
    cancel_and_reopen(world);
}

#[then("Infer constraints, Persist constraints and all nine inference types are checked")]
fn inference_types_are_restored(world: &mut ApplicationWorld) {
    all_types_restored(world);
}
