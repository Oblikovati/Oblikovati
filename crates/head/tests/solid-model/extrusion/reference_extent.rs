// SPDX-License-Identifier: GPL-2.0-only
use super::acceptance::feature_count;
use super::viewport_interaction::{click_text_native, replace_distance_native};
use super::*;

#[path = "between_faces.rs"]
mod between_faces;

#[path = "plane_reference.rs"]
mod plane_reference;

#[path = "distance_from_plane.rs"]
mod distance_from_plane;

#[path = "preset_visibility.rs"]
mod preset_visibility;

#[path = "preview_visibility.rs"]
mod preview_visibility;

#[given("a sketch-plane start and numeric distance")]
#[given("the numeric Distance A extrusion preview")]
fn numeric_preview(world: &mut ApplicationWorld) {
    selected_rectangle_preview(world);
    world.run_in_window("Extrude", "SketchPlane");
    assert!(world.run_is_enabled("OK"));
}

#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with a selected profile in an active Extrude preview"
)]
#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with a selected profile and numeric Distance A preview"
)]
#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with an uncommitted rectangular Extrude preview"
)]
#[given(
    "a copy of `fixtures/cad/revolve/revolve-profile.opd` is open with a selected rectangular profile and Default direction"
)]
pub(super) fn native_fixture_preview(world: &mut ApplicationWorld) {
    let copy = world.scratch_folder().join("revolve-profile.opd");
    let source = super::super::scenario_root().join("fixtures/cad/revolve/revolve-profile.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    world.click_text("Extrude");
    world.pump(Vec::new());
    waiting_extrusion_shows_profile(world);
    waiting_extrusion_shows_default_preview(world);
}

#[when("the From reference control is activated")]
fn activate_from(world: &mut ApplicationWorld) {
    choose_extent(world, "SketchPlane", "From/To");
}

#[then("missing From and To references show Select Face")]
fn both_faces_required(world: &mut ApplicationWorld) {
    let rows = request_rows(world);
    world.save_capture("from-to-required");
    assert_eq!(rows.len(), 2, "{}", world.window_text("Extrude"));
    assert!(rows[0][1] < rows[1][1]);
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
    choose_extent(world, "From/To", "SketchPlane");
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
    choose_extent(world, "Distance", "ToFace");
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

fn choose_extent(world: &mut ApplicationWorld, shown: &str, choice: &str) {
    click_in_extrude(world, shown);
    let row = world
        .frame
        .runs
        .iter()
        .find(|row| row.text == choice && world.frame.is_popup_row(row))
        .unwrap_or_else(|| panic!("expected {choice} in the extent popup"));
    let point = super::super::diagnostics::pointer_pixels(row.center());
    world.save_capture(&format!("extent-choose-{choice}"));
    super::viewport_interaction::click_at(world, point);
    world.pump(Vec::new());
}

#[test]
fn from_dropdown_restores_the_numeric_preview() {
    let mut world = ApplicationWorld::new();
    numeric_preview(&mut world);
    activate_from(&mut world);
    both_faces_required(&mut world);
    missing_faces_disable_extrusion(&mut world);
    restore_numeric(&mut world);
    numeric_preview_returns(&mut world);
    world.save_capture("extrude-extent-dropdowns");
    world.close();
}

fn click_in_extrude(world: &mut ApplicationWorld, label: &str) {
    let point = world.run_in_window("Extrude", label).center();
    super::viewport_interaction::click_at(world, super::super::diagnostics::pointer_pixels(point));
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
    click_in_extrude(&mut world, "SelectFace");
    pick_top_face(&mut world);
    assert_eq!(world.state().selected_face_appearances.len(), 1);
    assert!(world.state().extrude_draft_preview.is_some());
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(&mut world), 1);
    click_in_extrude(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 2);
    assert_committed_extent(&mut world);
    world.close();
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
    super::viewport_interaction::click_at(world, pixel);
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
