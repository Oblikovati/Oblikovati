// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};

#[path = "browser/pick_priority.rs"]
mod pick_priority;

#[test]
fn selection_modes_filter_viewport_faces_without_blocking_measure() {
    use oblikovati_gui::{GuiInspectAction, GuiSelectionMode};
    let (mut owner, _) = owner();
    let document = super::part_scene::open_prism(&mut owner, false);
    owner.take_scene().unwrap();
    let line = [[1., 1., 10.], [0., 0., -1.]];
    assert!(owner.click_model(document, line).unwrap());
    assert!(owner.model_face.is_some());
    assert!(
        owner
            .set_selection_mode_from_gui(GuiSelectionMode::Components)
            .unwrap()
    );
    assert!(!owner.click_model(document, line).unwrap());
    assert!(owner.model_face.is_some());
    owner
        .inspect_from_gui(GuiInspectAction::StartMeasure)
        .unwrap();
    assert!(owner.click_model(document, line).unwrap());
    assert!(
        owner
            .gui_measure_panel()
            .unwrap()
            .rows
            .iter()
            .any(|row| row.contains("Area"))
    );
    assert!(
        owner
            .set_selection_mode_from_gui(GuiSelectionMode::Faces)
            .unwrap()
    );
    assert!(owner.click_model(document, line).unwrap());
    assert!(owner.model_face.is_some());
}

#[test]
fn selection_mode_respects_the_measure_owner_document() {
    use oblikovati_gui::{GuiInspectAction, GuiSelectionMode};
    let (mut owner, awake) = owner();
    assert!(
        owner
            .set_selection_mode_from_gui(GuiSelectionMode::Components)
            .is_err()
    );
    super::part_scene::open_prism(&mut owner, false);
    owner
        .set_selection_mode_from_gui(GuiSelectionMode::Components)
        .unwrap();
    owner
        .inspect_from_gui(GuiInspectAction::StartMeasure)
        .unwrap();
    let other = create(&mut owner, &awake);
    assert!(
        !owner
            .click_model(other, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
    );
}

#[test]
fn occurrence_selection_rejects_stale_paths_without_editing_history() {
    let (mut owner, _) = owner();
    let path = std::path::PathBuf::from("/fixtures/browser-assembly.oad");
    owner.documents().files().saved.borrow_mut().insert(path.clone(),
        b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: Same, virtual: true}, {name: Same, virtual: true, virtualPartNumber: '2'}]}\n".to_vec());
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_string_lossy().into_owned(),
            ..Default::default()
        })
        .unwrap()
        .id;
    let browser = owner.gui_browser_snapshot(document).unwrap();
    let first = browser.occurrences[0].path.clone();
    let second = browser.occurrences[1].path.clone();
    assert_ne!(first, second);
    let history = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    owner
        .browser_choice(
            document,
            &oblikovati_gui::GuiBrowserChoice::Occurrence(second.clone()),
        )
        .unwrap();
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_occurrence,
        Some(second.clone())
    );
    owner
        .browser_choice(document, &oblikovati_gui::GuiBrowserChoice::Root)
        .unwrap();
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_occurrence,
        None
    );
    owner
        .browser_choice(
            document,
            &oblikovati_gui::GuiBrowserChoice::Occurrence(second.clone()),
        )
        .unwrap();
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        history
    );
    owner
        .document_commands(document)
        .run("Swap occurrences", |snapshot, _| -> Result<(), ()> {
            snapshot.assembly.as_mut().unwrap().swap(0, 1);
            Ok(())
        })
        .unwrap();
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_occurrence,
        None
    );
    assert!(
        owner
            .browser_choice(
                document,
                &oblikovati_gui::GuiBrowserChoice::Occurrence(second)
            )
            .is_err()
    );
}

