// SPDX-License-Identifier: GPL-2.0-only
//! Joining separate oblique segments through the native Coincident tool.
use super::*;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{Sketch2D, SketchPlane};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("two separate oblique segments in the recorded endpoint-join fixture")]
fn separate_segments(world: &mut ApplicationWorld) {
    super::coincident_context::draw_oblique_lines(world);
    world.record_initial_sketch();
}
#[when("I invoke Coincident")]
fn invoke_coincident(world: &mut ApplicationWorld) {
    world.click_text("Coincident");
}
#[when("I select the first segment end followed by the second segment start")]
fn join_endpoints(world: &mut ApplicationWorld) {
    world.sketch_click([750, 350]);
    world.sketch_click([850, 400]);
}
fn endpoints(geometry: &Sketch2D) -> [[f64; 2]; 2] {
    let ids = [
        geometry.lines()[0].endpoints[1],
        geometry.lines()[1].endpoints[0],
    ];
    ids.map(|id| geometry.point(id).unwrap().position)
}
#[then("the endpoints meet at a new shared visible location")]
fn endpoints_meet(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.geometric_constraints().count(), 1);
    let pixels = endpoints(&geometry).map(|point| {
        crate::HeadWindow::sketch_pixel(&world.app().window, SketchPlane::xy(), point).unwrap()
    });
    assert!((pixels[0][0] - pixels[1][0]).abs() <= 1 && (pixels[0][1] - pixels[1][1]).abs() <= 1);
}
#[then("both segments change their visible extent or placement")]
fn both_segments_change(world: &mut ApplicationWorld) {
    let after = endpoints(&world.edited_geometry());
    let before = endpoints(&world.observed_sketch.as_ref().unwrap().geometry);
    assert_ne!(before[0], after[0]);
    assert_ne!(before[1], after[1]);
}
#[then("the prompt returns to first curve or point")]
fn ready_for_next_pick(world: &mut ApplicationWorld) {
    world.expect_status("Coincident: Select first curve or point");
    assert!(!world.state().sketch_at_ready);
}
#[then("the joined geometry remains")]
fn joined_remains(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_at_ready);
    endpoints_meet(world);
}
#[when("I press F8")]
fn show_constraint_marks(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::F8, false);
}
#[then("constraint markers are displayed including the marker at the joint")]
fn joint_marker(world: &mut ApplicationWorld) {
    world.expect_annotation("C");
    let mark = world
        .app()
        .owner
        .sketch_constraint_marks()
        .into_iter()
        .find(|mark| mark.text == "C")
        .expect("expected a Coincident mark");
    let geometry = world.edited_geometry();
    assert_eq!(mark.at, endpoints(&geometry)[0]);
}
