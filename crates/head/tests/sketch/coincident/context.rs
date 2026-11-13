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
