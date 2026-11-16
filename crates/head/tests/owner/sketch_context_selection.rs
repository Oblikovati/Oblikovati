// SPDX-License-Identifier: GPL-2.0-only
//! Context choice uses checked sketch selection (#4041).
use super::slot_input::deliver;
use super::*;
use crate::{HeadSketchWork, placement_test_inputs::button};
use oblikovati_model::sketch::{
    SketchEntityId, SketchPlane, SketchPointInput, SketchPrimitiveCreation,
};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};

fn overlapping_owner() -> (TestOwner, u64, SketchEntityId, SketchEntityId) {
    let (mut owner, _) = owner();
    let document = Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Overlap".into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.project_origin = false;
    let sketch = owner.create_sketch_for_edit(document, &request).unwrap();
    let mut commands = owner.document_commands(document);
    let point = commands
        .create_sketch_primitive(
            sketch,
            &SketchPrimitiveCreation::Point { position: [0., 0.] },
        )
        .unwrap()
        .entity;
    let line = commands
        .create_sketch_primitive(
            sketch,
            &SketchPrimitiveCreation::Line {
                endpoints: [
                    SketchPointInput::New([-10., 0.]),
                    SketchPointInput::New([10., 0.]),
                ],
                construction: false,
                centerline: false,
            },
        )
        .unwrap()
        .entity;
    (owner, document, point, line)
}
fn open_context(owner: &mut TestOwner) -> oblikovati_gui::GuiSketchContext {
    let event = ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Right,
        pressed: true,
        position: [0, 0],
        modifiers: Default::default(),
    });
    deliver(owner, &[event]);
    owner.gui_command_state().unwrap().sketch_context.unwrap()
}
#[test]
fn selecting_offered_curve_replaces_point_without_history() {
    let (mut owner, _, point, line) = overlapping_owner();
    deliver(&mut owner, &[button([0, 0], true)]);
    let before = owner.gui_command_state().unwrap();
    assert_eq!(before.sketch_selection.entities[0].1, point);
    let context = open_context(&mut owner);
    assert_eq!(
        context
            .candidates
            .iter()
            .map(|item| item.entity)
            .collect::<Vec<_>>(),
        [point, line]
    );
    assert!(
        owner
            .select_sketch_context_candidate(context.token, line)
            .unwrap()
    );
    let after = owner.gui_command_state().unwrap();
    assert_eq!(after.sketch_selection.entities[0].1, line);
    assert_eq!(before.undo_label, after.undo_label);
    assert_eq!(before.redo_label, after.redo_label);
    assert!(after.sketch_context.is_none());
    assert!(
        owner
            .select_sketch_context_candidate(context.token, line)
            .is_err()
    );
}
#[test]
fn stale_context_token_and_unoffered_entity_are_refused() {
    let (mut owner, _, point, line) = overlapping_owner();
    let first = open_context(&mut owner);
    let second = open_context(&mut owner);
    assert_ne!(first.token, second.token);
    assert!(
        owner
            .select_sketch_context_candidate(first.token, line)
            .is_err()
    );
    assert!(
        owner
            .select_sketch_context_candidate(second.token, SketchEntityId(u64::MAX))
            .is_err()
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .is_empty()
    );
    assert!(
        owner
            .select_sketch_context_candidate(second.token, point)
            .unwrap()
    );
}
#[test]
fn context_rejects_a_changed_document_generation() {
    let (mut owner, document, _, line) = overlapping_owner();
    let context = open_context(&mut owner);
    let edit = owner.active_sketch_edit().unwrap();
    owner
        .document_commands(document)
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Point {
                position: [30., 30.],
            },
        )
        .unwrap();
    assert!(
        owner
            .select_sketch_context_candidate(context.token, line)
            .is_err()
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .is_empty()
    );
}

#[test]
fn failed_new_hit_test_retires_the_old_context() {
    use super::slot_input::PlanarInputWindow;
    let (mut owner, _, _, line) = overlapping_owner();
    let old = open_context(&mut owner);
    let right = ViewportEvent::Input(ViewportInput::Button {
        button: ViewportButton::Right,
        pressed: true,
        position: [0, 0],
        modifiers: Default::default(),
    });
    let window = PlanarInputWindow {
        fail: true,
        ..Default::default()
    };
    owner.process_window_inputs(&[right], &window).unwrap();
    assert!(owner.gui_command_state().unwrap().sketch_context.is_none());
    assert!(
        owner
            .select_sketch_context_candidate(old.token, line)
            .is_err()
    );
}

#[test]
fn context_rejects_a_different_active_document() {
    let (mut owner, old_document, _, line) = overlapping_owner();
    let context = open_context(&mut owner);
    let new_document = Client::new(owner.document_host())
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Other".into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert_ne!(old_document, new_document);
    assert!(
        owner
            .select_sketch_context_candidate(context.token, line)
            .is_err()
    );
    let session = owner.documents().workspace().session(old_document).unwrap();
    let sketch = session.snapshot().sketches.sketches()[0].id();
    assert!(session.selected_sketch_picks(sketch).is_empty());
}

#[test]
fn context_rejects_a_different_active_sketch() {
    let (mut owner, document, _, line) = overlapping_owner();
    let context = open_context(&mut owner);
    owner.finish_sketch_edit().unwrap();
    let mut request = super::sketch_edit::request(SketchPlane::xy());
    request.name = "Other".into();
    request.project_origin = false;
    owner.create_sketch_for_edit(document, &request).unwrap();
    assert!(
        owner
            .select_sketch_context_candidate(context.token, line)
            .is_err()
    );
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_selection
            .entities
            .is_empty()
    );
}
#[test]
fn dismiss_only_retires_matching_context_and_rejects_replay() {
    let (mut owner, _, _, line) = overlapping_owner();
    let old = open_context(&mut owner);
    let current = open_context(&mut owner);
    assert!(!owner.retire_sketch_context(old.token));
    assert!(owner.retire_sketch_context(current.token));
    assert!(
        owner
            .select_sketch_context_candidate(current.token, line)
            .is_err()
    );
    let next = open_context(&mut owner);
    assert!(
        owner
            .select_sketch_context_candidate(next.token, line)
            .unwrap()
    );
    assert!(!owner.retire_sketch_context(next.token));
    assert_eq!(
        owner.gui_command_state().unwrap().sketch_selection.entities[0].1,
        line
    );
}
