// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn native_dimension_editor_retains_arithmetic_through_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    typed_rectangle(&mut world, "10", "6");
    world.open_dimension_label("10");
    world.type_text("2*6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_annotation("12");
    world.open_dimension_label("12");
    assert!(world.window_text("Edit Dimension").contains("2mm*6ul"));
    world.escape();
    world.undo();
    world.expect_annotation("10");
    world.redo();
    world.expect_annotation("12");
    world.open_dimension_label("12");
    assert!(world.window_text("Edit Dimension").contains("2mm*6ul"));
    world.close();
}

#[test]
fn native_dimension_editor_keeps_rejected_input_for_correction() {
    let mut world = ApplicationWorld::new();
    typed_rectangle(&mut world, "10", "6");
    world.open_dimension_label("10");
    world.type_text("2*");
    world.pump(Vec::new());
    let invalid = world.run_in_window("Edit Dimension", "2*").color;
    assert!(
        invalid[0] > 200 && invalid[1] < 100 && invalid[2] < 100,
        "{invalid:?}"
    );
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert!(world.window_text("Edit Dimension").contains("2*"));
    world.expect_annotation("10");
    assert!(world.status_shows("Command failed"));
    world.expect_text("The expression cannot be evaluated.");
    world.click_text("OK");
    assert!(world.text_is_selected("2*"));
    world.key(ViewportPhysicalKey::End, false);
    world.type_text("6");
    world.pump(Vec::new());
    let valid = world.run_in_window("Edit Dimension", "2*6").color;
    assert!(valid[1] > 100 && valid[2] > 100, "{valid:?}");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.expect_annotation("12");
    world.close();
}
