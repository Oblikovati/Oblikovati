// SPDX-License-Identifier: GPL-2.0-only
//! Open the saved block before setting the requested starting view.

use super::super::viewport_orientation::open_saved_navigation_block;
use super::super::viewport_pan::{NavigationControl, bar_icon_center};
use super::*;
use std::path::Path;

const ORBIT_BLOCK: &str = "fixtures/cad/body-properties-start.opd";
const CURRENT_ORBIT_BLOCK: &str = "fixtures/cad/docking-visible-solid-current.opd";

#[given(
    "`fixtures/cad/body-properties-start.opd` is open with its rectangular solid visible in the active part in the MCAD Software"
)]
fn saved_block_visible(world: &mut ApplicationWorld) {
    open_orbit_block(world, ORBIT_BLOCK);
}

fn open_orbit_block(world: &mut ApplicationWorld, relative_path: &str) {
    open_saved_navigation_block(world, &super::super::scenario_root().join(relative_path));
    world.observed_view_camera = Some(camera(world));
}

#[given(
    "`fixtures/cad/body-properties-start.opd` is open with a rectangular face oriented toward the viewer in the active part in the MCAD Software"
)]
fn saved_block_face_on(world: &mut ApplicationWorld) {
    saved_block_visible(world);
    expect_face_on(world);
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with a rectangular face oriented toward the viewer in the active part in the MCAD Software"
)]
fn current_saved_block_face_on(world: &mut ApplicationWorld) {
    open_orbit_block(world, CURRENT_ORBIT_BLOCK);
    expect_face_on(world);
}

fn expect_face_on(world: &mut ApplicationWorld) {
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
    incline_block(world);
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with its solid at an inclined orientation in the MCAD Software"
)]
fn current_saved_block_inclined(world: &mut ApplicationWorld) {
    current_saved_block_face_on(world);
    incline_block(world);
}

fn incline_block(world: &mut ApplicationWorld) {
    choose_rotate(world);
    first_direction(world);
    view_changes(world);
    choose(world, "OK");
    solid_extent(world);
    world.observed_view_camera = Some(camera(world));
}

#[cfg(target_os = "linux")]
#[test]
fn desktop_orbit_drag_and_escape_keep_the_saved_block_active() {
    use super::super::viewport_pan::native_input;

    let mut world = ApplicationWorld::new();
    open_saved_navigation_block(&mut world, &bundled_orbit_block());
    let before = camera(&mut world);
    let control = diagnostics::pointer_pixels(bar_icon_center(NavigationControl::Orbit));
    native_input::click(&mut world, control);
    free_rotation(&mut world);
    expect_guide(&mut world, true);
    native_input::drag(&mut world, DRAG[0], DRAG[1]);
    assert_ne!(camera(&mut world).eye, before.eye);
    world.save_capture("saved-block-orbit-guide");
    native_input::escape(&mut world);
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
