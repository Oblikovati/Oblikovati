// SPDX-License-Identifier: GPL-2.0-only
//! Native preselection of a work plane supported by a current solid face.
use super::super::{ApplicationWorld, diagnostics, workspace_layout::browser_feature};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::CreateWorkPlaneArgs};
use oblikovati_gui::GuiAuthoringContext;
use oblikovati_model::document_sketch::SketchPlacement;

#[given("Work Plane1 was created 5 mm offset from the planar face")]
fn face_work_plane(world: &mut ApplicationWorld) {
    browser_feature::open_extruded_part_for_second_sketch(world);
    world.add_finished_sketch();
    let bodies = Client::new(world.app().owner.document_host())
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap();
    let key = bodies[0]
        .faces
        .as_ref()
        .unwrap()
        .iter()
        .find(|face| (face.point.as_ref().unwrap()[2] - 2.).abs() < 1e-9)
        .unwrap()
        .key
        .clone();
    let plane = Client::new(world.app().owner.document_host())
        .work_planes()
        .create(&CreateWorkPlaneArgs {
            kind: "plane-offset".into(),
            refs: Some(vec![key]),
            offset: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(plane.name, "Work Plane1");
    world.pump(Vec::new());
    world.expect_text("Work Plane1");
}
#[when("the user preselect Work Plane1 and invoke Start 2D Sketch")]
fn select_work_plane(world: &mut ApplicationWorld) {
    world.click_text("Work Plane1");
    assert!(world.row_highlighted("Work Plane1"));
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
}
#[then("empty Sketch3 is created without another support click")]
fn work_plane_sketch_started(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##plane-selection").is_none());
    let state = world.state();
    assert_eq!(state.authoring, GuiAuthoringContext::Sketch);
    assert_eq!(state.dimensions_needed, Some(0));
    let browser = state.browser.unwrap();
    let plane = browser
        .features
        .iter()
        .find(|row| row.name == "Work Plane1")
        .unwrap()
        .id;
    let sketch = browser
        .sketches
        .iter()
        .find(|row| row.name == "Sketch3")
        .unwrap();
    assert_eq!(state.active_sketch, Some(sketch.id));
    assert!(sketch.inspection.as_ref().unwrap().healthy);
    let document = state.document.unwrap();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot.sketches.get(sketch.id).unwrap().placement(),
        SketchPlacement::WorkPlane(plane)
    );
    world.expect_text("Finish Sketch");
    world.expect_status("Fully Constrained");
}
#[then("Finish Sketch retains Sketch3 and Work Plane1")]
fn finish_work_plane_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
    world.expect_text("Sketch3");
    world.expect_text("Work Plane1");
}
