// SPDX-License-Identifier: GPL-2.0-only
//! Verify first-save cancellation with native keyboard and pointer events.
use super::{ApplicationWorld, diagnostics, x11_input};

#[test]
fn x11_first_save_cancel_keeps_unsaved_part_and_proposal() {
    let mut world = ApplicationWorld::new();
    world.pump(Vec::new());
    x11_input::focus_window();
    for label in ["New", "Metric", "Standard metric part", "Create"] {
        native_click_label(&mut world, label);
    }
    let document = world.active_id();
    let proposal = format!("{}.opd", world.document_name(document));
    for _ in 0..2 {
        x11_input::press_control_s();
        world.pump(Vec::new());
        world.expect_text("Save As");
        assert!(
            world.text_is_selected(&proposal),
            "expected selected proposal {proposal:?}"
        );
        x11_input::press_escape();
        world.pump(Vec::new());
        world.expect_no_text("Save in");
        assert_eq!(world.active_id(), document);
        assert!(world.state().document_storage[&document].path.is_none());
    }
    world.close();
}

fn native_click_label(world: &mut ApplicationWorld, label: &str) {
    let center = super::quick_access_icons::standard_icon_center(label)
        .unwrap_or_else(|| world.run(label).center());
    x11_input::click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}
