// SPDX-License-Identifier: GPL-2.0-only
//! Native tests for resetting ribbon customizations.

use super::*;

#[test]
fn reset_ribbon_cancels_or_persists_defaults() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Close");
    world.expect_text("User Commands");
    open_ribbon_reset(&mut world);
    world.click_text("No");
    world.expect_text("User Commands");
    open_ribbon_reset(&mut world);
    world.click_text("Yes");
    world.expect_text("Start 2D Sketch");
    world.click_text("Tools");
    world.expect_no_text("User Commands");
    world.reopen_application();
    world.add_part();
    world.click_text("Tools");
    world.expect_no_text("User Commands");
    world.close();
}

fn open_ribbon_reset(world: &mut ApplicationWorld) {
    let title = diagnostics::pointer_pixels(world.run("Options").center());
    world.right_click(title);
    world.click_text("Ribbon Appearance");
    world.click_text("Reset Ribbon");
    assert_reset_centered();
    world.save_capture("reset-ribbon-confirmation");
}

fn assert_reset_centered() {
    let (position, size) =
        diagnostics::window_geometry("Reset Ribbon").expect("expected reset dialog");
    let context = unsafe { imgui::sys::igGetCurrentContext().as_ref() }.expect("expected GUI");
    let display = [context.IO.DisplaySize.x, context.IO.DisplaySize.y];
    for axis in 0..2 {
        assert!((position[axis] + size[axis] * 0.5 - display[axis] * 0.5).abs() < 2.);
    }
}

#[test]
fn failed_ribbon_reset_keeps_applied_commands() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    open_tools_customization(&mut world);
    stage_sketch_visibility(&mut world);
    click_in_customize(&mut world, "Apply");
    click_in_customize(&mut world, "Close");
    let destination = world
        .preferences_directory()
        .join("ribbon-customization.xml");
    std::fs::remove_file(&destination).unwrap();
    std::fs::create_dir(&destination).unwrap();
    open_ribbon_reset(&mut world);
    world.click_text("Yes");
    assert!(diagnostics::window_geometry("Reset Ribbon").is_some());
    world.expect_text("complete atomic file replacement");
    world.click_text("No");
    world.expect_text("User Commands");
    world.close();
}
