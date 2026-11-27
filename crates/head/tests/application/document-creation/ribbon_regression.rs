// SPDX-License-Identifier: GPL-2.0-only
//! The creation scenario observes the ribbon without activating its commands.
use super::*;

#[test]
fn observing_assembly_ribbon_does_not_open_the_place_panel() {
    let mut world = ApplicationWorld::new();
    open_chooser(&mut world);
    metric_folder(&mut world);
    select_assembly(&mut world);
    create_part(&mut world);
    assembly_ribbon(&mut world);
    assert!(super::super::diagnostics::window_geometry("Place Component").is_none());
    let before = world.state().documents.len();
    world.click_text("Place");
    assert!(super::super::diagnostics::window_geometry("Place Component").is_some());
    assert_eq!(world.state().documents.len(), before);
    world.close();
}
