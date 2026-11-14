// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn extrusion_requested_sketch_resumes_and_reports_an_unconstrained_profile() {
    let mut world = ApplicationWorld::new();
    let document = world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
    world.draw_rectangle();
    world.finish_sketch();
    world.pump(Vec::new());
    world.expect_text("Profiles");
    world.expect_text("Distance A");
    world.expect_text("10 mm");
    world.expect_text("degrees of freedom 4: expected 0");
    world.expect_no_text("1 Profile");
    assert!(!world.state().extrude_preview_active);
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert!(session.snapshot().work_geometry.features().is_empty());
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert!(!world.state().extrude_preview_active);
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert!(session.snapshot().work_geometry.features().is_empty());
    world.close();
}

#[test]
fn extrusion_requested_constrained_sketch_shows_and_cancels_preview() {
    let mut world = ApplicationWorld::new();
    let document = world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
    world.draw_rectangle();
    constrain_rectangle(&mut world, document);
    world.finish_sketch();
    world.pump(Vec::new());
    world.expect_text("Profiles");
    world.expect_text("1 Profile");
    world.expect_text("Distance A");
    world.expect_text("10 mm");
    assert!(world.state().extrude_preview_active);
    assert!(!world.app().window.scene_colors.is_empty());
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::diagnostics::window_occlusions();
    assert!(super::solid_visibility::visible_preview_fill(
        &capture, &windows
    ));
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert!(!world.state().extrude_preview_active);
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert!(session.snapshot().work_geometry.features().is_empty());
    world.close();
}

pub(super) fn constrain_rectangle(world: &mut ApplicationWorld, document: u64) {
    use oblikovati_api::{
        client::Client,
        wire::{AddConstraintArgs, AddDimensionArgs},
    };
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    let sketch = session.snapshot().sketches.sketches().first().unwrap();
    let sketch_id = sketch.id();
    let geometry = sketch.geometry();
    let [first, second, ..] = geometry.lines() else {
        panic!("expected rectangle lines")
    };
    let points = [first.endpoints[0], first.endpoints[1], second.endpoints[1]];
    let positions = points.map(|id| geometry.point(id).unwrap().position);
    let distances = [
        (positions[1][0] - positions[0][0]).abs(),
        (positions[2][1] - positions[1][1]).abs(),
    ];
    let owner = &mut world.app().owner;
    Client::new(owner.document_host())
        .constrain()
        .add(&AddConstraintArgs {
            sketch_index: 0,
            kind: "fix".into(),
            entities: Some(vec![points[0].0]),
            ..Default::default()
        })
        .unwrap();
    for (pair, (orientation, distance)) in points
        .windows(2)
        .zip(["horizontal", "vertical"].into_iter().zip(distances))
    {
        Client::new(owner.document_host())
            .dimension()
            .add(&AddDimensionArgs {
                sketch_index: 0,
                kind: "distance".into(),
                entities: Some(pair.iter().map(|id| id.0).collect()),
                expression: format!("{distance:.17} cm"),
                orientation: orientation.into(),
                ..Default::default()
            })
            .unwrap();
    }
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .analyze_sketch(sketch_id)
            .unwrap()
            .degrees_of_freedom,
        0
    );
}

#[test]
fn ordinary_sketch_finish_does_not_open_extrusion() {
    let mut world = ApplicationWorld::new();
    world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.start_sketch();
    world.draw_rectangle();
    world.finish_sketch();
    world.pump(Vec::new());
    world.expect_no_text("Profiles");
    assert!(!world.state().extrude_preview_active);
    world.close();
}

