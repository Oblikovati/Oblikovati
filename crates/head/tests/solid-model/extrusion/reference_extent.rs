// SPDX-License-Identifier: GPL-2.0-only
use super::acceptance::feature_count;
use super::viewport_interaction::{click_text_native, replace_distance_native};
use super::*;

#[given("a sketch-plane start and numeric distance")]
#[given("the numeric Distance A extrusion preview")]
fn numeric_preview(world: &mut ApplicationWorld) {
    selected_rectangle_preview(world);
    world.run_in_window("Extrude", "SketchPlane");
    assert!(world.run_is_enabled("OK"));
}

#[when("the From reference control is activated")]
fn activate_from(world: &mut ApplicationWorld) {
    click_in_extrude(world, "SketchPlane");
}

#[then("missing From and To references show Select Face")]
fn both_faces_required(world: &mut ApplicationWorld) {
    let rows = request_rows(world);
    assert_eq!(rows.len(), 2);
    assert!(rows[0][1] < rows[1][1]);
    world.save_capture("from-to-required");
}

#[then("OK is disabled without a shaded solid preview")]
#[then("no reference means disabled OK")]
fn missing_faces_disable_extrusion(world: &mut ApplicationWorld) {
    assert!(!world.run_is_enabled("OK"));
    assert!(!world.state().extrude_preview_active);
    assert!(world.app().window.scene_triangles.is_empty());
    assert_eq!(feature_count(world), 0);
}

#[when("the reference control is restored")]
fn restore_numeric(world: &mut ApplicationWorld) {
    click_in_extrude(world, "From/To");
}

#[then("Sketch Plane and the numeric distance preview return")]
fn numeric_preview_returns(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "SketchPlane");
    world.run_in_window("Extrude", "DistanceA");
    assert!(world.run_is_enabled("OK"));
    waiting_extrusion_shows_default_preview(world);
}

#[when("the user activates the termination control")]
fn activate_to(world: &mut ApplicationWorld) {
    click_in_extrude(world, "Distance");
}

#[then("a To reference selector appears")]
fn to_face_required(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "ToFace");
    assert_eq!(request_rows(world).len(), 1);
    world.save_capture("to-face-required");
}

fn request_rows(world: &ApplicationWorld) -> Vec<[f32; 2]> {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "SelectFace")
        .map(|run| run.center())
        .collect()
}

fn click_in_extrude(world: &mut ApplicationWorld, label: &str) {
    let point = world.run_in_window("Extrude", label).center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}

#[test]
fn viewport_face_pick_previews_and_commits_reference_extent() {
    let mut world = ApplicationWorld::new();
    new_metric_part(&mut world);
    super::viewport_interaction::committed_extrusion(&mut world);
    click_text_native(&mut world, "Extrude");
    replace_distance_native(&mut world, "5");
    activate_to(&mut world);
    assert!(!world.run_is_enabled("OK"));
    move_extrude_aside(&mut world);
    click_in_extrude(&mut world, "SelectFace");
    pick_top_face(&mut world);
    assert_eq!(world.state().selected_face_appearances.len(), 1);
    assert!(world.state().extrude_reference_preview.is_some());
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(&mut world), 1);
    click_in_extrude(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 2);
    assert_committed_extent(&mut world);
    world.close();
}

fn move_extrude_aside(world: &mut ApplicationWorld) {
    let (position, _) = super::super::diagnostics::window_geometry("Extrude").unwrap();
    let from = super::super::diagnostics::pointer_pixels([position[0] + 120., position[1] + 8.]);
    world.drag(from, [from[0] + 340, from[1] - 150]);
}

fn pick_top_face(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let geometry = session.snapshot().sketches.sketches()[0].geometry();
    let [x, y] = geometry.lines().iter().fold([0.; 2], |sum, line| {
        let point = geometry.point(line.endpoints[0]).unwrap().position;
        [sum[0] + point[0] / 4., sum[1] + point[1] / 4.]
    });
    let pixel = world
        .app()
        .window
        .native
        .view
        .model_pixel([x, y, 1.])
        .unwrap();
    super::super::x11_input::click(pixel);
    world.pump(Vec::new());
}

fn assert_committed_extent(world: &mut ApplicationWorld) {
    use oblikovati_model::feature::{part::PartFeatureDefinition, sketch_prism::SketchPrismExtent};
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let feature = &session.snapshot().work_geometry.features()[1];
    let PartFeatureDefinition::SketchPrism(prism) = feature.definition() else {
        panic!("expected a committed prism");
    };
    assert!(matches!(
        prism.extent(),
        Some(SketchPrismExtent::ToFace { .. })
    ));
}
