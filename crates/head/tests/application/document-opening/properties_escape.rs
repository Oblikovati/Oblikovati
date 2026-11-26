// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn escape_closes_file_properties_before_the_open_dialog() {
    let mut world = ApplicationWorld::new();
    dialog_open(&mut world);
    saved_part_selected(&mut world);
    choose_properties(&mut world);
    world.run_in_window("File Properties", "General");

    world.key(ViewportPhysicalKey::Escape, false);
    open_again(&mut world);
    part_stays_selected(&mut world);

    world.key(ViewportPhysicalKey::Escape, false);
    world.expect_no_text("Look in");
    world.close();
}
