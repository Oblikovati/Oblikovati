// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[path = "selection_events/deletion.rs"]
mod deletion;
use oblikovati_automation::{
    EventRelay, EventSink, SelectionClick, SelectionDispatch, SessionHost,
};
use oblikovati_session::selection::{
    SelectionClickMode::{Replace, Toggle},
    SelectionSet, SketchPick, SketchPickTarget,
};
#[path = "../../support/selection_sink.rs"]
mod selection_sink;
use selection_sink::RecordingEventSink;

fn file_click_owner(kind: DocumentType) -> (DiscoveryWorkspace, u64, DocumentSketchId, QueryGuids) {
    let (mut files, document, source) = configured_files(kind, Default::default());
    let sketch = insert_circle(&mut files, document);
    (files, document, sketch, source)
}
fn checked_click(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) -> SelectionClick {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let entity = circle_identity(snapshot, sketch);
    let picked = files
        .commands(document)
        .unwrap()
        .sketch_pick(sketch, SketchPickTarget::Entity(entity))
        .unwrap();
    SelectionClick {
        document,
        sketch,
        picked: Some(picked),
        mode: Replace,
        tool_active: false,
    }
}
fn assert_accepted(receipt: SelectionDispatch, count: Option<usize>) {
    assert_eq!(receipt.count, count);
    assert!(receipt.delivery_error.is_none());
}
fn relay_counts(sink: RecordingEventSink) -> Vec<i64> {
    sink.payloads
        .iter()
        .map(|payload| {
            let event: SelectionChangedEvent = serde_json::from_slice(payload).unwrap();
            assert_eq!(event.type_, EVENT_SELECTION_CHANGED);
            event.count
        })
        .collect()
}
#[test]
fn file_host_relays_accepted_clicks_and_empty_changes_for_part_and_assembly() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        verify_file_clicks(kind);
    }
}
fn verify_file_clicks(kind: DocumentType) {
    let (mut files, document, sketch, mut source) = file_click_owner(kind);
    let revision = owner_revision(&files, document);
    let click = checked_click(&mut files, document, sketch);
    let mut picks = SelectionSet::default();
    let mut relay = EventRelay::new(RecordingEventSink::default());
    {
        let host = FileHost::new(&mut files, &mut source, "test");
        verify_repeated_click(&host, &click, &mut picks, &mut relay);
        verify_empty_clicks(&host, click, &mut picks, &mut relay);
    }
    assert_eq!(relay_counts(relay.into_sink()), [1, 1, 0]);
    assert_eq!(owner_revision(&files, document), revision);
}
fn verify_empty_clicks(
    host: &FileHost<'_, QueryObserver, QueryObserver, DiscoveryFiles>,
    mut click: SelectionClick,
    picks: &mut SelectionSet<SketchPick>,
    relay: &mut EventRelay<RecordingEventSink>,
) {
    click.tool_active = true;
    assert_accepted(host.selection_click(&click, picks, relay).unwrap(), None);
    click.tool_active = false;
    click.picked = None;
    click.mode = Toggle;
    assert_accepted(host.selection_click(&click, picks, relay).unwrap(), None);
    click.mode = Replace;
    assert_accepted(host.selection_click(&click, picks, relay).unwrap(), Some(0));
    assert_accepted(host.selection_click(&click, picks, relay).unwrap(), None);
}
#[test]
fn delivery_failure_reports_accepted_selection_and_retry_does_not_repeat_the_click() {
    let (mut files, document, sketch, mut source) = file_click_owner(DOCUMENT_PART);
    let click = checked_click(&mut files, document, sketch);
    let revision = owner_revision(&files, document);
    let mut picks = SelectionSet::default();
    let mut relay = EventRelay::new(RecordingEventSink {
        remaining_refusals: 1,
        ..Default::default()
    });
    let receipt = FileHost::new(&mut files, &mut source, "test")
        .selection_click(&click, &mut picks, &mut relay)
        .unwrap();
    assert_delivery_failure(receipt, &picks, &click);
    relay.selection_changed(1).unwrap();
    assert_eq!(relay_counts(relay.into_sink()), [1]);
    assert_eq!(owner_revision(&files, document), revision);
}
#[test]
fn invalid_document_refuses_before_selection_or_notification_changes() {
    let (mut files, document, sketch, mut source) = file_click_owner(DOCUMENT_PART);
    let mut click = checked_click(&mut files, document, sketch);
    let mut picks = SelectionSet::default();
    picks.add(click.picked.unwrap());
    let before = picks.items().to_vec();
    let mut relay = EventRelay::new(RecordingEventSink::default());
    click.document = u64::MAX;
    assert!(
        FileHost::new(&mut files, &mut source, "test")
            .selection_click(&click, &mut picks, &mut relay)
            .is_err()
    );
    assert_eq!(picks.items(), before);
    assert_eq!(relay.into_sink().attempts, 0);
}
#[test]
fn stale_candidate_refusal_sends_no_event_and_does_not_replace_current_picks() {
    let (mut files, document, sketch, mut source) = file_click_owner(DOCUMENT_PART);
    let stale = checked_click(&mut files, document, sketch);
    change_owner_width(&mut files, document);
    let fresh = checked_click(&mut files, document, sketch);
    let mut picks = SelectionSet::default();
    picks.add(fresh.picked.unwrap());
    let mut relay = EventRelay::new(RecordingEventSink::default());
    assert!(
        FileHost::new(&mut files, &mut source, "test")
            .selection_click(&stale, &mut picks, &mut relay)
            .is_err()
    );
    assert_eq!(picks.items(), &[fresh.picked.unwrap()]);
    assert_eq!(relay.into_sink().attempts, 0);
}

