// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};

#[given("an existing solid has a circular profile on its top face")]
fn block_has_a_circular_top_profile(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
    let profile = DocumentSketchId(2);
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.id == profile && sketch.visible)
    );
    let workspace = world.app().owner.documents().workspace();
    let snapshot = workspace
        .session(workspace.active_document().unwrap())
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot
            .resolve_sketch_plane(profile)
            .unwrap()
            .to_model([0., 0.])
            .unwrap()[2],
        2.
    );
    assert_eq!(
        snapshot
            .sketches
            .get(profile)
            .unwrap()
            .geometry()
            .circle(SketchEntityId(36))
            .unwrap()
            .radius,
        0.5
    );
}

#[given("a straight path extending upward from the profile")]
fn straight_path_extends_upward(world: &mut ApplicationWorld) {
    let workspace = world.app().owner.documents().workspace();
    let snapshot = workspace
        .session(workspace.active_document().unwrap())
        .unwrap()
        .snapshot();
    let path = DocumentSketchId(3);
    let plane = snapshot.resolve_sketch_plane(path).unwrap();
    let geometry = snapshot.sketches.get(path).unwrap().geometry();
    let line = geometry.line(SketchEntityId(40)).unwrap();
    let ends = line.endpoints.map(|id| {
        plane
            .to_model(geometry.point(id).unwrap().position)
            .unwrap()
    });
    assert_eq!(ends, [[-2., -1.25, 2.], [-2., -1.25, 5.]]);
}

#[when("Sweep is invoked and New Solid is selected")]
fn create_separate_solid(world: &mut ApplicationWorld) {
    invoke_sweep(world);
    click_panel(world, "NewSolid");
}

#[then("Body Name shows Solid2 and OK is enabled")]
fn new_solid_name_is_shown(world: &mut ApplicationWorld) {
    let panel = super::super::world::squeezed(&world.window_text("Sweep"));
    assert!(panel.contains("BodyNameSolid2"), "{panel}");
    assert!(world.run_is_enabled("OK"));
    assert!(!world.state().sweep_preview.unwrap().request.join);
}

#[then("Sweep1 and Solid Bodies(2) appear")]
fn two_solid_bodies_are_shown(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 2);
    assert!(
        browser
            .bodies
            .iter()
            .any(|body| body.name == "Solid2" && body.faces == 3)
    );
    world.expect_text("Sweep1");
    world.expect_text("Solid Bodies");
}

#[when("Sweep1 is edited to Join and accepted")]
fn edit_sweep_to_join(world: &mut ApplicationWorld) {
    let original = world
        .state()
        .browser
        .unwrap()
        .bodies
        .into_iter()
        .find(|body| body.name == "Solid1")
        .unwrap();
    reopen_sweep(world);
    click_panel(world, "Join");
    let draft = world.state().sweep_preview.unwrap();
    assert_eq!(draft.result, GuiSweepPreview::Ready);
    assert!(draft.request.join);
    assert!(world.run_is_enabled("OK"));
    accept(world);
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].key, original.key);
    assert!(browser.sweep_edits[&browser.bodies[0].feature].join);
}

#[then("the raised boss remains and Solid Bodies(1) is shown")]
fn joined_part_has_one_body_with_the_boss(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 8);
    let top = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions.map(|point| point[2]))
        .fold(f32::NEG_INFINITY, f32::max);
    assert!(
        (top - 5.).abs() < 0.01,
        "expected the raised Sweep cap at 5 cm, got {top}"
    );
    world.expect_text("Sweep1");
    world.expect_text("Solid Bodies");
}
