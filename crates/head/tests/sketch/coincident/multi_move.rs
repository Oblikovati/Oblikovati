// SPDX-License-Identifier: GPL-2.0-only
//! Native multi-selection Move, solved preview, and history.
use super::move_precise::open_move;
use super::*;
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportSketchPreview;

#[test]
fn precise_relative_move_accepts_keyboard_input_and_replays_history() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    super::coincident_context::draw_oblique_lines(&mut world);
    let before = world.edited_geometry();
    let point = before.lines()[0].endpoints[0];
    let origin = before.point(point).unwrap().position;
    open_move(&mut world);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    world.sketch_click([685, 325]);
    let reference = world.run_in_window("Precise Input", "Abs").center();
    world.click(diagnostics::pointer_pixels(reference));
    world.click_text("Relative");
    let base =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), origin).unwrap();
    world.sketch_click(base);
    world.expect_status("Move: Select end point");
    world.click_precise_field("X");
    world.type_text("10");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    let after = world.edited_geometry();
    let moved = after.point(point).unwrap().position;
    assert!(
        (moved[0] - origin[0] - 1.).abs() < 1e-8,
        "{origin:?} -> {moved:?}"
    );
    assert!(
        (moved[1] - origin[1]).abs() < 1e-8,
        "{origin:?} -> {moved:?}"
    );
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    world.undo();
    assert!(world.edited_geometry().same_recipe(&before));
    world.redo();
    assert!(world.edited_geometry().same_recipe(&after));
    world.close();
}

#[test]
fn invalid_precise_move_can_be_corrected_without_creating_history() {
    use oblikovati_viewport::ViewportPhysicalKey;
    let mut world = ApplicationWorld::new();
    super::coincident_context::draw_oblique_lines(&mut world);
    let before = world.edited_geometry();
    let point = before.lines()[0].endpoints[0];
    let origin = before.point(point).unwrap().position;
    open_move(&mut world);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    world.sketch_click([685, 325]);
    let base =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), origin).unwrap();
    world.sketch_click(base);
    let history = world.state().undo_label;
    world.click_precise_field("X");
    world.type_text("q");
    world.key(ViewportPhysicalKey::Enter, false);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Target
    );
    assert!(world.edited_geometry().same_recipe(&before));
    assert_eq!(world.state().undo_label, history);
    world.click_precise_field("X");
    world.type_text("-5");
    world.key(ViewportPhysicalKey::Tab, false);
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Geometry
    );
    assert!(!world.edited_geometry().same_recipe(&before));
    world.close();
}

#[test]
fn move_triad_can_be_repositioned_reset_and_cancelled_without_placing_geometry() {
    let mut world = ApplicationWorld::new();
    super::coincident_context::draw_oblique_lines(&mut world);
    let before = world.edited_geometry();
    let history = world.state().undo_label;
    open_move(&mut world);
    let check = world.run_in_window("Move", "PreciseInput").center();
    world.click(diagnostics::pointer_pixels(check));
    world.sketch_click([685, 325]);
    world.click_text("Reposition Triad");
    world.sketch_click([800, 600]);
    world.sketch_click([900, 650]);
    let (_, triad) = world.app().owner.precise_triad_position().unwrap();
    assert_ne!(triad, [0., 0.]);
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    open_move(&mut world);
    world.sketch_click([685, 325]);
    assert_eq!(world.app().owner.precise_triad_position().unwrap().1, triad);
    world.click_text("Reset to Origin");
    assert_eq!(
        world.app().owner.precise_triad_position().unwrap().1,
        [0., 0.]
    );
    world.click_text("Reposition Triad");
    world.escape();
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    assert!(world.edited_geometry().same_recipe(&before));
    assert_eq!(world.state().undo_label, history);
    world.escape();
    assert!(world.state().sketch_move.is_none());
    world.close();
}

#[test]
fn move_two_segments_as_one_edit_and_remember_selection_mode() {
    let mut world = ApplicationWorld::new();
    super::coincident_context::draw_oblique_lines(&mut world);
    let before = world.edited_geometry();
    let ids = [before.lines()[0].id, before.lines()[1].id];
    let points = [
        before.lines()[0].endpoints[0],
        before.lines()[1].endpoints[0],
    ];
    let origins = points.map(|id| before.point(id).unwrap().position);
    open_move(&mut world);
    let optimize = world
        .run_in_window("Move", "OptimizeforSingleSelection")
        .center();
    world.click(diagnostics::pointer_pixels(optimize));
    assert!(!world.state().sketch_move.unwrap().optimize_single);
    world.sketch_click([685, 325]);
    world.sketch_click([915, 440]);
    let status = world.state().sketch_move.unwrap();
    assert_eq!(status.stage, GuiSketchMoveStage::Geometry);
    assert_eq!(status.selection_count, 2);
    let button = world.run_in_window("Move", "BasePoint").center();
    world.click(diagnostics::pointer_pixels(button));
    assert_eq!(
        world.state().sketch_move.unwrap().stage,
        GuiSketchMoveStage::Base
    );
    let base = crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), origins[0])
        .unwrap();
    let target = [base[0] + 120, base[1] + 80];
    world.sketch_click(base);
    world.hover(target);
    let preview = crate::HeadSketchWork::sketch_preview(&world.app().owner).unwrap();
    assert!(matches!(preview, ViewportSketchPreview::Resolved { members, .. } if members == ids));
    assert!(world.edited_geometry().same_recipe(&before));
    world.sketch_click(target);
    let after = world.edited_geometry();
    for (index, id) in points.into_iter().enumerate() {
        let start =
            crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), origins[index])
                .unwrap();
        let end = crate::HeadWindow::sketch_pixel(
            &world.app().window,
            SketchPlane::xy(),
            after.point(id).unwrap().position,
        )
        .unwrap();
        let expected = [start[0] + 120, start[1] + 80];
        assert!(
            (end[0] - expected[0]).abs() <= 1 && (end[1] - expected[1]).abs() <= 1,
            "point {id:?}: expected {expected:?}, observed {end:?}"
        );
    }
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    world.undo();
    assert!(world.edited_geometry().same_recipe(&before));
    world.redo();
    assert!(world.edited_geometry().same_recipe(&after));
    open_move(&mut world);
    assert!(!world.state().sketch_move.unwrap().optimize_single);
    world.close();
}