#[test]
fn extrusion_distance_zero_disables_commit_and_accepts_recovery() {
    let (mut world, document) = requested_extrusion_fixture();
    for invalid in ["0", "-1 mm", "2 deg", "missing"] {
        set_extrusion_distance(&mut world, invalid);
        assert_invalid_distance(&mut world, invalid);
    }
    world.key(oblikovati_viewport::ViewportPhysicalKey::Enter, false);
    world.expect_text("Distance A");
    for valid in ["8", "1 mm + 2 mm", "depth"] {
        set_extrusion_distance(&mut world, valid);
        assert_valid_distance(&mut world, valid);
    }
    check_changed_parameter(&mut world);
    let generation = document_generation(&mut world, document);
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert_no_feature(&mut world, document, generation);
    world.close();
}

fn requested_extrusion_fixture() -> (ApplicationWorld, u64) {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    let mut world = ApplicationWorld::new();
    let document = world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.click_text("Extrude");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
    world.draw_rectangle();
    constrain_rectangle(&mut world, document);
    Client::new(world.app().owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "depth".into(),
            expression: "8 mm".into(),
            ..Default::default()
        })
        .unwrap();
    world.finish_sketch();
    world.pump(Vec::new());
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
    (world, document)
}
fn check_changed_parameter(world: &mut ApplicationWorld) {
    set_depth_expression(world, "0 mm");
    assert_invalid_distance(world, "depth");
    set_depth_expression(world, "8 mm");
    assert_valid_distance(world, "depth");
}
fn set_depth_expression(world: &mut ApplicationWorld, expression: &str) {
    use oblikovati_api::{client::Client, wire::ParameterSetArgs};
    Client::new(world.app().owner.document_host())
        .parameters()
        .set(&ParameterSetArgs {
            name: "depth".into(),
            expression: expression.into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}
fn set_extrusion_distance(world: &mut ApplicationWorld, value: &str) {
    let label = world.run("Distance A").clone();
    world.click([label.minimum[0] as i32 - 30, label.center()[1] as i32]);
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
    assert_eq!(
        world.state().extrude_distance_validation.unwrap().distance,
        value
    );
}
fn assert_invalid_distance(world: &mut ApplicationWorld, value: &str) {
    assert!(
        !world.run_is_enabled("OK"),
        "invalid {value:?} must disable OK"
    );
    let warning = world.run("Invalid distance");
    assert!(
        warning.color[0] > warning.color[1] * 2,
        "warning must be red"
    );
    assert!(
        world
            .state()
            .extrude_distance_validation
            .unwrap()
            .error
            .is_some()
    );
}
fn assert_valid_distance(world: &mut ApplicationWorld, value: &str) {
    assert!(world.run_is_enabled("OK"), "valid {value:?} must enable OK");
    world.expect_no_text("Invalid distance");
    assert!(
        world
            .state()
            .extrude_distance_validation
            .unwrap()
            .error
            .is_none()
    );
}

#[test]
fn rejected_extrusion_retains_draft_and_success_commits_once() {
    let (mut world, document) = requested_extrusion_fixture();
    let generation = document_generation(&mut world, document);
    set_extrusion_profile(&mut world, "0", "99");
    world.click_text("OK");
    world.pump(Vec::new());
    world.expect_text("Extrusion failed");
    world.expect_text("Distance A");
    assert_no_feature(&mut world, document, generation);
    set_extrusion_profile(&mut world, "99", "0");
    world.click_text("OK");
    world.pump(Vec::new());
    world.expect_no_text("Distance A");
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert_eq!(session.snapshot().work_geometry.features().len(), 1);
    world.close();
}
fn document_generation(world: &mut ApplicationWorld, document: u64) -> u64 {
    world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .generation()
}
fn assert_no_feature(world: &mut ApplicationWorld, document: u64, generation: u64) {
    let session = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap();
    assert_eq!(session.generation(), generation, "draft changed history");
    assert!(session.snapshot().work_geometry.features().is_empty());
}
fn set_extrusion_profile(world: &mut ApplicationWorld, old: &str, value: &str) {
    let at = world.run_in_window("Extrude", old).center();
    world.click(super::diagnostics::pointer_pixels(at));
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text(value);
    world.pump(Vec::new());
}