#[test]
fn an_unpickable_assembly_placement_reports_a_selection_failure() {
    let (mut owner, _) = owner();
    super::part_scene::open_prism(&mut owner, false);
    let path = std::path::PathBuf::from("/fixtures/singular-assembly.oad");
    let component = std::env::temp_dir().join("head-scene.opd");
    let package = format!(
        "schemaVersion: 2\ndocumentType: 2\nmodel:\n  occurrences:\n    - name: Part:1\n      component: {}\n      transform: [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1]\n",
        component.display()
    );
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), package.into_bytes());
    let document = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_string_lossy().into_owned(),
            ..Default::default()
        })
        .unwrap()
        .id;
    assert!(
        owner
            .click_model(document, [[1., 1., 10.], [0., 0., -1.]])
            .unwrap()
    );
    assert!(
        matches!(owner.take_command_notice(), Some(crate::DesktopCommandNotice::SelectionFailed { cause }) if cause.contains("invertible"))
    );
}

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

#[test]
fn browser_selection_is_separate_from_editing_and_toggles_without_history() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let history = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    assert!(
        owner
            .select_browser_sketch_from_gui(document, sketch, false)
            .unwrap()
    );
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches,
        [sketch]
    );
    assert!(owner.active_sketch_edit().is_none());
    assert!(
        owner
            .select_browser_sketch_from_gui(document, sketch, true)
            .unwrap()
    );
    assert!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches
            .is_empty()
    );
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
fn browser_selection_refuses_stale_rows_and_is_scoped_to_the_active_document() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    owner
        .select_browser_sketch_from_gui(document, sketch, false)
        .unwrap();
    assert!(
        owner
            .select_browser_sketch_from_gui(document, DocumentSketchId(999), false)
            .is_err()
    );
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches,
        [sketch]
    );
    let second = create(&mut owner, &awake);
    assert!(
        owner
            .select_browser_sketch_from_gui(document, sketch, false)
            .is_err()
    );
    assert!(
        owner
            .gui_browser_snapshot(second)
            .unwrap()
            .selected_sketches
            .is_empty()
    );
}
#[test]
fn browser_selection_does_not_take_placement_tool_clicks() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner
        .start_sketch_placement(document, sketch, crate::SketchPlacementTool::Rectangle)
        .unwrap();
    assert!(
        !owner
            .select_browser_sketch_from_gui(document, sketch, false)
            .unwrap()
    );
    assert!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches
            .is_empty()
    );
}
#[test]
fn browser_selection_is_retired_when_its_document_closes() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    owner
        .select_browser_sketch_from_gui(document, sketch, false)
        .unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    assert!(owner.selected_browser_sketches(document).is_empty());
    owner.poll(1).unwrap();
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        0
    );
}

#[test]
fn browser_selection_snapshot_tracks_undo_and_redo_without_reusing_row_positions() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    owner
        .select_browser_sketch_from_gui(document, sketch, false)
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches
            .is_empty()
    );
    assert!(
        owner
            .select_browser_sketch_from_gui(document, sketch, false)
            .is_err()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(
        owner
            .gui_browser_snapshot(document)
            .unwrap()
            .selected_sketches,
        [sketch]
    );
}

#[test]
fn canonical_selection_query_reads_the_sketch_selected_in_the_browser() {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(document, &sketch_edit::request(SketchPlane::xy()))
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    owner
        .select_browser_sketch_from_gui(document, sketch, false)
        .unwrap();
    let selected = Client::new(owner.document_host())
        .model()
        .selection()
        .unwrap();
    assert_eq!(selected.count, 1);
    assert_eq!(selected.kinds, Some(vec![8]));
    assert_eq!(selected.refs, Some(vec![String::new()]));
    let query = oblikovati_automation::FileQueryHost::new(owner.documents());
    assert_eq!(Client::new(query).model().selection().unwrap(), selected);
    assert!(owner.active_sketch_edit().is_none());
}

#[path = "browser/selection_transport.rs"]
mod selection_transport;
