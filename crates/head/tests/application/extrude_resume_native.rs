// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn extrusion_requested_free_sketch_shows_and_cancels_preview() {
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
    world.expect_text("1 Profile");
    assert!(world.state().extrude_preview_active);
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

#[test]
fn extrusion_preview_tracks_distance_and_parameter_changes() {
    let (mut world, document) = requested_extrusion_fixture();
    let initial = preview_height(&mut world);
    let stale = world
        .app()
        .window
        .gui_actions
        .iter()
        .find_map(|action| match action {
            oblikovati_gui::GuiShellAction::PreviewExtrude(Some(request)) => Some(request.clone()),
            _ => None,
        })
        .expect("expected the initial preview request");
    set_extrusion_distance(&mut world, "8 mm");
    let eight = preview_height(&mut world);
    assert!(
        eight < initial,
        "preview did not shrink: {initial} to {eight}"
    );
    set_extrusion_distance(&mut world, "0");
    assert!(!world.state().extrude_preview_active);
    assert!(world.app().window.scene_triangles.is_empty());
    set_extrusion_distance(&mut world, "depth");
    assert!((preview_height(&mut world) - eight).abs() < 0.0001);
    edit_depth_in_parameter_window(&mut world);
    let updated = preview_height(&mut world);
    assert!(updated > eight * 1.9);
    assert!(
        !world
            .app()
            .owner
            .extrusion_preview_from_gui(Some(&stale))
            .unwrap()
    );
    world.pump(Vec::new());
    assert_eq!(preview_height(&mut world), updated);
    let generation = document_generation(&mut world, document);
    world.click_text("Cancel");
    world.pump(Vec::new());
    assert!(!world.state().extrude_preview_active);
    assert!(world.app().window.scene_triangles.is_empty());
    assert_no_feature(&mut world, document, generation);
    world.close();
}
fn preview_height(world: &mut ApplicationWorld) -> f32 {
    assert!(world.state().extrude_preview_active);
    world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
        .map(|position| position[2])
        .fold(0.0_f32, f32::max)
}

fn edit_depth_in_parameter_window(world: &mut ApplicationWorld) {
    world.click(super::toolbar_icon_center("Parameters"));
    let (position, size) = super::diagnostics::window_geometry("Parameters").unwrap();
    let name = super::rendered_label_in_rect(
        "depth",
        position,
        [position[0] + 200., position[1] + size[1]],
    );
    let equation_x = super::diagnostics::pointer_pixels([position[0] + 290., position[1]])[0];
    world.click([equation_x, name[1]]);
    world.key(oblikovati_viewport::ViewportPhysicalKey::A, true);
    world.type_text("16 mm");
    world.click_text("Done");
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .parameters
            .iter()
            .any(|row| row.name == "depth" && row.expression == "16 mm")
    );
}

#[test]
fn extrude_shortcut_reopens_after_cancel_without_intercepting_text() {
    let (mut world, document) = requested_extrusion_fixture();
    let field = world.run("Distance A").clone();
    world.click([field.minimum[0] as i32 - 30, field.center()[1] as i32]);
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.type_text("e");
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .extrude_distance_validation
            .unwrap()
            .distance
            .ends_with('e')
    );
    world.click_text("Cancel");
    world.pump(Vec::new());
    world.expect_no_text("Distance A");
    let generation = document_generation(&mut world, document);
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.pump(Vec::new());
    world.expect_text("Distance A");
    assert_no_feature(&mut world, document, generation);
    world.close();
}

#[test]
fn remapped_extrude_shortcut_respects_focus() {
    use oblikovati_api::{client::Client, wire::SetChordArgs};
    let (mut world, _) = requested_extrusion_fixture();
    world.click_text("Cancel");
    Client::new(world.app().owner.document_host())
        .keymap()
        .set_chord(&SetChordArgs {
            action_id: "Create.Extrude".into(),
            chord: "Ctrl+E".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.expect_no_text("Distance A");
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(false)]);
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, true);
    world.expect_no_text("Distance A");
    world.pump(vec![oblikovati_viewport::ViewportEvent::Focus(true)]);
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, true);
    world.pump(Vec::new());
    world.expect_text("Distance A");
    world.close();
}

#[test]
fn extrusion_shortcut_leaves_sketch_alias_input_available() {
    let mut world = ApplicationWorld::new();
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.expect_no_text("XY Plane");
    world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.start_sketch();
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.type_text("e");
    world.pump(Vec::new());
    assert_eq!(world.state().sketch_alias.as_deref(), Some("e"));
    world.expect_no_text("Distance A");
    world.close();
}

#[test]
fn extrusion_shortcut_on_empty_part_starts_plane_selection() {
    let mut world = ApplicationWorld::new();
    world.add_template(oblikovati_gui::GuiTemplateKind::Part);
    world.key(oblikovati_viewport::ViewportPhysicalKey::E, false);
    world.pump(Vec::new());
    world.expect_text("XY Plane");
    world.click_text("XY Plane");
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    world.close();
}
