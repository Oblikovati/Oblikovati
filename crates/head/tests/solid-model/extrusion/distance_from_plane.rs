// SPDX-License-Identifier: GPL-2.0-only
use super::super::{direction::preview_depth, distance_a_input};
use super::{choose_extent, click_in_extrude, feature_count};
use crate::HeadWork;
use cucumber::{given, then, when};
use oblikovati_model::feature::{
    FeatureId, part::PartFeatureDefinition, sketch_prism::SketchPrismExtent,
    work_plane::WorkPlaneReference,
};
use oblikovati_viewport::ViewportPhysicalKey;

use super::super::super::{scenario_root, world::ApplicationWorld};

const FIXTURE: &str = "fixtures/cad/extrusion/profile-from-plane-3mm.opd";

#[given("a fresh copy of `fixtures/cad/extrusion/profile-from-plane-3mm.opd` is open")]
fn open_part(world: &mut ApplicationWorld) {
    let copy = world.scratch_folder().join("profile-from-plane-3mm.opd");
    std::fs::copy(scenario_root().join(FIXTURE), &copy).unwrap();
    world.open_fixture(&copy);
}

#[given(
    "the user makes Sketch1 and Work Plane1 visible and selects the sketch's rectangular profile"
)]
fn start_from_rectangle(world: &mut ApplicationWorld) {
    let sketch = world.state().browser.unwrap().sketches.remove(0);
    assert_eq!(sketch.name, "Sketch1");
    assert!(sketch.visible);
    world.click_text("Sketch1");
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "1Profile");
}

#[given("Work Plane1 is parallel to and 3 mm from the sketch plane in the default direction")]
fn offset_plane_is_visible(world: &mut ApplicationWorld) {
    let update = world.app().owner.datum_planes().unwrap();
    assert_eq!(update.planes.len(), 1);
    assert!(update.planes[0].corners.iter().all(|point| point[2] == 0.3));
    assert_caps(world, [0., 1.]);
}

#[when("the user selects that work plane as From and specifies a distance of 8 mm")]
fn select_eight_millimeter_extent(world: &mut ApplicationWorld) {
    choose_extent(world, "SketchPlane", "ReferencePlane");
    assert!(!world.run_is_enabled("OK"));
    click_in_extrude(world, "SelectFace");
    world.click_text("Work Plane1");
    world.click(distance_a_input(world));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("8 mm");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "WorkPlane");
}

#[then("a valid extrusion starts at the work plane")]
fn preview_starts_at_plane(world: &mut ApplicationWorld) {
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(world), 1);
    assert!((preview_depth(world)[0] - 0.3).abs() < 1e-6);
}

#[then("the other cap is 8 mm farther in the default direction")]
fn preview_ends_at_distance(world: &mut ApplicationWorld) {
    assert_caps(world, [0.3, 1.1]);
    world.save_capture("extrude-from-plane-preview");
}

fn assert_caps(world: &mut ApplicationWorld, caps: [f32; 2]) {
    let actual = preview_depth(world);
    for (actual, expected) in actual.into_iter().zip(caps) {
        assert!(
            (actual - expected).abs() < 1e-6,
            "{actual} against {expected}"
        );
    }
}

#[test]
fn numeric_extrusion_from_plane_commits_and_reopens() {
    let mut world = ApplicationWorld::new();
    let copy = world.scratch_folder().join("profile-from-plane-3mm.opd");
    let fixture = include_bytes!(
        "../../../../../test-utilities/fixtures/extrusion/profile-from-plane-3mm.opd"
    );
    std::fs::write(&copy, fixture).unwrap();
    world.open_fixture(&copy);
    start_from_rectangle(&mut world);
    offset_plane_is_visible(&mut world);
    select_eight_millimeter_extent(&mut world);
    preview_starts_at_plane(&mut world);
    preview_ends_at_distance(&mut world);
    click_in_extrude(&mut world, "OK");
    let id = world.active_id();
    let path = world.save_part_to_file(id, "saved-offset-extrusion.opd");
    assert_committed_feature(&mut world);
    let close = oblikovati_api::wire::CloseDocumentArgs { id, force: false };
    assert!(world.app().owner.close_document_from_gui(&close).unwrap());
    world.pump(Vec::new());
    assert_ne!(world.state().document, Some(id));
    world.open_fixture(&path);
    assert_ne!(world.active_id(), id);
    assert_committed_feature(&mut world);
    world.close();
}

fn assert_committed_feature(world: &mut ApplicationWorld) {
    assert_eq!(feature_count(world), 2);
    assert_caps(world, [0.3, 1.1]);
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let feature = session.snapshot().work_geometry.features().last().unwrap();
    let PartFeatureDefinition::SketchPrism(prism) = feature.definition() else {
        panic!("expected a committed Extrusion");
    };
    assert_eq!(
        prism.extent(),
        Some(&SketchPrismExtent::DistanceFromPlane {
            from: WorkPlaneReference::Feature(FeatureId(1)),
        })
    );
    assert_eq!(prism.recipe().version, 4);
}
