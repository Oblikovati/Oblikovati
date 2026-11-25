// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn native_status_bar_marks_fully_constrained_sketches_in_each_palette() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    assert_eq!(world.dimensions_needed(), 0);
    for (theme, success) in [
        ("Dark", [0x22, 0xc5, 0x5e]),
        ("Light", [0x16, 0xa3, 0x4a]),
        ("High Contrast", [0x4a, 0xde, 0x80]),
    ] {
        world.app().owner.select_theme_from_gui(theme).unwrap();
        world.pump(Vec::new());
        world.save_capture(&format!("status-{theme}"));
        let caption = world.run_in_window("##status-bar", "FullyConstrained");
        assert_eq!(caption.color, success, "{theme} constraint badge");
        let dof = world.run_in_window("##status-bar", "DOF0");
        assert!(dof.maximum[0] < caption.minimum[0]);
        assert_ne!(dof.color, success);
    }

    world.app().owner.select_theme_from_gui("Dark").unwrap();
    world.pump(Vec::new());
    world.draw_rectangle();
    let needed = world.dimensions_needed();
    assert!(needed > 0);
    let unconstrained = world.run_in_window("##status-bar", &format!("{needed}dimensionsneeded"));
    assert_ne!(unconstrained.color, [0x22, 0xc5, 0x5e]);
    world.save_capture("status-dimensions-needed");
    world.close();
}
