// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::workspace_layout::browser_feature::click_branch_arrow;
use super::super::super::world::ApplicationWorld;
use super::{activate_to, click_in_extrude, feature_count};
use oblikovati_model::feature::{
    FeatureId, part::PartFeatureDefinition, sketch_prism::SketchPrismExtent,
    work_plane::WorkPlaneReference,
};

#[test]
fn browser_plane_picks_preview_and_commit_reference_extent() {
    let mut world = ApplicationWorld::new();
    open_plane_fixture(&mut world);
    let before = feature_count(&mut world);
    enter_to_face(&mut world);

    pick_plane(&mut world, "SelectFace", "XY Plane", 0.);
    assert_eq!(feature_count(&mut world), before);
    pick_plane(&mut world, "WorkPlane", "Work Plane1", 1.);
    assert_eq!(feature_count(&mut world), before);

    let plane = authored_plane(&mut world);
    click_in_extrude(&mut world, "OK");
    assert_eq!(feature_count(&mut world), before + 1);
    assert_committed_extent(&mut world, plane);
    world.close();
}

fn authored_plane(world: &mut ApplicationWorld) -> FeatureId {
    let browser = world.state().browser.unwrap();
    let id = browser
        .features
        .iter()
        .find(|row| row.name == "Work Plane1")
        .unwrap()
        .id;
    assert_eq!(browser.selected_feature, Some(id));
    id
}

fn open_plane_fixture(world: &mut ApplicationWorld) {
    let copy = world.scratch_folder().join("face-extension-candidate.opd");
    std::fs::write(
        &copy,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/face-extension-candidate.opd"
        ),
    )
    .unwrap();
    world.open_fixture(&copy);
    if !world.frame.runs.iter().any(|row| row.text == "XYPlane") {
        click_branch_arrow(world, "Origin");
    }
}

fn enter_to_face(world: &mut ApplicationWorld) {
    world.click_text("Sketch2");
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.run_in_window("Extrude", "1Profile");
    world.click_text("XY Plane");
    activate_to(world);
    assert!(!world.run_is_enabled("OK"));
}

fn pick_plane(world: &mut ApplicationWorld, selector: &str, row: &str, end: f32) {
    click_in_extrude(world, selector);
    world.click_text(row);
    world.pump(Vec::new());
    assert_preview(world, end);
}

fn assert_committed_extent(world: &mut ApplicationWorld, authored_plane: FeatureId) {
    let document = world.active_id();
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let feature = session.snapshot().work_geometry.features().last().unwrap();
    let PartFeatureDefinition::SketchPrism(recipe) = feature.definition() else {
        panic!("expected a committed Extrusion");
    };
    assert_eq!(
        recipe.extent(),
        Some(&SketchPrismExtent::ToFace {
            to: WorkPlaneReference::Feature(authored_plane)
        })
    );
}

fn assert_preview(world: &mut ApplicationWorld, end: f32) {
    world.run_in_window("Extrude", "WorkPlane");
    assert!(world.run_is_enabled("OK"));
    let mut caps = [f32::INFINITY, f32::NEG_INFINITY];
    for point in world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .flat_map(|triangle| triangle.positions)
    {
        caps[0] = caps[0].min(point[2]);
        caps[1] = caps[1].max(point[2]);
    }
    assert!((caps[0] + 1.2).abs() < 1e-5, "{caps:?}");
    assert!((caps[1] - end).abs() < 1e-5, "{caps:?}");
}
