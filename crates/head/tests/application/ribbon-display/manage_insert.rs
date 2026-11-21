// SPDX-License-Identifier: GPL-2.0-only
//! Verify that unavailable Manage Insert commands remain visible and disabled.
use super::*;

const INSERT_COMMANDS: [&str; 5] = [
    "Derive",
    "Feature",
    "Insert iFeature",
    "Insert Object",
    "Import",
];

#[test]
#[ignore = "requires an X11 window manager"]
fn manage_insert_commands_are_visible_but_unavailable() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    world.click_text("Manage");
    world.pump(Vec::new());
    for label in INSERT_COMMANDS {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected unavailable command"
        );
        assert!(world.run(label).minimum[1] < band_height());
    }
    let command = super::super::diagnostics::pointer_pixels(world.run("Derive").center());
    super::super::x11_input::click(command);
    world.pump(Vec::new());
    assert_eq!(world.active_id(), part);
    world.expect_text("Derive");
    world.close();
}
