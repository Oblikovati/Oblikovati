// SPDX-License-Identifier: GPL-2.0-only
//! Native mouse input for the Open dialog context menu.
use super::super::super::{desktop_input, diagnostics};
use super::*;

#[test]
fn native_context_menu_dismissal_keeps_the_open_dialog() {
    let mut world = ApplicationWorld::new();
    part_tile(&mut world);
    part_tile_selected(&mut world);
    let tile = world.run(PART_FILE).clone();
    desktop_input::right_click(&mut world, diagnostics::pointer_pixels(tile.center()));
    settle(&mut world);
    tile_menu(&mut world);
    open_with_part(&mut world);
    world.save_capture("native-open-context-menu-visible-3963");
    desktop_input::click(&mut world, diagnostics::pointer_pixels(file_list_gutter()));
    settle(&mut world);
    world.expect_no_text("Rename");
    no_selected_items(&mut world);
    open_with_part(&mut world);
    world.save_capture("native-open-context-menu-3963");
    world.close();
}
