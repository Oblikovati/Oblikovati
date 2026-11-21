// SPDX-License-Identifier: GPL-2.0-only
//! The compact ribbon group and the viewport keep separate input ownership.
use super::*;
use cucumber::{given, then, when};

const PANE: &str = "##ribbon-temporary-panel";
const MODIFY_COMMANDS: [&str; 6] = ["Hole", "Slot", "Fillet", "Chamfer", "Shell", "Draft"];

pub(super) fn open_solid_part(world: &mut ApplicationWorld) {
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(world);
    world.previous_document = Some(world.active_id());
}

#[given("the 3D Model ribbon is docked on the left with Modify and Work Features buttons visible")]
fn compact_groups_on_left(world: &mut ApplicationWorld) {
    open_solid_part(world);
    docked_left_on_model(world);
    expect_vertical(world, &["Modify", "Work Features"]);
}

#[given("no adjacent group panel is open")]
#[then("the command list closes")]
fn compact_group_closed(_: &mut ApplicationWorld) {
    assert!(
        window_geometry(PANE).is_none(),
        "expected the command pane to close"
    );
}

fn expect_blue_group(world: &mut ApplicationWorld, group: &str) {
    let ribbon = if window_geometry(FLOATING).is_some() {
        FLOATING
    } else {
        COLUMN
    };
    let run = world.run_in_window(ribbon, &group.replace(' ', ""));
    let point = [run.center()[0] as usize, run.minimum[1] as usize - 5];
    let [red, green, blue, _] = world.pixel_at(point);
    assert!(
        blue > green && green > red,
        "{group} color {:?}: expected blue",
        [red, green, blue]
    );
}

#[then("the Modify button is highlighted blue")]
#[then("Modify is highlighted blue")]
fn modify_is_blue(world: &mut ApplicationWorld) {
    expect_blue_group(world, "Modify");
}

#[then("Work Features is highlighted blue")]
fn work_features_is_blue(world: &mut ApplicationWorld) {
    expect_blue_group(world, "Work Features");
}

fn expect_commands_beside(world: &ApplicationWorld, labels: &[&str]) {
    let (position, size) = window_geometry(FLOATING)
        .or_else(|| window_geometry(COLUMN))
        .expect("expected a floating or left-docked ribbon");
    let (pane, _) = window_geometry(PANE).expect("expected a temporary command pane");
    assert!(
        pane[0] >= position[0] + size[0],
        "{pane:?}: expected a pane beside the ribbon"
    );
    for label in labels {
        world.run_in_window(PANE, label);
    }
}

#[then(
    "an adjacent command list opens with Hole, Slot, Fillet, Chamfer, Shell, and other Modify commands"
)]
fn floating_modify_commands(world: &mut ApplicationWorld) {
    expect_commands_beside(world, &MODIFY_COMMANDS);
}

#[then(
    "an adjacent panel lists Hole, Slot, Fillet, Chamfer, Shell, Draft, and other Modify commands"
)]
fn docked_modify_commands(world: &mut ApplicationWorld) {
    floating_modify_commands(world);
    world.run_in_window(PANE, "Thread");
}

#[then("an adjacent panel lists Plane, Axis, Point, and UCS")]
fn docked_work_commands(world: &mut ApplicationWorld) {
    expect_commands_beside(world, &["Plane", "Axis", "Point", "UCS"]);
}

#[then("the Work Features panel closes")]
fn modify_replaces_work_features(world: &mut ApplicationWorld) {
    assert!(
        !world
            .window_text(PANE)
            .split_whitespace()
            .any(|word| word == "UCS")
    );
    same_part(world);
}

#[when("the user selects a face in the viewport outside the command list")]
fn pick_around_command_pane(world: &mut ApplicationWorld) {
    let at = super::super::ribbon_keyboard_navigation::visible_face_point(world);
    click_native(world, at);
}

#[then("the face is highlighted with a one-face selection indicator")]
fn one_face_is_selected(world: &mut ApplicationWorld) {
    assert_eq!(world.state().selected_face_appearances.len(), 1);
    world.run_in_window("##status-bar", "1Face");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_selected_face_fill(
        &capture, &windows
    ));
}

#[when("the user clicks empty workspace outside the part")]
fn click_empty_model_view(world: &mut ApplicationWorld) {
    let at = super::super::viewport_entity_selection::empty_viewport_point(world);
    click_native(world, at);
    same_part(world);
}

#[then("the face selection clears")]
fn face_is_deselected(world: &mut ApplicationWorld) {
    assert!(world.state().selected_face_appearances.is_empty());
    world.expect_no_text("1Face");
}

fn click_native(world: &mut ApplicationWorld, at: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(at);
    #[cfg(not(target_os = "linux"))]
    world.click(at);
    world.pump(Vec::new());
}
