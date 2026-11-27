// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{viewport_orientation, viewport_zoom};
use super::*;

#[test]
fn applied_ribbon_zoom_commands_use_the_viewport_navigation() {
    let mut world = ApplicationWorld::new();
    viewport_orientation::open_block(&mut world);
    add_zoom_commands(&mut world);
    for (command, prompt) in [
        ("Zoom", "drag to zoom"),
        ("Zoom Window", "select the window's first corner"),
        ("Zoom Selected", "select an entity to zoom"),
    ] {
        world.click_text(command);
        world.expect_status(prompt);
        world.escape();
        world.expect_status("Ready");
    }
    viewport_zoom::shrink_the_block(&mut world);
    let [width, _] = viewport_orientation::solid_extent(&mut world);
    world.click_text("Zoom All");
    world.pump(Vec::new());
    assert!(viewport_orientation::solid_extent(&mut world)[0] > width);
    assert!(viewport_orientation::whole_block_visible(&mut world));
    world.close();
}

fn add_zoom_commands(world: &mut ApplicationWorld) {
    open_tools_customization(world);
    click_in_customize(world, "AllCommands");
    world.click_text("View Tab");
    for command in ["Zoom", "ZoomAll", "ZoomSelected", "ZoomWindow"] {
        transaction_steps::add_command(world, command);
        toggle_row_option(world, command, "Text");
    }
    click_in_customize(world, "Apply");
    click_in_customize(world, "Close");
    world.expect_text("User Commands");
}
