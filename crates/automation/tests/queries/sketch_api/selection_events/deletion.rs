// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_automation::{SelectionDelete, SelectionDeletionDispatch};
fn deletion_input(click: &SelectionClick) -> SelectionDelete {
    SelectionDelete {
        document: click.document,
        active_sketch: Some(click.sketch),
        tool_active: false,
    }
}
fn selected_click(click: &SelectionClick) -> SelectionSet<SketchPick> {
    let mut picks = SelectionSet::default();
    picks.add(click.picked.unwrap());
    picks
}
fn assert_retirement(receipt: SelectionDeletionDispatch, picks: &SelectionSet<SketchPick>) {
    assert!(receipt.delivery_error.is_none());
    assert_eq!(receipt.retired.entities.len(), 1);
    assert_eq!(receipt.retired.constraints.len(), 1);
    assert!(picks.items().is_empty());
}
#[test]
fn file_editor_delete_notifies_zero_after_accepted_history_for_both_owners() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_file_delete(kind);
    }
}
fn verify_file_delete(kind: DocumentType) {
    let (mut files, document, sketch, mut source) = file_click_owner(kind);
    let click = checked_click(&mut files, document, sketch);
    let mut picks = selected_click(&click);
    let mut relay = EventRelay::new(RecordingEventSink::default());
    let receipt = FileHost::new(&mut files, &mut source, "test")
        .selection_delete(&deletion_input(&click), &mut picks, &mut relay)
        .unwrap();
    assert_retirement(receipt, &picks);
    assert_eq!(relay_counts(relay.into_sink()), [0]);
    assert_committed_delete(files.workspace(), document, sketch);
}
#[test]
fn delete_notification_failure_keeps_the_accepted_geometry_edit_and_undo_step() {
    let (mut files, document, sketch, mut source) = file_click_owner(DOCUMENT_PART);
    let click = checked_click(&mut files, document, sketch);
    let mut picks = selected_click(&click);
    let mut relay = refusing_relay();
    let receipt = FileHost::new(&mut files, &mut source, "test")
        .selection_delete(&deletion_input(&click), &mut picks, &mut relay)
        .unwrap();
    assert!(receipt.delivery_error.is_some());
    assert_eq!(receipt.retired.entities.len(), 1);
    assert!(picks.items().is_empty());
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(owner_circle_count(files.workspace(), document, sketch), 1);
    relay.selection_changed(0).unwrap();
    assert_eq!(relay_counts(relay.into_sink()), [0]);
}
#[test]
fn no_editor_active_tool_or_empty_picks_do_not_send_delete_notifications() {
    let (mut files, document, sketch, mut source) = file_click_owner(DOCUMENT_PART);
    let click = checked_click(&mut files, document, sketch);
    let before = owner_revision(&files, document);
    let mut picks = selected_click(&click);
    let mut relay = EventRelay::new(RecordingEventSink::default());
    {
        let host = FileHost::new(&mut files, &mut source, "test");
        verify_delete_noops(&host, &click, &mut picks, &mut relay);
    }
    assert_eq!(relay.into_sink().attempts, 0);
    assert_eq!(owner_revision(&files, document), before);
}
fn assert_no_delete(receipt: SelectionDeletionDispatch) {
    assert!(receipt.retired.entities.is_empty());
    assert!(receipt.retired.constraints.is_empty());
    assert!(receipt.delivery_error.is_none());
}
#[test]
fn session_host_delete_publishes_the_same_zero_event_and_one_undo_step() {
    let (mut workspace, click) = workspace_click_owner();
    let mut picks = selected_click(&click);
    let mut relay = EventRelay::new(RecordingEventSink::default());
    {
        let host = SessionHost::new(&mut workspace);
        assert_retirement(
            host.selection_delete(&deletion_input(&click), &mut picks, &mut relay)
                .unwrap(),
            &picks,
        );
    }
    assert_committed_delete(&workspace, click.document, click.sketch);
    assert_eq!(relay_counts(relay.into_sink()), [0]);
}
#[test]
fn vetoed_session_delete_retains_picks_and_sends_no_selection_event() {
    let (mut workspace, input, picked) = veto_selection_owner();
    let before = workspace
        .session(input.document)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    let mut picks = SelectionSet::default();
    picks.add(picked);
    let mut relay = EventRelay::new(RecordingEventSink::default());
    assert!(
        SessionHost::new(&mut workspace)
            .selection_delete(&input, &mut picks, &mut relay)
            .is_err()
    );
    assert_eq!(picks.items(), &[picked]);
    assert_eq!(relay.into_sink().attempts, 0);
    assert_no_committed_delete(&workspace, input.document, &before);
}

fn owner_circle_count(workspace: &Workspace, document: u64, sketch: DocumentSketchId) -> usize {
    let snapshot = workspace.session(document).unwrap().snapshot();
    snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .circles()
        .len()
}
fn assert_committed_delete(workspace: &Workspace, document: u64, sketch: DocumentSketchId) {
    let owner = workspace.session(document).unwrap();
    assert_eq!(owner.undo_labels().last(), Some(&"Delete Sketch Entities"));
    assert_eq!(owner_circle_count(workspace, document, sketch), 0);
}
fn refusing_relay() -> EventRelay<RecordingEventSink> {
    EventRelay::new(RecordingEventSink {
        remaining_refusals: 1,
        ..Default::default()
    })
}
fn verify_delete_noops(
    host: &FileHost<'_, QueryObserver, QueryObserver, DiscoveryFiles>,
    click: &SelectionClick,
    picks: &mut SelectionSet<SketchPick>,
    relay: &mut EventRelay<RecordingEventSink>,
) {
    for (active_sketch, tool_active) in [(None, false), (Some(click.sketch), true)] {
        let input = SelectionDelete {
            document: click.document,
            active_sketch,
            tool_active,
        };
        assert_no_delete(host.selection_delete(&input, picks, relay).unwrap());
        assert_eq!(picks.items(), &[click.picked.unwrap()]);
    }
    picks.clear();
    assert_no_delete(
        host.selection_delete(&deletion_input(click), picks, relay)
            .unwrap(),
    );
}
fn veto_selection_owner() -> (Workspace, SelectionDelete, SketchPick) {
    let (original, click) = workspace_click_owner();
    let snapshot = original.session(click.document).unwrap().snapshot().clone();
    let (mut workspace, document) = crate::sketch_updates::veto_workspace(snapshot);
    let picked = workspace
        .commands(document)
        .unwrap()
        .sketch_pick(click.sketch, click.picked.unwrap().target())
        .unwrap();
    let input = SelectionDelete {
        document,
        ..deletion_input(&click)
    };
    (workspace, input, picked)
}
fn assert_no_committed_delete(
    workspace: &Workspace,
    document: u64,
    before: &oblikovati_model::document_sketch::DocumentSketches,
) {
    let owner = workspace.session(document).unwrap();
    assert!(before.same_recipe(&owner.snapshot().sketches));
    assert!(owner.undo_labels().is_empty());
}
