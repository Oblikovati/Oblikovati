// SPDX-License-Identifier: GPL-2.0-only
//! Context-menu behavior at a coincident pair of line endpoints.
use super::*;
use cucumber::{given, then, when};
use oblikovati_model::sketch::SketchPlane;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) fn draw_oblique_lines(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    for (start, end) in [([620, 300], [750, 350]), ([850, 400], [980, 480])] {
        world.click_text("Line");
        world.sketch_click(start);
        world.sketch_click(end);
        world.escape();
    }
}
#[given("the joined line endpoints and visible constraints")]
fn joined_lines(world: &mut ApplicationWorld) {
    draw_oblique_lines(world);
    world.click_text("Coincident");
    world.sketch_click([750, 350]);
    world.sketch_click([850, 400]);
    let geometry = world.edited_geometry();
    assert_eq!(geometry.constraints().len(), 1);
    let ends = [
        geometry.lines()[0].endpoints[1],
        geometry.lines()[1].endpoints[0],
    ];
    let positions = ends.map(|id| geometry.point(id).unwrap().position);
    let pixels = positions.map(|point| {
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), point).unwrap()
    });
    assert!(
        (pixels[0][0] - pixels[1][0]).abs() <= 1 && (pixels[0][1] - pixels[1][1]).abs() <= 1,
        "expected joined endpoints at the same screen location: {pixels:?}"
    );
    world.escape();
    world.key(ViewportPhysicalKey::F8, false);
    world.pump(Vec::new());
    world.expect_annotation("C");
    let mark = world
        .app()
        .owner
        .sketch_constraint_marks()
        .into_iter()
        .find(|mark| mark.text == "C")
        .expect("expected a visible Coincident mark");
    let position =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), mark.at).unwrap();
    world.observed_view = Some(position);
    world.record_initial_sketch();
}
#[when("I right-click the joint")]
fn right_click_joint(world: &mut ApplicationWorld) {
    world.right_click(world.observed_view.expect("expected the joint position"));
    world.pump(Vec::new());
}
#[then("a Point context menu opens with Repeat Coincident and general point commands")]
fn point_menu(world: &mut ApplicationWorld) {
    world.expect_text("Repeat Coincident");
    world.expect_text("Center Point");
}
#[then("marking-menu Delete is disabled")]
fn delete_is_disabled(world: &mut ApplicationWorld) {
    let joint = world.observed_view.unwrap();
    let delete = world
        .frame
        .runs
        .iter()
        .find(|run| {
            run.text == "Delete"
                && run.center()[0] > joint[0] as f32 - 10.
                && run.center()[1] > joint[1] as f32
        })
        .expect("expected Delete in the point context menu");
    assert!(delete.alpha < u8::MAX);
}
#[then("the menu closes")]
fn menu_closes(world: &mut ApplicationWorld) {
    world.expect_no_text("Repeat Coincident");
}
#[when("I select the joint point and right-click it again")]
fn select_and_open(world: &mut ApplicationWorld) {
    world.sketch_click(world.observed_view.unwrap());
    assert!(world.state().sketch_delete_available);
    right_click_joint(world);
}
#[then("the point context menu appears without an identified constraint-removal command")]
fn no_constraint_removal(world: &mut ApplicationWorld) {
    point_menu(world);
    world.expect_no_text("Delete Coincident Constraint");
}
#[then("the menu closes and both joined segments remain")]
fn geometry_remains(world: &mut ApplicationWorld) {
    menu_closes(world);
    let current = world.edited_geometry();
    let before = &world.observed_sketch.as_ref().unwrap().geometry;
    assert_eq!(current.lines().len(), 2);
    assert!(current.same_recipe(before));
}

#[test]
fn curve_context_deletes_its_relation_without_moving_geometry_and_supports_history() {
    let mut world = ApplicationWorld::new();
    joined_lines(&mut world);
    let before = world.edited_geometry();
    let line = before.lines()[0];
    let points = line.endpoints.map(|id| before.point(id).unwrap().position);
    let midpoint = std::array::from_fn(|axis| (points[0][axis] + points[1][axis]) * 0.5);
    let pixel =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), midpoint).unwrap();
    world.right_click(pixel);
    world.pump(Vec::new());
    world.expect_text("Delete Coincident Constraint");
    world.click_text("Delete Coincident Constraint");
    let deleted = world.edited_geometry();
    assert!(deleted.geometric_constraints().next().is_none());
    assert_eq!(deleted.points(), before.points());
    assert_eq!(deleted.lines(), before.lines());
    assert!(world.app().owner.sketch_constraint_marks().is_empty());
    world.undo();
    assert!(world.edited_geometry().same_recipe(&before));
    assert!(
        world
            .app()
            .owner
            .sketch_constraint_marks()
            .iter()
            .any(|mark| mark.text == "C")
    );
    world.redo();
    assert!(world.edited_geometry().same_recipe(&deleted));
    world.close();
}