fn circle_identity(snapshot: &DocumentSnapshot, sketch: DocumentSketchId) -> SketchEntityId {
    snapshot.sketches.get(sketch).unwrap().geometry().circles()[0].id
}
fn verify_repeated_click(
    host: &FileHost<'_, QueryObserver, QueryObserver, DiscoveryFiles>,
    click: &SelectionClick,
    picks: &mut SelectionSet<SketchPick>,
    relay: &mut EventRelay<RecordingEventSink>,
) {
    assert_accepted(host.selection_click(click, picks, relay).unwrap(), Some(1));
    assert_accepted(host.selection_click(click, picks, relay).unwrap(), Some(1));
}
fn assert_delivery_failure(
    receipt: SelectionDispatch,
    picks: &SelectionSet<SketchPick>,
    click: &SelectionClick,
) {
    assert_eq!(receipt.count, Some(1));
    assert!(
        receipt
            .delivery_error
            .unwrap()
            .to_string()
            .contains("delivery failed")
    );
    assert_eq!(picks.items(), &[click.picked.unwrap()]);
}
fn change_owner_width(files: &mut DiscoveryWorkspace, document: u64) {
    files
        .commands(document)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
}
fn workspace_click_owner() -> (Workspace, SelectionClick) {
    let (mut workspace, document, sketch) = crate::sketch_updates::circle_workspace();
    let entity = circle_identity(workspace.session(document).unwrap().snapshot(), sketch);
    let picked = workspace
        .commands(document)
        .unwrap()
        .sketch_pick(sketch, SketchPickTarget::Entity(entity))
        .unwrap();
    let click = SelectionClick {
        document,
        sketch,
        picked: Some(picked),
        mode: Replace,
        tool_active: false,
    };
    (workspace, click)
}
#[test]
fn session_host_notifies_without_changing_document_focus_or_history() {
    let (mut workspace, click) = workspace_click_owner();
    let other_units = UnitsOfMeasure::default();
    let other = register(&mut workspace, other_units, &mut QueryGuids(20));
    let generation = workspace.session(click.document).unwrap().generation();
    let mut picks = SelectionSet::default();
    let mut relay = EventRelay::new(RecordingEventSink::default());
    {
        let host = SessionHost::new(&mut workspace);
        assert_accepted(
            host.selection_click(&click, &mut picks, &mut relay)
                .unwrap(),
            Some(1),
        );
    }
    assert_workspace_position(&workspace, click.document, generation, other);
    assert_eq!(relay_counts(relay.into_sink()), [1]);
}
struct ReenteringEventSink<'a, 'w> {
    host: &'a SessionHost<'w, QueryObserver, QueryObserver>,
    failure: Option<String>,
}
impl EventSink for ReenteringEventSink<'_, '_> {
    fn send_event(&mut self, _: &[u8]) -> Result<(), String> {
        self.failure = self
            .host
            .call(METHOD_PARAMETERS_GET_SETTINGS, b"{}")
            .err()
            .map(|error| error.to_string());
        Ok(())
    }
}
#[test]
fn synchronous_event_callback_reentry_is_refused_and_host_recovers_after_delivery() {
    let (mut workspace, click) = workspace_click_owner();
    let host = SessionHost::new(&mut workspace);
    let mut picks = SelectionSet::default();
    let mut relay = EventRelay::new(ReenteringEventSink {
        host: &host,
        failure: None,
    });
    assert_accepted(
        host.selection_click(&click, &mut picks, &mut relay)
            .unwrap(),
        Some(1),
    );
    let failure = relay.into_sink().failure.unwrap();
    assert!(failure.contains("reentry is refused"));
    assert!(host.call(METHOD_PARAMETERS_GET_SETTINGS, b"{}").is_ok());
    assert_eq!(picks.items(), &[click.picked.unwrap()]);
}

fn assert_workspace_position(workspace: &Workspace, document: u64, generation: u64, active: u64) {
    assert_eq!(workspace.active_document(), Some(active));
    assert_eq!(
        workspace.session(document).unwrap().generation(),
        generation
    );
}
