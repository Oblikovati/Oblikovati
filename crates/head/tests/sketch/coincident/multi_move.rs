// SPDX-License-Identifier: GPL-2.0-only
//! Native multi-selection Move, solved preview, and history.
use super::*;
use oblikovati_gui::GuiSketchMoveStage;
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportSketchPreview;

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
    world.click_text("Move");
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
    let preview = crate::HeadWork::sketch_preview(&world.app().owner).unwrap();
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
    world.click_text("Move");
    assert!(!world.state().sketch_move.unwrap().optimize_single);
    world.close();
}
