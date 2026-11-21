// SPDX-License-Identifier: GPL-2.0-only
//! Open the saved block before setting the requested starting view.

use super::super::viewport_orientation::open_saved_navigation_block;
use super::*;
use std::path::Path;

const ORBIT_BLOCK: &str = "fixtures/cad/body-properties-start.opd";

#[given(
    "`fixtures/cad/body-properties-start.opd` is open with its rectangular solid visible in the active part in the MCAD Software"
)]
fn saved_block_visible(world: &mut ApplicationWorld) {
    open_saved_navigation_block(world, &super::super::scenario_root().join(ORBIT_BLOCK));
    world.observed_view_camera = Some(camera(world));
}

#[given(
    "`fixtures/cad/body-properties-start.opd` is open with a rectangular face oriented toward the viewer in the active part in the MCAD Software"
)]
fn saved_block_face_on(world: &mut ApplicationWorld) {
    saved_block_visible(world);
    let view = camera(world);
    let normal: [f64; 3] = std::array::from_fn(|axis| view.eye[axis] - view.target[axis]);
    let aligned = normal.iter().filter(|axis| axis.abs() > SAME_VIEW).count();
    assert_eq!(aligned, 1, "expected a face-on view, got {normal:?}");
}

#[given(
    "`fixtures/cad/body-properties-start.opd` is open with its solid at an inclined orientation in the MCAD Software"
)]
fn saved_block_inclined(world: &mut ApplicationWorld) {
    saved_block_face_on(world);
    choose_rotate(world);
    first_direction(world);
    view_changes(world);
    choose(world, "OK");
    solid_extent(world);
    world.observed_view_camera = Some(camera(world));
}

#[cfg(target_os = "linux")]
#[test]
fn x11_orbit_drag_and_escape_keep_the_saved_block_active() {
    let mut world = ApplicationWorld::new();
    open_saved_navigation_block(&mut world, &bundled_orbit_block());
    let before = camera(&mut world);
    start_orbit_with_x11(&mut world);
    drag_orbit_with_x11(&mut world);
    assert_ne!(camera(&mut world).eye, before.eye);
    world.save_capture("saved-block-orbit-guide");
    super::super::x11_input::press_escape();
    world.pump(Vec::new());
    world.expect_ready();
    expect_guide(&mut world, false);
    super::super::viewport_pan::expect_part_and_solid(&mut world);
    world.close();
}

#[test]
fn rotate_dialog_dismissal_replaces_the_saved_part_open_notice() {
    for button in ["OK", "Cancel"] {
        let mut world = ApplicationWorld::new();
        open_saved_navigation_block(&mut world, &bundled_orbit_block());
        world.expect_status("Opened document");
        choose_rotate(&mut world);
        first_direction(&mut world);
        choose(&mut world, button);
        dialog_closed_ready(&mut world);
        world.close();
    }
}

fn bundled_orbit_block() -> std::path::PathBuf {
    Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/document-saving/SaveAsFolder/ExistingSolidPart.opd")
        .canonicalize()
        .unwrap()
}

#[cfg(target_os = "linux")]
fn start_orbit_with_x11(world: &mut ApplicationWorld) {
    use super::super::x11_input::{click, focus_window, move_pointer};
    focus_window();
    world.pump(Vec::new());
    let control = world.run_in_window("##navigation-bar", "O").center();
    let control = diagnostics::pointer_pixels(control);
    move_pointer(control);
    world.pump(Vec::new());
    click(control);
    world.pump(Vec::new());
    free_rotation(world);
    expect_guide(world, true);
}

#[cfg(target_os = "linux")]
fn drag_orbit_with_x11(world: &mut ApplicationWorld) {
    use super::super::x11_input::{left_button, move_pointer};
    let [start, end] = DRAG;
    move_pointer(start);
    world.pump(Vec::new());
    left_button(start, true);
    world.pump(Vec::new());
    move_pointer(end);
    world.pump(Vec::new());
    left_button(end, false);
    world.pump(Vec::new());
}
