// SPDX-License-Identifier: GPL-2.0-only
//! Native selection of an exact planar face for a new sketch.
use super::super::{
    ApplicationWorld, capture_framebuffer, diagnostics, solid_visibility, viewport_orientation,
    workspace_layout::browser_feature,
};
use cucumber::{given, then, when};
use oblikovati_model::document_sketch::SketchPlacement;

#[given("isolated Part3 contains a 10 mm rectangular extrusion")]
fn part_with_extrusion(world: &mut ApplicationWorld) {
    browser_feature::open_extruded_part_for_face_sketch(world);
    browser_feature::expect_solid_visible(world);
    assert_eq!(world.browser_sketch_count(), 1);
}
#[when("the user invoke Start 2D Sketch and select its broad planar face")]
fn select_face_for_sketch(world: &mut ApplicationWorld) {
    let capture = capture_framebuffer(&mut world.app().window.native);
    let windows = diagnostics::window_occlusions();
    let point = solid_visibility::visible_fixture_point(&capture, &windows)
        .expect("expected a visible face");
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.click(point);
    world.pump(Vec::new());
    let state = world.state();
    let sketch = state.active_sketch.expect("expected a sketch on the face");
    let document = state.document.unwrap();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert!(matches!(
        snapshot.sketches.get(sketch).unwrap().placement(),
        SketchPlacement::Face(_)
    ));
}
#[then("FRONT view and empty Fully Constrained status are displayed")]
fn face_sketch_view(world: &mut ApplicationWorld) {
    viewport_orientation::expect_control_shows(world, "Front");
    assert_eq!(world.dimensions_needed(), 0);
    world.expect_status("Fully Constrained");
}
#[when("the user choose Finish Sketch")]
fn finish_face_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
}
#[then("Sketch2 is retained beside Extrusion1")]
fn retained_face_sketch(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(browser.sketches.iter().any(|row| row.name == "Sketch2"));
    assert!(browser.features.iter().any(|row| row.name == "Extrusion1"));
    assert_eq!(browser.features.len(), 1);
    world.expect_text("Sketch2");
    world.expect_text("Extrusion1");
}

#[test]
fn clicking_an_origin_plane_in_an_empty_view_starts_a_sketch() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    let point = world
        .app()
        .window
        .native
        .view
        .model_pixel([0., 0., 0.])
        .unwrap();
    world.click(point);
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert_eq!(world.dimensions_needed(), 0);
    world.close();
}