#[test]
fn stale_context_relation_cannot_delete_a_different_relation() {
    use oblikovati_api::{client::Client, wire::DeleteConstraintArgs};
    let mut world = ApplicationWorld::new();
    joined_lines(&mut world);
    let geometry = world.edited_geometry();
    let ends = geometry.lines()[0]
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let midpoint = std::array::from_fn(|axis| (ends[0][axis] + ends[1][axis]) * 0.5);
    let pixel =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), midpoint).unwrap();
    world.right_click(pixel);
    world.pump(Vec::new());
    let id = world.state().sketch_context.unwrap().relations[0].id;
    Client::new(world.app().owner.document_host())
        .constrain()
        .delete(&DeleteConstraintArgs {
            sketch_index: 0,
            constraint_index: 0,
        })
        .unwrap();
    let error =
        crate::HeadWork::delete_sketch_relation_from_gui(&mut world.app().owner, id).unwrap_err();
    assert!(error.contains("expected an available relation"));
    assert_eq!(world.edited_geometry().geometric_constraints().count(), 0);
    world.close();
}

#[test]
fn move_freed_segment_with_canvas_points_and_replay_history() {
    let mut world = ApplicationWorld::new();
    joined_lines(&mut world);
    let joined = world.edited_geometry();
    let lines = joined.lines().to_vec();
    let first = lines[0]
        .endpoints
        .map(|id| joined.point(id).unwrap().position);
    let second = lines[1]
        .endpoints
        .map(|id| joined.point(id).unwrap().position);
    let midpoint = std::array::from_fn(|axis| (first[0][axis] + first[1][axis]) * 0.5);
    let pixel =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), midpoint).unwrap();
    world.right_click(pixel);
    world.click_text("Delete Coincident Constraint");
    let deleted = world.edited_geometry();
    world.click_text("Move");
    world.expect_status("Move: Select geometry");
    world.sketch_click(pixel);
    world.expect_status("Move: Select base point");
    let base =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), first[0]).unwrap();
    world.sketch_click(base);
    world.expect_status("Move: Select end point");
    let target = [base[0] + 120, base[1] + 80];
    world.hover(target);
    assert!(crate::HeadWork::sketch_preview(&world.app().owner).is_some());
    assert!(world.edited_geometry().same_recipe(&deleted));
    world.sketch_click(target);
    let moved = world.edited_geometry();
    assert_ne!(
        moved.point(lines[0].endpoints[0]).unwrap().position,
        first[0]
    );
    assert_eq!(
        lines[1]
            .endpoints
            .map(|id| moved.point(id).unwrap().position),
        second
    );
    let done = world.run_in_window("Move", "Done").center();
    world.click(diagnostics::pointer_pixels(done));
    world.undo();
    assert!(world.edited_geometry().same_recipe(&deleted));
    world.undo();
    assert!(world.edited_geometry().same_recipe(&joined));
    world.redo();
    assert!(world.edited_geometry().same_recipe(&deleted));
    world.redo();
    assert!(world.edited_geometry().same_recipe(&moved));
    world.close();
}

#[test]
fn cancelling_a_move_discards_its_solved_preview_and_preserves_history() {
    let mut world = ApplicationWorld::new();
    joined_lines(&mut world);
    let geometry = world.edited_geometry();
    let ends = geometry.lines()[0]
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let midpoint = std::array::from_fn(|axis| (ends[0][axis] + ends[1][axis]) * 0.5);
    let middle =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), midpoint).unwrap();
    let base =
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), ends[0]).unwrap();
    let history = world.state().undo_label;
    world.click_text("Move");
    world.sketch_click(middle);
    world.sketch_click(base);
    world.hover([base[0] + 120, base[1] + 80]);
    assert!(crate::HeadWork::sketch_preview(&world.app().owner).is_some());
    world.escape();
    assert!(crate::HeadWork::sketch_preview(&world.app().owner).is_none());
    assert!(world.edited_geometry().same_recipe(&geometry));
    assert_eq!(world.state().undo_label, history);
    world.right_click([1000, 600]);
    world.expect_text("Repeat Move");
    world.close();
}
