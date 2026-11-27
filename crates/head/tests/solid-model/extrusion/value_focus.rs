// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::diagnostics;
use super::super::direction::{choose_asymmetric, preview_depth};
use super::super::replace_extrusion_distance;
use super::{ApplicationWorld, feature_count, native_fixture_preview};
use cucumber::{given, then, when};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportModifiers, ViewportPhysicalKey};

#[given("a feature Properties panel stays open while values are edited")]
fn open_asymmetric_fields(world: &mut ApplicationWorld) {
    native_fixture_preview(world);
    choose_asymmetric(world);
    world.pump(Vec::new());
    assert_eq!(preview_depth(world), [-0.5, 1.]);
}

#[given("a value editor has keyboard focus")]
fn focus_first_distance(world: &mut ApplicationWorld) {
    replace_extrusion_distance(world, "8");
    assert!(distance_active("##distance-a"));
    assert_eq!(preview_depth(world), [-0.5, 0.8]);
}

#[when("the user presses Tab")]
fn tab_to_second_distance(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::Tab, false);
    world.pump(Vec::new());
}

#[then("focus advances to the next available field in reading order")]
fn second_distance_has_focus(world: &mut ApplicationWorld) {
    assert!(
        distance_active("Distance B"),
        "Distance B must own the keyboard"
    );
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("6");
    world.pump(Vec::new());
    assert_eq!(preview_depth(world), [-0.6, 0.8]);
}

#[when("the user presses Shift+Tab")]
fn tab_back_to_first_distance(world: &mut ApplicationWorld) {
    let event = |pressed| {
        ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: ViewportPhysicalKey::Tab,
            layout_key: ViewportPhysicalKey::Tab,
            pressed,
            command: None,
            modifiers: ViewportModifiers {
                shift: pressed,
                ..Default::default()
            },
        })
    };
    world.pump(vec![event(true), event(false)]);
    world.pump(Vec::new());
}

#[then("focus returns to the original field")]
fn first_distance_has_focus(world: &mut ApplicationWorld) {
    assert!(
        distance_active("##distance-a"),
        "Distance A must own the keyboard again"
    );
    assert_eq!(preview_depth(world), [-0.6, 0.8]);
}

#[then("no feature has been committed")]
fn draft_stays_uncommitted(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 0);
    assert!(world.state().undo_labels.is_empty());
    world.run_in_window("Extrude", "DistanceA");
    assert!(world.run_is_enabled("OK"));
}

fn distance_active(label: &str) -> bool {
    // The native child region uses the parent window's ID for its name.
    let window = unsafe { imgui::sys::igFindWindowByName(c"Extrude".as_ptr()) };
    assert!(!window.is_null());
    let id = unsafe {
        imgui::sys::ImGuiWindow_GetID_Str(
            window,
            c"##extrude-properties".as_ptr(),
            std::ptr::null(),
        )
    };
    diagnostics::window_input_active(&format!("Extrude/{id:08X}"), label)
}
