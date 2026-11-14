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
