// SPDX-License-Identifier: GPL-2.0-only
//! Native mouse input for the Open dialog context menu.
use super::super::super::{diagnostics, x11_input};
use super::*;

#[test]
fn native_context_menu_dismissal_keeps_the_open_dialog() {
    let mut world = ApplicationWorld::new();
    part_tile(&mut world);
    part_tile_selected(&mut world);
    let tile = world.run(PART_FILE).clone();
    x11_input::right_click(diagnostics::pointer_pixels(tile.center()));
    settle(&mut world);
    tile_menu(&mut world);
    open_with_part(&mut world);
    world.save_capture("native-open-context-menu-visible-3963");
    let empty = [tile.center()[0] + 120., tile.center()[1] + 90.];
    x11_input::click(diagnostics::pointer_pixels(empty));
    settle(&mut world);
    world.expect_no_text("Rename");
    no_selected_items(&mut world);
    open_with_part(&mut world);
    world.save_capture("native-open-context-menu-3963");
    world.close();
}
