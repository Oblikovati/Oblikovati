// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};

#[test]
fn browser_reads_current_document_and_owned_sketch_edit_state() {
    let (mut owner, awake) = owner();
    assert!(owner.gui_browser_snapshot(999).is_err());
    let document = create(&mut owner, &awake);
    let empty = owner.gui_browser_snapshot(document).unwrap();
    assert_eq!(empty.name, "Owner part");
    assert!(empty.has_origin && empty.sketches.is_empty() && empty.features.is_empty());
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    let editing = owner.gui_browser_snapshot(document).unwrap();
    assert_eq!(editing.sketches[0].id, sketch);
    assert!(editing.sketches[0].inspection.as_ref().unwrap().editing);
    owner.finish_sketch_edit().unwrap();
    assert!(
        !owner.gui_browser_snapshot(document).unwrap().sketches[0]
            .inspection
            .as_ref()
            .unwrap()
            .editing
    );
}

#[test]
fn browser_activation_uses_identity_with_duplicate_names_and_preserves_history() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let first = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let second = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xz()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let browser = owner.gui_browser_snapshot(document).unwrap();
    assert_eq!(browser.sketches[0].name, browser.sketches[1].name);
    assert_ne!(first, second);
    let history = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    assert!(
        owner
            .edit_existing_sketch_from_gui(document, second)
            .unwrap()
    );
    assert_eq!(owner.active_sketch_edit().unwrap().sketch, second);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        history
    );
}

#[test]
fn browser_activation_refuses_missing_and_inactive_document_identity() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    assert!(
        owner
            .edit_existing_sketch_from_gui(document, DocumentSketchId(999))
            .is_err()
    );
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    create(&mut owner, &awake);
    assert!(
        owner
            .edit_existing_sketch_from_gui(document, sketch)
            .is_err()
    );
    assert!(owner.active_sketch_edit().is_none());
}
