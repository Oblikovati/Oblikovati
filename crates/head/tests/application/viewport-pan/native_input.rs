// SPDX-License-Identifier: GPL-2.0-only
//! Exercise viewport navigation through the Linux window's native input.
use super::super::{desktop_input::*, rendered_label_center};
use super::*;

#[test]
fn desktop_pan_drag_moves_the_view_and_keeps_the_tool_active() {
    let mut world = ApplicationWorld::new();
    open_block(&mut world);
    start_pan(&mut world);
    let before = camera(&mut world);
    drag(
        &mut world,
        EMPTY_VIEW,
        [EMPTY_VIEW[0] + 80, EMPTY_VIEW[1] + 40],
    );
    assert_pan_translation(before, camera(&mut world));
    world.expect_status("Pan:");
    escape(&mut world);
    world.expect_ready();
    world.close();
}

#[test]
fn desktop_pan_context_menu_owns_right_click_and_restores_part_menu() {
    let mut world = ApplicationWorld::new();
    let path = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/main-window-frame-start.opd")
        .canonicalize()
        .unwrap();
    open_saved_navigation_block(&mut world, &path);
    let before = camera(&mut world);
    let document = world.state().document;
    exercise_pan_menu(&mut world, before);
    assert_eq!(camera(&mut world), before);
    assert_eq!(world.state().document, document);
    assert!(!world.state().document_storage[&world.active_id()].unsaved_edits);
    right_click(&mut world, EMPTY_VIEW);
    world.pump(Vec::new());
    world.expect_text("Look At");
    escape(&mut world);
    world.close();
}

fn start_pan(world: &mut ApplicationWorld) {
    let point = diagnostics::pointer_pixels(bar_icon_center(NavigationControl::Pan));
    click(world, point);
    world.expect_status("Pan:");
}

fn open_pan_menu(world: &mut ApplicationWorld) {
    right_click(world, EMPTY_VIEW);
    world.pump(Vec::new());
    world.expect_status("Pan:");
}

fn exercise_pan_menu(world: &mut ApplicationWorld, before: ViewportCamera) {
    start_pan(world);
    open_pan_menu(world);
    pan_menu(world);
    world.save_capture("pan-context-4059");
    click(world, rendered_label_center("Done"));
    world.expect_ready();
    assert_eq!(camera(world), before);
    start_pan(world);
    open_pan_menu(world);
    click(world, rendered_label_center("Zoom"));
    world.expect_status("Zoom:");
    escape(world);
    world.expect_ready();
}
