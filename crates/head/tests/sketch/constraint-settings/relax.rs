// SPDX-License-Identifier: GPL-2.0-only
//! Exercise Relax Mode preference staging through the native window.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
const CHOICES: [&str; 12] = [
    "Coincident",
    "Tangent",
    "Smooth(G2)",
    "Symmetric",
    "Collinear",
    "Concentric",
    "Horizontal",
    "Vertical",
    "Parallel",
    "Perpendicular",
    "Equal",
    "Fix",
];
fn open_relax(world: &mut ApplicationWorld) {
    world.click_text("Constraint Settings");
    world.click_text("Relax Mode");
}
fn assert_choices(world: &ApplicationWorld, selected: impl Fn(usize) -> bool) {
    for (index, label) in CHOICES.into_iter().enumerate() {
        assert_eq!(world.checked(label), selected(index), "{label}");
    }
}
#[given("Relax Mode is disabled with twelve removal choices")]
fn disabled_with_choices(world: &mut ApplicationWorld) {
    world.open_application_options();
    open_relax(world);
    assert!(!world.checked("Enable Relax Mode"));
    world.observed_counts = CHOICES
        .iter()
        .map(|label| usize::from(world.checked(label)))
        .collect();
}
#[given("Preserve dimensions with equation is checked and disabled")]
fn preserve_checked_disabled(world: &mut ApplicationWorld) {
    assert!(world.checked("Preserve dimensions with equation"));
    assert_ne!(
        world.run("Preserve dimensions with equation").alpha,
        u8::MAX
    );
    world.click_text("Cancel");
    world.close_application_options();
}
#[when("the user enable Relax Mode")]
fn enable_relax(world: &mut ApplicationWorld) {
    world.click_text("Enable Relax Mode");
}
#[then(
    "the removal choices and bulk-selection buttons and equation-preservation option become available"
)]
fn options_available(world: &mut ApplicationWorld) {
    for label in CHOICES.into_iter().chain([
        "Select All",
        "Clear All",
        "Preserve dimensions with equation",
    ]) {
        assert_eq!(world.run(label).alpha, u8::MAX, "{label}");
    }
}
#[when("the user click Clear All and wait for settled painting")]
fn clear_all(world: &mut ApplicationWorld) {
    world.click_text("Clear All");
    world.pump(Vec::new());
}
#[then("all twelve removal choices are unchecked")]
fn none_selected(world: &mut ApplicationWorld) {
    assert_choices(world, |_| false);
}
#[when("the user click Select All")]
fn select_all(world: &mut ApplicationWorld) {
    world.click_text("Select All");
}
#[then("all twelve removal choices are checked")]
#[then("the twelve removal choices remain checked")]
fn all_selected(world: &mut ApplicationWorld) {
    assert_choices(world, |_| true);
}
#[when("the user clear Preserve dimensions with equation")]
fn clear_preserve(world: &mut ApplicationWorld) {
    world.click_text("Preserve dimensions with equation");
}
#[when("the user cancel and reopen Constraint Settings > Relax Mode")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    open_relax(world);
}
#[then("Relax Mode is disabled and the original removal choices are restored")]
fn initial_choices(world: &mut ApplicationWorld) {
    assert!(!world.checked("Enable Relax Mode"));
    let original = world.observed_counts.clone();
    assert_eq!(original.len(), CHOICES.len());
    assert_choices(world, |index| original[index] == 1);
}
#[then("Preserve dimensions with equation is checked and disabled")]
fn preserve_restored(world: &mut ApplicationWorld) {
    preserve_checked_disabled(world);
}

#[given("Relax Mode is initially disabled")]
fn initially_disabled(world: &mut ApplicationWorld) {
    disabled_with_choices(world);
    world.click_text("Cancel");
    world.close_application_options();
}
#[then("Coincident, Tangent, Smooth(G2) and Symmetric are unchecked")]
fn initial_unchecked(world: &mut ApplicationWorld) {
    for label in CHOICES.iter().take(4) {
        assert!(!world.checked(label), "{label}");
    }
}
#[then(
    "Collinear, Concentric, Horizontal, Vertical, Parallel, Perpendicular, Equal and Fix are checked"
)]
fn initial_checked(world: &mut ApplicationWorld) {
    for label in CHOICES.iter().skip(4) {
        assert!(world.checked(label), "{label}");
    }
}
#[then("Preserve dimensions with equation is checked")]
fn preserve_checked(world: &mut ApplicationWorld) {
    assert!(world.checked("Preserve dimensions with equation"));
}

#[when("the user open Constraint Settings to Relax Mode and enable Relax Mode")]
fn open_and_enable_relax(world: &mut ApplicationWorld) {
    world.open_application_options();
    open_relax(world);
    enable_relax(world);
}

#[then(
    "the removal choices, bulk-selection buttons and equation-preservation option become available"
)]
fn relax_controls_available(world: &mut ApplicationWorld) {
    options_available(world);
}

#[when("the user click Cancel and reopen Constraint Settings to Relax Mode")]
fn cancel_then_reopen_relax(world: &mut ApplicationWorld) {
    cancel_and_reopen(world);
}

#[given(
    "`fixtures/cad/constraint-settings/constraint-settings-line.opd` is open with Sketch1 being edited"
)]
fn open_horizontal_fixture(world: &mut ApplicationWorld) {
    super::constraint_settings_inference::open_named_sketch_fixture(
        world,
        "fixtures/cad/constraint-settings/constraint-settings-line.opd",
    );
    assert_eq!(world.edited_geometry().lines().len(), 1);
}

#[given("Relax Mode is disabled with the recorded removal choices")]
fn prepare_disabled_relax(world: &mut ApplicationWorld) {
    disabled_with_choices(world);
}

#[given("Preserve dimensions with equation is checked")]
fn preserve_is_checked_for_setup(world: &mut ApplicationWorld) {
    assert!(world.checked("Preserve dimensions with equation"));
    world.click_text("Cancel");
    world.close_application_options();
}

fn accept_relax(world: &mut ApplicationWorld) {
    let point = world.run_in_window("Constraint Settings", "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(point));
}

#[when("the user open Constraint Settings to Relax Mode, enable Relax Mode and accept")]
fn accept_enabled_relax(world: &mut ApplicationWorld) {
    open_and_enable_relax(world);
    accept_relax(world);
}

#[when("the user reopen Constraint Settings to Relax Mode")]
fn reopen_relax(world: &mut ApplicationWorld) {
    open_relax(world);
}

#[then("Enable Relax Mode remains checked")]
fn relax_remains_enabled(world: &mut ApplicationWorld) {
    assert!(world.checked("Enable Relax Mode"));
}

#[when("the user clear Enable Relax Mode and accept")]
fn accept_disabled_relax(world: &mut ApplicationWorld) {
    world.click_text("Enable Relax Mode");
    accept_relax(world);
}

#[then("the original disabled preference is restored")]
fn disabled_relax_restored(world: &mut ApplicationWorld) {
    open_relax(world);
    initial_choices(world);
}
