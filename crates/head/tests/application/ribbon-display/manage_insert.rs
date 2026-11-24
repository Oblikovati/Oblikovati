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

#[given("a part is active in the MCAD Software with the full Manage ribbon displayed")]
fn part_on_manage(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
    use crate::HeadWork;
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(1., 1.75).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    world.click_text("Manage");
    world.expect_text("Parameters");
}

#[given("the Insert panel displays its commands directly at the wider window size")]
#[then("the Insert commands appear directly in the ribbon again")]
fn insert_commands_inline(world: &mut ApplicationWorld) {
    for label in INSERT_COMMANDS {
        assert!(world.run(label).minimum[1] < band_height());
    }
}

#[when("the user narrows the application window until Insert becomes a compact panel button")]
fn narrow_insert(world: &mut ApplicationWorld) {
    super::window_compaction::resize_window(world, 640);
    insert_commands_closed(world);
}

#[then("the Manage tab remains active")]
fn manage_stays_active(world: &mut ApplicationWorld) {
    world.expect_text("Parameters");
    assert_eq!(world.active_id(), world.previous_document.unwrap());
}

#[when("the user clicks the compact Insert panel button")]
fn open_insert(world: &mut ApplicationWorld) {
    super::view_compaction::click_compact_panel(world, "Insert");
}

#[then("a temporary Insert panel opens below the ribbon with its commands")]
fn insert_commands_open(world: &mut ApplicationWorld) {
    for label in INSERT_COMMANDS {
        assert!(world.run(label).minimum[1] >= band_height());
    }
}

#[then("the temporary panel closes and the compact Insert button remains visible")]
fn insert_commands_closed(world: &mut ApplicationWorld) {
    world.expect_text("Insert");
    world.expect_no_text("Derive");
}

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
