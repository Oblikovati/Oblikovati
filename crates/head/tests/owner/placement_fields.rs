// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{rectangle_input::placement_owner, slot_input::deliver};
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportPhysicalKey};

fn field_key(key: ViewportPhysicalKey) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::PhysicalKey {
        key,
        layout_key: key,
        command: None,
        pressed: true,
        modifiers: Default::default(),
    })
}
fn enter_expression(owner: &mut TestOwner, expression: &str, key: ViewportPhysicalKey) {
    let mut events: Vec<ViewportEvent> = expression
        .chars()
        .map(|character| ViewportEvent::Input(ViewportInput::Text(character)))
        .collect();
    events.push(field_key(key));
    deliver(owner, &events);
}
fn parameter_rectangle_owner() -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, document, sketch) = placement_owner(SketchPlacementTool::Rectangle);
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            ..Default::default()
        })
        .unwrap();
    owner
        .start_sketch_placement(document, sketch, SketchPlacementTool::Rectangle)
        .unwrap();
    (owner, document, sketch)
}
#[test]
fn typed_rectangle_fields_commit_parameter_dimensions_in_one_history_step() {
    let (mut owner, document, sketch) = parameter_rectangle_owner();
    let before = owner.gui_command_state().unwrap();
    begin_rectangle_preview(&mut owner);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Tab);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Enter);
    verify_dimensions(&owner, document, sketch);
    deliver(
        &mut owner,
        &[key(oblikovati_viewport::ViewportKey::Undo, true)],
    );
    assert!(
        rectangle_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
}
fn begin_rectangle_preview(owner: &mut TestOwner) {
    deliver(
        owner,
        &[button([0, 0], true), button([0, 0], false), motion([4, 3])],
    );
}
fn rectangle_geometry(
    owner: &TestOwner,
    document: u64,
    sketch: DocumentSketchId,
) -> &oblikovati_model::sketch::Sketch2D {
    owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
}
#[test]
fn invalid_field_keeps_preview_and_history_until_corrected() {
    let (mut owner, document, sketch) = parameter_rectangle_owner();
    begin_rectangle_preview(&mut owner);
    let before = owner.gui_command_state().unwrap();
    enter_expression(&mut owner, "missing_size", ViewportPhysicalKey::Tab);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Enter);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert!(
        rectangle_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    assert!(owner.placement_preview().is_some());
    assert!(owner.sketch_preview().is_none());
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementFailed { cause }) if cause.contains("missing_size"))
    );
    enter_expression(&mut owner, "", ViewportPhysicalKey::Tab);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Enter);
    verify_dimensions(&owner, document, sketch);
}
#[test]
fn solved_preview_has_commit_geometry_and_does_not_reserve_live_identities() {
    let (mut owner, document, sketch) = parameter_rectangle_owner();
    begin_rectangle_preview(&mut owner);
    let before = owner.gui_command_state().unwrap();
    let mark = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .high_water();
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Tab);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Tab);
    let Some(oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry, members, ..
    }) = owner.sketch_preview()
    else {
        panic!("locked rectangle: expected solved private preview");
    };
    assert_eq!(members.len(), 4);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .high_water(),
        mark
    );
    assert!(
        rectangle_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    enter_expression(&mut owner, "", ViewportPhysicalKey::Enter);
    assert_eq!(
        rectangle_geometry(&owner, document, sketch).recipe(),
        geometry.recipe()
    );
}
#[test]
fn field_status_names_active_field_and_restart_does_not_reuse_locks() {
    let (mut owner, document, sketch) = parameter_rectangle_owner();
    begin_rectangle_preview(&mut owner);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Tab);
    assert!(
        matches!(owner.take_command_notice(), Some(DesktopCommandNotice::PlacementPrompt { message }) if message.contains("Width: size") && message.contains("[Height:"))
    );
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Enter);
    begin_rectangle_preview(&mut owner);
    deliver(&mut owner, &[button([4, 3], true)]);
    let geometry = rectangle_geometry(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 8);
    assert_eq!(geometry.dimensional_constraints().count(), 2);
}
#[test]
fn escape_clears_started_entry_before_canceling_the_placement_tool() {
    let (mut owner, document, sketch) = parameter_rectangle_owner();
    begin_rectangle_preview(&mut owner);
    enter_expression(&mut owner, "size", ViewportPhysicalKey::Escape);
    let cancel = key(oblikovati_viewport::ViewportKey::Cancel, true);
    assert!(owner.placement_active());
    deliver(
        &mut owner,
        &[ViewportEvent::Input(ViewportInput::Text('9')), cancel],
    );
    assert!(owner.placement_active());
    deliver(&mut owner, &[cancel]);
    assert!(!owner.placement_active());
    assert!(
        rectangle_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
}
fn verify_dimensions(owner: &TestOwner, document: u64, sketch: DocumentSketchId) {
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).unwrap().geometry();
    assert_eq!(geometry.lines().len(), 4);
    assert_eq!(geometry.dimensional_constraints().count(), 2);
    for (constraint, parameter) in geometry.dimensional_constraints() {
        assert_eq!(snapshot.parameters.get(parameter).unwrap().source, "size");
        assert!((geometry.measure_dimension(constraint.id).unwrap().value - 1.).abs() < 1e-9);
    }
}
