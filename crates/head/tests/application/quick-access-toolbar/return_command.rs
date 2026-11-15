// SPDX-License-Identifier: GPL-2.0-only
//! Return exits a sketch through the same owner action as Finish Sketch.
use super::*;
use oblikovati_gui::GuiAuthoringContext;

#[test]
fn return_finishes_a_rectangle_sketch_and_is_inert_at_part_level() {
    let mut world = ApplicationWorld::new();
    part_active(&mut world);
    open_customization(&mut world);
    world.click_text("Return");
    world.pump(Vec::new());
    assert!(on_row(&world, "Return"));
    assert_return_inert(&mut world);
    world.start_sketch();
    world.draw_rectangle();
    assert!(world.run_is_enabled("Return"));
    world.click(row_center(&world, "Return"));
    world.pump(Vec::new());
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert_eq!(world.browser_sketch_count(), 1);
    assert!(world.state().extrude_available);
    assert!(!world.run_is_enabled("Return"));
    verify_finish_history(&mut world);
    world.close();
}

fn verify_finish_history(world: &mut ApplicationWorld) {
    assert_eq!(world.state().undo_label.as_deref(), Some("Finish Sketch"));
    world.click(row_center(world, "Undo"));
    world.pump(Vec::new());
    assert_eq!(world.state().authoring, GuiAuthoringContext::Sketch);
    assert_eq!(world.state().redo_label.as_deref(), Some("Finish Sketch"));
    world.click(row_center(world, "Redo"));
    world.pump(Vec::new());
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert!(world.state().extrude_available);
}

fn assert_return_inert(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("Return"));
    let history = world.state().undo_labels;
    world.click(row_center(world, "Return"));
    world.pump(Vec::new());
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert_eq!(world.state().undo_labels, history);
}
