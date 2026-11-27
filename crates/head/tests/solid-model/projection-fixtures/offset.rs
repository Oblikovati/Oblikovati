// SPDX-License-Identifier: GPL-2.0-only
//! The source solid and offset plane are prepared through native controls.
use super::*;

#[given("Part7 contains the rectangular source solid")]
fn rectangular_source_in_part_seven(world: &mut ApplicationWorld) {
    for _ in 0..6 {
        world.add_part();
    }
    super::select_metric_template(world);
    super::create_projection_profile(world);
    super::accept_projection_extrusion(world);
    super::projection_source_exists(world);
    let document = world.active_id();
    assert_eq!(world.document_name(document), "Part7");
}

#[when("Offset from Plane is based on XZ Plane")]
fn choose_xz_plane_for_offset(world: &mut ApplicationWorld) {
    world.click_text("Plane");
    world.expect_text("Offset Plane");
    world.click_text("XZ Plane");
    world.expect_text("1 Plane");
}

#[when("its offset is set to 50 mm and accepted")]
fn accept_fifty_millimetre_offset(world: &mut ApplicationWorld) {
    world.click_text("10 mm");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("50 mm");
    world.click_text("OK");
    world.pump(Vec::new());
}

#[then("Work Plane1 is available for a sketch outside the source solid")]
fn inspect_projection_offset(world: &mut ApplicationWorld) {
    let planes = oblikovati_api::client::Client::new(world.app().owner.document_host())
        .work_planes()
        .list()
        .unwrap()
        .planes
        .unwrap();
    let plane = planes
        .iter()
        .find(|plane| plane.name == "Work Plane1")
        .expect("expected Work Plane1");
    let origin = plane.origin.as_ref().expect("expected evaluated origin");
    assert!(
        (origin[1].abs() - 5.0).abs() < 1e-9,
        "expected 50 mm offset, got {origin:?}"
    );
    assert!(plane.healthy, "expected healthy work plane");
    world.expect_text("Work Plane1");
    world.click_text("Work Plane1");
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    let state = world.state();
    assert_eq!(state.authoring, oblikovati_gui::GuiAuthoringContext::Sketch);
    let plane_id = state
        .browser
        .unwrap()
        .features
        .iter()
        .find(|feature| feature.name == "Work Plane1")
        .unwrap()
        .id;
    let document = world.active_id();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert_eq!(
        session
            .snapshot()
            .sketches
            .get(edit.sketch)
            .unwrap()
            .placement(),
        oblikovati_model::document_sketch::SketchPlacement::WorkPlane(plane_id)
    );
}
